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

package theme

import "github.com/charmbracelet/colorprofile"

// Capabilities describes the rendering surface a [Theme] resolves against.
// Themes branch on these fields for colors, plain-text fallbacks, and
// glamour presets.
//
// Capabilities is a plain constructible struct: the theme package has no
// IOStreams dependency, so tests build values directly. The nabat root
// package owns detection. When detection is uncertain, the framework
// reports the safer (less-feature) value.
type Capabilities struct {
	// Dark reports whether the terminal background is dark.
	Dark bool

	// BackgroundHex is the exact terminal background color when the
	// detector could read it. Empty when detection could not run.
	BackgroundHex string

	// Profile is the active [colorprofile.Profile] for the primary
	// output stream.
	Profile colorprofile.Profile

	// Interactive reports whether primary output is a TTY and input
	// allows prompting.
	Interactive bool

	// Width is the terminal width in cells, or 0 when unmeasured.
	Width int

	// Hyperlinks reports OSC 8 hyperlink support.
	Hyperlinks bool

	// Unicode is the terminal's Unicode capability tier.
	Unicode UnicodeLevel

	// ReducedMotion reports whether animations should be suppressed
	// (NO_MOTION, REDUCE_MOTION, and similar flags).
	ReducedMotion bool
}

// UnicodeLevel is how much Unicode the terminal can render. Tiers are
// monotonic: higher values include the lower ones, so consumers can
// compare for "at least N".
type UnicodeLevel uint8

// UnicodeLevel constants:
//
//   - [UnicodeASCII]: 7-bit ASCII only; use "+", "-", "|" for boxes.
//   - [UnicodeWide]: box-drawing, bullets, and arrows (framework default).
//   - [UnicodeEmoji]: emoji and multi-codepoint sequences.
const (
	UnicodeASCII UnicodeLevel = iota
	UnicodeWide
	UnicodeEmoji
)
