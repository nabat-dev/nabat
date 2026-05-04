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
	"io"
	"os"
)

// fileWriter is the interface any TTY-aware library should accept: an
// [io.Writer] that also exposes its file descriptor. Wrapping a writer
// in a [fdWriter] preserves Fd() through the colorprofile-aware writer
// stack, so downstream libraries (huh, bubbletea, glamour) can still
// detect the terminal even when they receive a wrapped value.
type fileWriter interface {
	io.Writer
	Fd() uintptr
}

// fileReader is the read-side companion to [fileWriter].
type fileReader interface {
	io.Reader
	Fd() uintptr
}

// fdWriter wraps an [io.Writer] (typically a colorprofile.Writer) and
// preserves a file descriptor obtained from the original *[os.File].
type fdWriter struct {
	io.Writer
	fd uintptr
}

func (w *fdWriter) Fd() uintptr { return w.fd }

// fdReader is the read-side companion to [fdWriter].
type fdReader struct {
	io.Reader
	fd uintptr
}

func (r *fdReader) Fd() uintptr { return r.fd }

// invalidFd is the sentinel returned by Fd() when the underlying writer
// or reader is not backed by a real file descriptor (e.g. tests using a
// [bytes.Buffer]). It is the maximum uintptr value, which never refers
// to a valid descriptor on supported platforms.
const invalidFd = ^uintptr(0)

// fdOf reports the file descriptor of v when it is an *[os.File] or already
// implements Fd() uintptr; otherwise it returns invalidFd.
func fdOf(v any) uintptr {
	switch f := v.(type) {
	case *os.File:
		if f == nil {
			return invalidFd
		}
		return f.Fd()
	case interface{ Fd() uintptr }:
		return f.Fd()
	default:
		return invalidFd
	}
}
