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
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check that writer implements [io.Writer].
var _ io.Writer = (*writer)(nil)

type spyWriter struct {
	buf       bytes.Buffer
	callCount int
	failOn    int // if > 0, the Nth Write (1-based) returns an error
}

func (s *spyWriter) Write(p []byte) (int, error) {
	s.callCount++
	if s.failOn > 0 && s.callCount == s.failOn {
		return 0, errors.New("write failed")
	}
	return s.buf.Write(p)
}

func TestWriter_Write_success(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := &writer{w: &buf}
	n, err := w.Write([]byte("hi"))
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, "hi", buf.String())
}

func TestWriter_Write_stickyError(t *testing.T) {
	t.Parallel()

	spy := &spyWriter{failOn: 1}
	w := &writer{w: spy}

	n1, err1 := w.Write([]byte("a"))
	assert.Equal(t, 0, n1)
	require.Error(t, err1)

	n2, err2 := w.Write([]byte("b"))
	assert.Equal(t, 0, n2)
	require.Error(t, err2)
	assert.Equal(t, err1, err2)

	assert.Equal(t, 1, spy.callCount, "underlying Write must not run after first error")
	assert.Empty(t, spy.buf.String())
}

func TestWriter_printf_skipsAfterError(t *testing.T) {
	t.Parallel()

	spy := &spyWriter{failOn: 1}
	w := &writer{w: spy}
	w.printf("first")
	w.printf("second")
	assert.Equal(t, 1, spy.callCount)
}

func TestWriter_println(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := &writer{w: &buf}
	w.println("line")
	assert.Equal(t, "line\n", buf.String())
}

func TestWriter_ioWriteString(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := &writer{w: &buf}
	_, err := io.WriteString(w, "abc")
	require.NoError(t, err)
	assert.Equal(t, "abc", buf.String())
}

func TestWriter_writeString(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	w := &writer{w: &buf}
	w.writeString("no newline")
	assert.Equal(t, "no newline", buf.String())
}
