// Copyright 2026 The Nabat Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nabat

import (
	"bytes"
	"io"
	"math"
	"os"

	"github.com/charmbracelet/colorprofile"
	"golang.org/x/term"
)

// DefaultWidth is the terminal width assumed when the underlying stream is
// not a terminal or its size cannot be determined.
const DefaultWidth = 80

// IOStreams bundles the input and output streams a CLI uses, alongside
// terminal capability detection and a uniform color/escape policy. A single
// IOStreams instance is shared across an [App] and every [Context] it produces.
//
// Out and ErrOut are wrapped at construction with a colorprofile-aware
// writer that strips ANSI escapes when the underlying stream is not a
// terminal (or when NO_COLOR is set) and preserves the original file
// descriptor for libraries that perform their own terminal detection.
//
// IOStreams is safe to construct concurrently with other operations but
// not safe for concurrent mutation via the SetXxxTTY methods.
type IOStreams struct {
	// In is the input stream. It is the user-supplied [io.Reader] (typically
	// [os.Stdin]) and is not wrapped. Treat as read-only after construction.
	In io.Reader

	// Out is the primary output stream — "the product" in POSIX terms. It
	// is wrapped at construction with a colorprofile-aware writer; do not
	// reassign after construction or the color-detection wrapping is lost.
	Out io.Writer

	// ErrOut is the diagnostics stream — errors, warnings, progress. It
	// shares the same wrapping policy as Out; do not reassign after
	// construction.
	ErrOut io.Writer

	rawIn  io.Reader
	rawOut io.Writer
	rawErr io.Writer

	stdinIsTTY  bool
	stdoutIsTTY bool
	stderrIsTTY bool

	outProfile colorprofile.Profile
	errProfile colorprofile.Profile

	stickyErr *stickyError
}

// NewSystemIO returns an IOStreams backed by [os.Stdin], [os.Stdout], and
// [os.Stderr]. Color profile and TTY status are detected against the real
// file descriptors and the process environment.
func NewSystemIO() *IOStreams {
	return NewIO(os.Stdin, os.Stdout, os.Stderr)
}

// NewIO returns an IOStreams over the supplied streams. Out and ErrOut are
// wrapped with the colorprofile policy; In is passed through unchanged.
//
// Pass *[os.File] values for production code (TTY detection works against the
// real file descriptor). For tests prefer [nabattest.NewIO], which also returns
// the underlying buffers for assertion.
func NewIO(in io.Reader, out, errOut io.Writer) *IOStreams {
	if in == nil {
		in = nopReader{}
	}
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	env := os.Environ()
	outProfile := colorprofile.Detect(out, env)
	errProfile := colorprofile.Detect(errOut, env)

	sticky := &stickyError{}

	wrappedIn := wrapReader(in)
	wrappedOut := wrapWriter(out, outProfile, sticky)
	wrappedErr := wrapWriter(errOut, errProfile, sticky)

	return &IOStreams{
		In:          wrappedIn,
		Out:         wrappedOut,
		ErrOut:      wrappedErr,
		rawIn:       in,
		rawOut:      out,
		rawErr:      errOut,
		stdinIsTTY:  isTerminal(in),
		stdoutIsTTY: isTerminal(out),
		stderrIsTTY: isTerminal(errOut),
		outProfile:  outProfile,
		errProfile:  errProfile,
		stickyErr:   sticky,
	}
}

// newTestIO returns an IOStreams whose three streams are *[bytes.Buffer] values.
// It is the internal equivalent of [nabattest.NewIO]; internal test files in
// package nabat use this to avoid the import cycle that would arise from
// importing nabattest.
func newTestIO() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return NewIO(in, out, errOut), in, out, errOut
}

// IsStdinTTY reports whether the input stream is a terminal.
func (s *IOStreams) IsStdinTTY() bool { return s.stdinIsTTY }

// IsStdoutTTY reports whether the output stream is a terminal.
func (s *IOStreams) IsStdoutTTY() bool { return s.stdoutIsTTY }

// IsStderrTTY reports whether the diagnostics stream is a terminal.
func (s *IOStreams) IsStderrTTY() bool { return s.stderrIsTTY }

// SetStdinTTY overrides the cached stdin TTY status. Tests use this to
// exercise the interactive code path against a buffer-backed stream.
func (s *IOStreams) SetStdinTTY(v bool) { s.stdinIsTTY = v }

// SetStdoutTTY overrides the cached stdout TTY status.
func (s *IOStreams) SetStdoutTTY(v bool) { s.stdoutIsTTY = v }

// SetStderrTTY overrides the cached stderr TTY status.
func (s *IOStreams) SetStderrTTY(v bool) { s.stderrIsTTY = v }

// ColorEnabled reports whether colorized output is in effect for the
// primary output stream.
func (s *IOStreams) ColorEnabled() bool {
	return s.outProfile > colorprofile.NoTTY
}

// CanPrompt reports whether interactive prompts are usable. It is true
// when both stdin and stdout are terminals.
func (s *IOStreams) CanPrompt() bool {
	return s.IsStdinTTY() && s.IsStdoutTTY()
}

// TerminalWidth returns the column width of the controlling terminal, or
// [DefaultWidth] when stdout is not a terminal or its size cannot be queried.
func (s *IOStreams) TerminalWidth() int {
	fd := fdOf(s.rawOut)
	if fd == invalidFd {
		return DefaultWidth
	}
	if fd > uintptr(math.MaxInt) {
		return DefaultWidth
	}
	w, _, err := term.GetSize(int(fd))
	if err != nil || w <= 0 {
		return DefaultWidth
	}
	return w
}

// Err returns the first write error encountered by Out or ErrOut, or nil
// if all writes have succeeded so far.
func (s *IOStreams) Err() error {
	if s.stickyErr == nil {
		return nil
	}
	return s.stickyErr.err
}

// RawIn returns the unwrapped input stream supplied to [NewIO] or [NewSystemIO].
func (s *IOStreams) RawIn() io.Reader { return s.rawIn }

// RawOut returns the unwrapped output stream supplied to [NewIO] or [NewSystemIO].
func (s *IOStreams) RawOut() io.Writer { return s.rawOut }

// RawErrOut returns the unwrapped diagnostics stream supplied to [NewIO]
// or [NewSystemIO].
func (s *IOStreams) RawErrOut() io.Writer { return s.rawErr }

type stickyError struct {
	err error
}

func (e *stickyError) record(err error) {
	if e.err == nil && err != nil {
		e.err = err
	}
}

type stickyWriter struct {
	w     io.Writer
	fd    uintptr
	state *stickyError
}

func (w *stickyWriter) Write(p []byte) (int, error) {
	if w.state.err != nil {
		return 0, w.state.err
	}
	n, err := w.w.Write(p)
	if err != nil {
		w.state.record(err)
	}
	return n, err
}

func (w *stickyWriter) Fd() uintptr { return w.fd }

func wrapWriter(w io.Writer, profile colorprofile.Profile, sticky *stickyError) io.Writer {
	cw := &colorprofile.Writer{Forward: w, Profile: profile}
	return &stickyWriter{w: cw, fd: fdOf(w), state: sticky}
}

func wrapReader(r io.Reader) io.Reader {
	if _, ok := r.(*os.File); ok {
		return r
	}
	if _, ok := r.(interface{ Fd() uintptr }); ok {
		return r
	}
	return &fdReader{Reader: r, fd: invalidFd}
}

func isTerminal(v any) bool {
	fd := fdOf(v)
	if fd == invalidFd {
		return false
	}
	if fd > uintptr(math.MaxInt) {
		return false
	}
	return term.IsTerminal(int(fd))
}

type nopReader struct{}

func (nopReader) Read(_ []byte) (int, error) { return 0, io.EOF }

var (
	_ fileWriter = (*stickyWriter)(nil)
	_ fileWriter = (*fdWriter)(nil)
	_ fileReader = (*fdReader)(nil)
)
