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

	"gopherly.dev/termio"
	"gopherly.dev/termio/colorprofile"
)

// IOStreams is the terminal I/O bundle shared by an [App] and every
// [Context] it produces. It is an alias for [termio.Streams]. Out and
// ErrOut are color-adaptive [*termio.Writer] values; In is the unwrapped
// reader. Safe to construct concurrently, but not for concurrent SetXxxTTY
// mutation.
type IOStreams = termio.Streams

// DefaultWidth is the terminal column width assumed when the underlying
// stream is not a terminal or its size cannot be determined.
const DefaultWidth = termio.DefaultWidth

// DefaultHeight is the terminal row count assumed when the underlying
// stream is not a terminal or its size cannot be determined.
const DefaultHeight = termio.DefaultHeight

// NewSystemIO returns an IOStreams backed by [os.Stdin], [os.Stdout], and
// [os.Stderr]. Color profile and TTY status are detected against the real
// file descriptors and the process environment.
func NewSystemIO() *IOStreams {
	env := os.Environ()
	return termio.System(
		termio.WithColorPolicy(colorprofile.Detect(os.Stdout, env)),
	)
}

// NewIO returns an IOStreams over the supplied streams. Out and ErrOut are
// color-adaptive; In is unchanged. Pass *[os.File] for real TTY detection;
// in tests prefer [nabattest.NewIO].
func NewIO(in io.Reader, out, errOut io.Writer) *IOStreams {
	env := os.Environ()
	return termio.New(in, out, errOut,
		termio.WithColorPolicy(colorprofile.Detect(out, env)),
	)
}
