package exercises

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/cleonte/lingscore/progress"
	"github.com/cleonte/tflings/internal/runner"
)

// ProgressFile is where progress is kept that cannot live in an exercise file.
// A plan, apply, cli or quiz exercise is done when its check has passed and its
// `I AM NOT DONE` line is gone; a fix exercise has no file to edit, so its
// completion is recorded here, next to info.toml. It also holds the reset
// counter of every exercise, which, with the exercise name, decides the fake
// AWS account the exercise runs in (a reset gets a fresh, empty account).
const ProgressFile = ".tflings.json"

// The file is kept by lingscore's progress package: the done passes, the hashes
// of the exercise files and the exercises `update` added are its own keys. What
// is tflings' own lives under two more keys, which the package keeps through
// every write:
//
//   - "resets" maps the name of an exercise to how many times it was reset.
//   - "salt" is a random string given to the exercises folder by `init`. It
//     names the folder's run folders and goes into its fake AWS accounts, so two
//     exercises folders on one machine (which share <home> and Floci) never share
//     state, and a reset in one does not touch the other. A folder made before
//     this existed has none, and keeps the names it always had.
const (
	keyResets = "resets"
	keySalt   = "salt"
)

func store(dir string) progress.Store { return progress.Store{Dir: dir, File: ProgressFile} }

func readProgressIn(dir string) progress.Data { return store(dir).Read() }

func passed(name string) bool { return store(".").Passed(name) }

// markPassed records that the check of an exercise passed, in the progress
// file of the current folder.
func markPassed(name string) error { return markPassedIn(".", name) }

// markPassedIn records the pass in the workspace in dir: the same file that
// ResetCount and RecordReset use.
func markPassedIn(dir, name string) error { return store(dir).RecordPass(name) }

// RecordPass records that the check of the exercise passed, as running it does.
func RecordPass(name string) error { return markPassed(name) }

// ResetCount is how many times the exercise in the workspace in dir was reset.
func ResetCount(dir, name string) int {
	var resets map[string]int
	readProgressIn(dir).Get(keyResets, &resets)
	return resets[name]
}

// RecordReset counts a reset of the exercise in the workspace in dir and
// forgets that it was done: it has to pass again, in a fresh account.
func RecordReset(dir, name string) error {
	return store(dir).Update(func(d *progress.Data) {
		var resets map[string]int
		d.Get(keyResets, &resets)
		if resets == nil {
			resets = map[string]int{}
		}
		resets[name]++
		_ = d.Set(keyResets, resets)
		delete(d.Done, name)
	})
}

// Originals returns the recorded hashes of the exercise files written into the
// workspace in dir.
func Originals(dir string) map[string]string { return store(dir).Originals() }

// RecordOriginals remembers the hashes of exercise files that tflings has just
// written into the workspace in dir, and which exercises `update` has just added.
func RecordOriginals(dir string, hashes map[string]string, added []string) error {
	return store(dir).RecordOriginals(hashes, added)
}

// IsNew reports whether `update` added the exercise to this workspace.
func IsNew(name string) bool { return IsNewIn(".", name) }

// IsNewIn is IsNew for the workspace in dir.
func IsNewIn(dir, name string) bool { return store(dir).IsNew(name) }

// ForgetPassed drops the recorded pass of an exercise in the workspace in dir
// (its file was reset, so it has to pass again).
func ForgetPassed(dir, name string) error { return store(dir).ForgetPass(name) }

// EnsureSalt gives the workspace in dir a random salt if it has none (see the
// "salt" key above). `init` calls it; nothing else does, so a folder made by an
// older tflings keeps the run folders and accounts it already has.
func EnsureSalt(dir string) error {
	return store(dir).Update(func(d *progress.Data) {
		var salt string
		if d.Get(keySalt, &salt) && salt != "" {
			return
		}
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return
		}
		_ = d.Set(keySalt, hex.EncodeToString(b))
	})
}

// Salt is the salt of the workspace in dir, empty for a folder without one.
func Salt(dir string) string {
	var salt string
	readProgressIn(dir).Get(keySalt, &salt)
	return salt
}

// runKey is the name of an exercise's run folder (and lock) under <home>/runs:
// the exercise name, with the folder's salt in front when it has one.
func runKey(dir, name string) string {
	if s := Salt(dir); s != "" {
		return s + "-" + name
	}
	return name
}

// AccountFor is the fake AWS account of an exercise in the workspace in dir, as
// it is right now: it depends on the exercise name, the folder's salt and the
// exercise's reset counter. extra is mixed in by tests.
func AccountFor(dir, name, extra string) string {
	key := name + extra
	if s := Salt(dir); s != "" {
		key += "@" + s
	}
	return runner.Account(key, ResetCount(dir, name))
}
