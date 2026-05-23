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

package nabat_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func TestNewSystemIO_returnsRealStreams(t *testing.T) {
	t.Parallel()

	s := nabat.NewSystemIO()
	require.NotNil(t, s)
	assert.NotNil(t, s.In)
	assert.NotNil(t, s.Out)
	assert.NotNil(t, s.ErrOut)
	assert.Same(t, os.Stdin, s.RawIn())
	assert.Same(t, os.Stdout, s.RawOut())
	assert.Same(t, os.Stderr, s.RawErrOut())
}

func TestNewIO_writesGoThroughBuffers(t *testing.T) {
	t.Parallel()

	in := bytes.NewBufferString("hello")
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	s := nabat.NewIO(in, out, errOut)

	_, err := s.Out.Write([]byte("a"))
	require.NoError(t, err)
	_, err = s.ErrOut.Write([]byte("b"))
	require.NoError(t, err)

	assert.Equal(t, "a", out.String())
	assert.Equal(t, "b", errOut.String())
}

func TestNewIO_defaultsToNonTTY(t *testing.T) {
	t.Parallel()

	s, in, out, errOut := nabattest.NewIO()
	require.NotNil(t, s)
	require.NotNil(t, in)
	require.NotNil(t, out)
	require.NotNil(t, errOut)

	assert.False(t, s.IsStdinTTY())
	assert.False(t, s.IsStdoutTTY())
	assert.False(t, s.IsStderrTTY())
	assert.False(t, s.CanPrompt())
	assert.False(t, s.ColorEnabled())
	assert.Equal(t, nabat.DefaultWidth, s.TerminalWidth())
}

func TestSetTTY_overridesDetection(t *testing.T) {
	t.Parallel()

	s, _, _, _ := nabattest.NewIO()
	s.SetStdinTTY(true)
	s.SetStdoutTTY(true)
	s.SetStderrTTY(true)

	assert.True(t, s.IsStdinTTY())
	assert.True(t, s.IsStdoutTTY())
	assert.True(t, s.IsStderrTTY())
	assert.True(t, s.CanPrompt())
}

func TestColorPolicy_stripsAnsiOnNonTTY(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	s := nabat.NewIO(bytes.NewBufferString(""), out, &bytes.Buffer{})

	_, err := s.Out.Write([]byte("\x1b[31mred\x1b[0m"))
	require.NoError(t, err)
	assert.Equal(t, "red", out.String(), "ANSI escapes should be stripped when stdout is not a TTY")
}

func TestColorPolicy_honorsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	out := &bytes.Buffer{}
	s := nabat.NewIO(bytes.NewBufferString(""), out, &bytes.Buffer{})

	assert.False(t, s.ColorEnabled())

	_, err := s.Out.Write([]byte("\x1b[31mred\x1b[0m"))
	require.NoError(t, err)
	assert.Equal(t, "red", out.String())
}

func TestColorPolicy_honorsCliColorForce(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	out := &bytes.Buffer{}
	s := nabat.NewIO(bytes.NewBufferString(""), out, &bytes.Buffer{})

	assert.True(t, s.ColorEnabled())

	_, err := s.Out.Write([]byte("\x1b[31mred\x1b[0m"))
	require.NoError(t, err)
	assert.Contains(t, out.String(), "\x1b[", "ANSI escapes should be preserved when CLICOLOR_FORCE is set")
}

func TestStickyError_subsequentWritesShortCircuit(t *testing.T) {
	t.Parallel()

	failingOut := &ioFailingWriter{}
	s := nabat.NewIO(bytes.NewBufferString(""), failingOut, &bytes.Buffer{})

	_, err := s.Out.Write([]byte("first"))
	require.Error(t, err)
	require.ErrorIs(t, err, errIOBoom)

	failingOut.fail = false
	n, err := s.Out.Write([]byte("second"))
	require.Error(t, err, "after sticky error, subsequent writes must keep failing")
	require.ErrorIs(t, err, errIOBoom)
	assert.Equal(t, 0, n)
	assert.Equal(t, 1, failingOut.calls, "underlying writer must not be called after sticky error")

	require.ErrorIs(t, s.Err(), errIOBoom)
}

func TestStickyError_isSharedBetweenOutAndErrOut(t *testing.T) {
	t.Parallel()

	failingErr := &ioFailingWriter{}
	out := &bytes.Buffer{}
	s := nabat.NewIO(bytes.NewBufferString(""), out, failingErr)

	_, err := s.ErrOut.Write([]byte("oops"))
	require.Error(t, err)

	n, writeErr := s.Out.Write([]byte("data"))
	require.Error(t, writeErr, "stdout writes should also short-circuit after a stderr failure")
	assert.Equal(t, 0, n)
	assert.Empty(t, out.String())
}

func TestFdPreserved_throughWrapper(t *testing.T) {
	t.Parallel()

	s := nabat.NewSystemIO()

	type fdGetter interface{ Fd() uintptr }

	out, ok := s.Out.(fdGetter)
	require.True(t, ok, "wrapped Out should implement Fd()")
	assert.Equal(t, os.Stdout.Fd(), out.Fd())

	errOut, ok := s.ErrOut.(fdGetter)
	require.True(t, ok, "wrapped ErrOut should implement Fd()")
	assert.Equal(t, os.Stderr.Fd(), errOut.Fd())
}

func TestNewIO_handlesNilStreams(t *testing.T) {
	t.Parallel()

	s := nabat.NewIO(nil, nil, nil)
	require.NotNil(t, s)

	_, err := s.Out.Write([]byte("ignored"))
	require.NoError(t, err)

	var buf [4]byte
	n, err := s.In.Read(buf[:])
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, io.EOF)
}

var errIOBoom = errors.New("boom")

type ioFailingWriter struct {
	fail  bool
	calls int
}

func (w *ioFailingWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.fail || w.calls == 1 {
		return 0, errIOBoom
	}
	return len(p), nil
}
