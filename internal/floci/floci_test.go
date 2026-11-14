package floci

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeDocker answers docker commands from a script and records them.
type fakeDocker struct {
	calls []string
	// reply maps the first two words of a command to its result.
	reply map[string]result
}

type result struct {
	out, err string
	fail     error
}

func (f *fakeDocker) Run(_ context.Context, args ...string) (string, string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	key := args[0]
	if len(args) > 1 {
		key += " " + args[1]
	}
	for _, k := range []string{key, args[0]} {
		if r, ok := f.reply[k]; ok {
			return r.out, r.err, r.fail
		}
	}
	return "", "", nil
}

func (f *fakeDocker) called(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

const noSuch = "Error response from daemon: No such object: tflings-floci"

func inspectJSON(running bool, health, image, port string) string {
	h := ""
	if health != "" {
		h = fmt.Sprintf(`,"Health":{"Status":%q}`, health)
	}
	return fmt.Sprintf(`[{"State":{"Running":%v%s},"Config":{"Image":%q},"HostConfig":{"PortBindings":{"4566/tcp":[{"HostPort":%q}]}}}]`, running, h, image, port)
}

// healthyServer plays Floci's health endpoint and returns a Manager that talks to it.
func healthyServer(t *testing.T, d Docker) (*Manager, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/_floci/health":
			w.WriteHeader(200)
		case "/_floci/state/reset":
			if r.Method != http.MethodPost {
				w.WriteHeader(405)
				return
			}
			w.WriteHeader(200)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(srv.Close)
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	p, _ := strconv.Atoi(port)
	cfg := DefaultConfig()
	cfg.Port = p
	return &Manager{Cfg: cfg, Docker: d, WaitHealthy: 2 * time.Second}, srv
}

func TestUpCreatesTheContainerWhenMissing(t *testing.T) {
	d := &fakeDocker{reply: map[string]result{
		"inspect --type": {err: noSuch, fail: errors.New("exit 1")},
		"image inspect":  {fail: errors.New("not found")},
	}}
	m, _ := healthyServer(t, d)
	var msgs []string
	m.Progress = func(s string) { msgs = append(msgs, s) }
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !d.called("pull --quiet "+Image) || !d.called("run -d --name tflings-floci") {
		t.Fatalf("expected a pull and a run, got %v", d.calls)
	}
	if len(msgs) == 0 || !strings.Contains(msgs[0], "pulling") {
		t.Errorf("the slow pull should be announced, got %v", msgs)
	}
	var run string
	for _, c := range d.calls {
		if strings.HasPrefix(c, "run ") {
			run = c
		}
	}
	for _, want := range []string{
		"-p 127.0.0.1:", ":4566", "-v tflings-floci-data:/app/data", "FLOCI_STORAGE_PERSISTENT_PATH=/app/data",
		"--cap-drop ALL", "--cap-add SYS_CHROOT", "--security-opt no-new-privileges", "--read-only", Image,
	} {
		if !strings.Contains(run, want) {
			t.Errorf("the run command lacks %q:\n%s", want, run)
		}
	}
	if strings.Contains(run, "docker.sock") {
		t.Error("the Docker socket must never be mounted")
	}
}

func TestUpReusesARunningContainer(t *testing.T) {
	d := &fakeDocker{reply: map[string]result{"inspect --type": {out: inspectJSON(true, "healthy", Image, "4577")}}}
	m, _ := healthyServer(t, d)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if d.called("run ") || d.called("start ") || d.called("rm ") || d.called("pull") {
		t.Errorf("a running container must be left alone, got %v", d.calls)
	}
}

func TestUpStartsAStoppedContainer(t *testing.T) {
	d := &fakeDocker{reply: map[string]result{"inspect --type": {out: inspectJSON(false, "", Image, "4577")}}}
	m, _ := healthyServer(t, d)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !d.called("start tflings-floci") || d.called("run ") {
		t.Errorf("got %v", d.calls)
	}
}

func TestUpReplacesAContainerFromAnotherImage(t *testing.T) {
	d := &fakeDocker{reply: map[string]result{"inspect --type": {out: inspectJSON(true, "healthy", "floci/floci:2.3.0", "4577")}}}
	m, _ := healthyServer(t, d)
	if err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !d.called("rm -fv tflings-floci") || !d.called("run ") {
		t.Errorf("the old container must be replaced, got %v", d.calls)
	}
	if d.called("volume rm") {
		t.Error("replacing the container must keep the data volume")
	}
}

func TestUpTimesOutWhenFlociNeverAnswers(t *testing.T) {
	d := &fakeDocker{reply: map[string]result{"inspect --type": {out: inspectJSON(true, "starting", Image, "4577")}}}
	cfg := DefaultConfig()
	cfg.Port = 1 // nothing listens here
	m := &Manager{Cfg: cfg, Docker: d, WaitHealthy: 300 * time.Millisecond}
	if err := m.Up(context.Background()); !errors.Is(err, ErrNotHealthy) {
		t.Fatalf("got %v", err)
	}
}

func TestDockerErrorsAreNamed(t *testing.T) {
	cases := map[string]error{
		"Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?": ErrDockerDown,
		"failed to connect to the docker API at unix:///x.sock; check if the path is correct":               ErrDockerDown,
		"Bind for 127.0.0.1:4577 failed: port is already allocated":                                         ErrPortInUse,
	}
	for stderr, want := range cases {
		d := &fakeDocker{reply: map[string]result{"inspect --type": {err: stderr, fail: errors.New("exit 1")}}}
		m, _ := healthyServer(t, d)
		if _, err := m.Current(context.Background()); !errors.Is(err, want) && want != ErrPortInUse {
			t.Errorf("%q: got %v, want %v", stderr, err, want)
		}
	}
	if got := classify("Bind for 127.0.0.1:4577 failed: port is already allocated", errors.New("x")); !errors.Is(got, ErrPortInUse) {
		t.Errorf("got %v", got)
	}
	if got := classify("", &exec.Error{Name: "docker", Err: exec.ErrNotFound}); !errors.Is(got, ErrNoDocker) {
		t.Errorf("a missing docker command must be ErrNoDocker, got %v", got)
	}
}

func TestDownKeepsOrWipesTheVolume(t *testing.T) {
	d := &fakeDocker{}
	m, _ := healthyServer(t, d)
	if err := m.Down(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if d.called("volume rm") {
		t.Error("down must keep the data volume")
	}
	if err := m.Down(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if !d.called("volume rm tflings-floci-data") {
		t.Errorf("down --wipe must remove the volume, got %v", d.calls)
	}
}

func TestHealthyAndReset(t *testing.T) {
	m, srv := healthyServer(t, &fakeDocker{})
	if !m.Healthy(context.Background()) {
		t.Error("the server answers 200, so Floci is healthy")
	}
	if err := m.Reset(context.Background()); err != nil {
		t.Error(err)
	}
	srv.Close()
	if m.Healthy(context.Background()) {
		t.Error("a closed server is not healthy")
	}
	if err := m.Reset(context.Background()); err == nil {
		t.Error("reset must fail when Floci does not answer")
	}
}

// The real thing: a container of its own (another name, port and volume), so
// the one a learner uses is never touched. Skipped without Docker.
func TestRealDocker(t *testing.T) {
	if os.Getenv("TFLINGS_SKIP_DOCKER") != "" {
		t.Skip("TFLINGS_SKIP_DOCKER is set")
	}
	cfg := Config{Name: "tflings-floci-test", Port: 4579, Volume: "tflings-floci-test-data", Image: Image}
	m := &Manager{Cfg: cfg, Docker: CLI{}, WaitHealthy: 40 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, err := m.CheckDocker(ctx); err != nil {
		t.Skipf("no usable Docker: %v", err)
	}
	_ = m.Down(ctx, true)
	defer func() { _ = m.Down(context.Background(), true) }()

	if err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	st, err := m.Current(ctx)
	if err != nil || !st.Running || st.Image != Image || st.HostPort != "4579" {
		t.Fatalf("unexpected state %+v, %v", st, err)
	}
	if err := m.Up(ctx); err != nil { // idempotent
		t.Fatal(err)
	}
	if err := m.Reset(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Down(ctx, false); err != nil {
		t.Fatal(err)
	}
	if st, _ := m.Current(ctx); st.Exists {
		t.Error("down must remove the container")
	}
	if m.Healthy(ctx) {
		t.Error("nothing should answer after down")
	}
	// The data volume survived `down`, and a new container uses it.
	if err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
}
