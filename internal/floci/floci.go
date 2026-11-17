// Package floci manages the one Floci container that tflings uses as its local
// AWS. It talks to Docker through the docker CLI, so it works with whatever
// Docker the learner has (Docker Desktop, OrbStack, Colima, Lima, Linux).
package floci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Image is the Floci image, pinned by digest (spike 05). Bumping it is a
// deliberate change: run the whole exercise suite first.
const Image = "floci/floci@sha256:5ee85db604c4aa74ac76e07ea5cc83086f37bb472bf3dbb97e34019e48efae9c"

// Version is the Floci release the digest above is.
const Version = "2.4.0"

// Config says which container to manage. The defaults are what tflings uses;
// tests use another name, port and volume so they never touch the real one.
type Config struct {
	Name   string
	Port   int // published on 127.0.0.1 only
	Volume string
	Image  string
}

// DefaultConfig is the container tflings runs for learners.
func DefaultConfig() Config {
	return Config{Name: "tflings-floci", Port: 4577, Volume: "tflings-floci-data", Image: Image}
}

// Docker runs one docker CLI command.
type Docker interface {
	Run(ctx context.Context, args ...string) (stdout, stderr string, err error)
}

// CLI is the real docker command line.
type CLI struct{ Binary string }

// Run runs the docker CLI with args and returns its stdout and stderr.
func (c CLI) Run(ctx context.Context, args ...string) (string, string, error) {
	bin := c.Binary
	if bin == "" {
		bin = "docker"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

// Errors the manager can name, so that callers print a fix and not Docker's text.
var (
	ErrNoDocker   = errors.New("docker is not installed (no `docker` command found)")
	ErrDockerDown = errors.New("docker is installed but its daemon is not reachable")
	ErrPortInUse  = errors.New("the Floci port is already in use by another program")
	ErrNotHealthy = errors.New("floci did not become healthy in time")
	ErrNameInUse  = errors.New("a container with that name exists but is not tflings's Floci")
)

// Manager manages the Floci container.
type Manager struct {
	Cfg    Config
	Docker Docker
	HTTP   *http.Client
	// Progress receives short messages about slow steps (pulling the image).
	Progress func(string)
	// WaitHealthy is how long Up waits for Floci to answer; default 30 s.
	WaitHealthy time.Duration
}

// New returns a Manager for the default container using the docker CLI.
func New() *Manager {
	return &Manager{Cfg: DefaultConfig(), Docker: CLI{}, HTTP: &http.Client{Timeout: 2 * time.Second}}
}

func (m *Manager) progress(s string) {
	if m.Progress != nil {
		m.Progress(s)
	}
}

// URL is where the container answers on this machine.
func (m *Manager) URL() string { return fmt.Sprintf("http://127.0.0.1:%d", m.Cfg.Port) }

// Healthy asks Floci itself. It is quick and needs no Docker, so `list` and
// `run` can use it to decide which exercises can run.
func (m *Manager) Healthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL()+"/_floci/health", nil)
	if err != nil {
		return false
	}
	resp, err := m.client().Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK
}

func (m *Manager) client() *http.Client {
	if m.HTTP != nil {
		return m.HTTP
	}
	return &http.Client{Timeout: 2 * time.Second}
}

// State is what Docker says about the container.
type State struct {
	Exists  bool
	Running bool
	// Health is Docker's health status: "healthy", "unhealthy", "starting" or "".
	Health string
	// Image is the image reference the container was created from.
	Image    string
	HostPort string
}

// Current reports whether the container exists and in what state.
func (m *Manager) Current(ctx context.Context) (State, error) {
	out, stderr, err := m.Docker.Run(ctx, "inspect", "--type", "container", m.Cfg.Name)
	if err != nil {
		if isNoSuchObject(stderr) {
			return State{}, nil
		}
		return State{}, classify(stderr, err)
	}
	var infos []struct {
		State struct {
			Running bool
			Health  *struct{ Status string }
		}
		Config     struct{ Image string }
		HostConfig struct {
			PortBindings map[string][]struct{ HostPort string }
		}
	}
	if err := json.Unmarshal([]byte(out), &infos); err != nil {
		return State{}, fmt.Errorf("cannot read `docker inspect` output: %w", err)
	}
	if len(infos) != 1 {
		return State{}, fmt.Errorf("cannot read `docker inspect` output: expected one container, got %d", len(infos))
	}
	i := infos[0]
	s := State{Exists: true, Running: i.State.Running, Image: i.Config.Image}
	if i.State.Health != nil {
		s.Health = i.State.Health.Status
	}
	if b := i.HostConfig.PortBindings["4566/tcp"]; len(b) > 0 {
		s.HostPort = b[0].HostPort
	}
	return s, nil
}

func isNoSuchObject(stderr string) bool {
	return strings.Contains(stderr, "No such object") || strings.Contains(stderr, "No such container")
}

// classify turns Docker's failure text into one of the named errors.
func classify(stderr string, err error) error {
	var notFound *exec.Error
	switch {
	case errors.As(err, &notFound), errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		return ErrNoDocker
	case strings.Contains(stderr, "Cannot connect to the Docker daemon"),
		strings.Contains(stderr, "failed to connect to the docker API"),
		strings.Contains(stderr, "error during connect"),
		strings.Contains(stderr, "Is the docker daemon running"):
		return ErrDockerDown
	case strings.Contains(stderr, "port is already allocated"),
		strings.Contains(stderr, "address already in use"):
		return ErrPortInUse
	}
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		return err
	}
	return fmt.Errorf("docker: %s", firstLine(msg))
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// CheckDocker reports whether the docker daemon answers, with its version.
func (m *Manager) CheckDocker(ctx context.Context) (string, error) {
	out, stderr, err := m.Docker.Run(ctx, "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "", classify(stderr, err)
	}
	return strings.TrimSpace(out), nil
}

// runArgs is the `docker run` command of the container: published on
// 127.0.0.1 only, data on a named volume at /app/data (any other mount leaks an
// anonymous volume per run), no extra capabilities beyond what the entrypoint
// needs, a read-only root and no Docker socket (spike 05).
func (m *Manager) runArgs() []string {
	return []string{
		"run", "-d", "--name", m.Cfg.Name,
		"--label", "io.tflings.managed=true",
		"-p", fmt.Sprintf("127.0.0.1:%d:4566", m.Cfg.Port),
		"-e", "FLOCI_STORAGE_MODE=persistent",
		"-e", "FLOCI_STORAGE_PERSISTENT_PATH=/app/data",
		"-v", m.Cfg.Volume + ":/app/data",
		"--cap-drop", "ALL", "--cap-add", "SYS_CHROOT", "--cap-add", "SETUID", "--cap-add", "SETGID",
		"--security-opt", "no-new-privileges",
		"--read-only", "--tmpfs", "/tmp",
		m.Cfg.Image,
	}
}

// Up makes sure the container runs the pinned image and answers. It reuses a
// running one, starts a stopped one, and replaces one made from another image
// (the data volume is kept). It pulls the image if it is not there.
func (m *Manager) Up(ctx context.Context) error {
	st, err := m.Current(ctx)
	if err != nil {
		return err
	}
	if st.Exists && st.Image != m.Cfg.Image {
		m.progress("replacing the Floci container: it was made from another image")
		if err := m.remove(ctx); err != nil {
			return err
		}
		st = State{}
	}
	switch {
	case st.Exists && st.Running:
		// nothing to do
	case st.Exists:
		if _, stderr, err := m.Docker.Run(ctx, "start", m.Cfg.Name); err != nil {
			return classify(stderr, err)
		}
	default:
		if err := m.ensureImage(ctx); err != nil {
			return err
		}
		if _, stderr, err := m.Docker.Run(ctx, m.runArgs()...); err != nil {
			return classify(stderr, err)
		}
	}
	return m.waitHealthy(ctx)
}

func (m *Manager) ensureImage(ctx context.Context) error {
	if _, _, err := m.Docker.Run(ctx, "image", "inspect", m.Cfg.Image); err == nil {
		return nil
	}
	m.progress("pulling Floci " + Version + " (about 370 MB, once)")
	if _, stderr, err := m.Docker.Run(ctx, "pull", "--quiet", m.Cfg.Image); err != nil {
		return classify(stderr, err)
	}
	return nil
}

func (m *Manager) waitHealthy(ctx context.Context) error {
	wait := m.WaitHealthy
	if wait <= 0 {
		wait = 30 * time.Second
	}
	deadline := time.Now().Add(wait)
	for {
		if m.Healthy(ctx) {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrNotHealthy
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (m *Manager) remove(ctx context.Context) error {
	// -f stops it, -v removes anonymous volumes (there should be none); the
	// named data volume stays.
	if _, stderr, err := m.Docker.Run(ctx, "rm", "-fv", m.Cfg.Name); err != nil && !isNoSuchObject(stderr) {
		return classify(stderr, err)
	}
	return nil
}

// Down stops and removes the container. The data volume stays unless wipe is
// true, which also deletes everything every exercise has built.
func (m *Manager) Down(ctx context.Context, wipe bool) error {
	if err := m.remove(ctx); err != nil {
		return err
	}
	if wipe {
		if _, stderr, err := m.Docker.Run(ctx, "volume", "rm", m.Cfg.Volume); err != nil && !strings.Contains(stderr, "No such volume") {
			return classify(stderr, err)
		}
	}
	return nil
}

// Reset empties every account in Floci at once (there is no per-account reset,
// spike 02). It leaves the container running.
func (m *Manager) Reset(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.URL()+"/_floci/state/reset", nil)
	if err != nil {
		return err
	}
	resp, err := m.client().Do(req)
	if err != nil {
		return fmt.Errorf("floci is not answering: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("floci refused the reset: HTTP %d", resp.StatusCode)
	}
	return nil
}
