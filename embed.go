// Package tflings holds the exercises that ship inside the tflings binary.
//
// The CLI itself lives in internal/ and cmd/tflings. This package sits at the
// root of the module because go:embed can't reach into parent folders.
package tflings

import "embed"

// FS contains the exercise manifest and every exercise, as they are in the
// repository. `tflings init` writes it to disk (without the check.yaml files,
// which stay in the binary) and `tflings reset` restores single exercises
// from it.
//
//go:embed info.toml all:exercises
var FS embed.FS

// Solutions contains a working version of each exercise, laid out like
// exercises/ (solutions/<topic>/<name>/<file>). It is only ever read by
// `tflings solution`; `tflings init` must not write it, which is why it is
// separate from FS.
//
//go:embed all:solutions
var Solutions embed.FS
