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
	"fmt"
	"io"
)

// writer wraps an [io.Writer] with sticky-error semantics: the first write error
// is stored, and later calls become no-ops. Nabat output methods construct one
// per call around c.io.Out (or c.io.ErrOut) so a broken pipe in the middle of a
// Print/JSON/Table sequence does not surface as a noisy second error and so
// every output method shares the same color-profile and sticky-error policy.
type writer struct {
	w   io.Writer
	err error
}

// Write implements [io.Writer]. After the first error, it returns that error
// without writing to the underlying writer.
func (w *writer) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	var n int
	n, w.err = w.w.Write(p)
	return n, w.err
}

// Err returns the first write error encountered by this writer, or nil. Used by
// structured-output methods (JSON/YAML/TOML) so callers can return the error
// instead of silently swallowing it.
func (w *writer) Err() error { return w.err }

func (w *writer) println(a ...any) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintln(w.w, a...)
}

func (w *writer) printf(format string, a ...any) {
	if w.err != nil {
		return
	}
	_, w.err = fmt.Fprintf(w.w, format, a...)
}

// writeString writes s without adding a newline. It uses [io.WriteString] when
// the underlying writer supports it.
func (w *writer) writeString(s string) {
	if w.err != nil {
		return
	}
	_, w.err = io.WriteString(w.w, s)
}
