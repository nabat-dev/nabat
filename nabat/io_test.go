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
	assert.Equal(t, os.Stdout.Fd(), s.Out.Fd())
	assert.Equal(t, os.Stderr.Fd(), s.ErrOut.Fd())
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

	_, err := s.Out.Write([]byte("\x1b[31mred\x1b[0m"))
	require.NoError(t, err)
	assert.Equal(t, "red", out.String())
}

func TestColorPolicy_honorsCliColorForce(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	out := &bytes.Buffer{}
	s := nabat.NewIO(bytes.NewBufferString(""), out, &bytes.Buffer{})

	_, err := s.Out.Write([]byte("\x1b[31mred\x1b[0m"))
	require.NoError(t, err)
	assert.Contains(t, out.String(), "\x1b[", "ANSI escapes should be preserved when CLICOLOR_FORCE is set")
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
	assert.False(t, s.IsInteractive())
	assert.Equal(t, nabat.DefaultWidth, s.TerminalWidth())
}
