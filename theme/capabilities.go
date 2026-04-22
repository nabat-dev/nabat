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

// Capabilities describes the rendering surface a [Theme] is being resolved
// for. Themes branch on these fields to pick capability-aware colors, fall
// back to plain text when colors are unavailable, and choose the right
// glamour preset for the current background luminance.
//
// Capabilities is a plain, constructible struct on purpose: the theme/
// package is a leaf with no IOStreams dependency, so tests build a
// Capabilities value directly to exercise theme branches without having
// to stand up an IO bundle. The nabat root package owns capability
// detection and produces a populated Capabilities value at App.finalize
// time.
//
// Phase 9 widened the struct beyond the original three fields with
// [Width], [BackgroundHex], [Hyperlinks], [Unicode], and
// [ReducedMotion] so theme recipes and consumers can branch on the
// terminal facts that matter beyond just dark / light. Detection
// defaults remain conservative: in doubt, the framework reports the
// safer (less-feature) value.
type Capabilities struct {
	// Dark reports whether the terminal background is dark. Themes use it
	// to pick foreground hex values that contrast appropriately and to
	// pick between glamour's "dark" and "light" presets.
	Dark bool

	// BackgroundHex is the exact terminal background color when the
	// detector could read it (typically via the OSC 11 query). Empty
	// when the detector could not run (non-TTY output, redirected
	// input, custom IO bundle). Themes that want to adapt to the
	// real background — say, a manifest tuned for a specific
	// off-white — branch on this value; themes that only need
	// "darkish vs lightish" stick with [Dark].
	BackgroundHex string

	// Profile is the active [colorprofile.Profile] for the primary output
	// stream. Themes use it to fall back from full-color hex values to
	// ANSI16/ANSI256 indexes, and to switch glamour to "notty" when the
	// stream is plain text.
	Profile colorprofile.Profile

	// Interactive reports whether the primary output stream is a TTY and
	// the input stream allows prompting. Themes that animate or use
	// background color blocks check this so non-interactive output stays
	// pipe-friendly.
	Interactive bool

	// Width is the terminal width in cells, or 0 when the framework
	// could not measure it (non-TTY output, sandboxed test bundle).
	// Themes use it for table / tree decisions; the framework's own
	// table layout reads it through the IO bundle.
	Width int

	// Hyperlinks reports whether the terminal supports OSC 8
	// hyperlinks. Themes / extensions that emit clickable URLs gate
	// the escape-sequence emission on this flag so they degrade to
	// raw URL text in unsupported terminals (most CI runners,
	// minimal SSH targets, etc.).
	Hyperlinks bool

	// Unicode reports the terminal's Unicode capability tier. Themes
	// pick enumerator characters (•, -, └─) and prefix glyphs based
	// on this; sticking to ASCII when the terminal cannot render
	// box-drawing characters keeps the output readable on legacy
	// Windows consoles and minimal CI environments.
	Unicode UnicodeLevel

	// ReducedMotion reports whether the framework should suppress
	// animations (spinners, progress sweeps, transition effects).
	// Set when the env signals "no motion please" via NO_MOTION,
	// REDUCE_MOTION, or similar accessibility flags. Consumers that
	// run animations check this and substitute a static rendering.
	ReducedMotion bool
}

// UnicodeLevel describes how much of the Unicode plane the terminal
// can render correctly. The tiers compose monotonically — higher
// values include the lower ones — so consumers can compare with
// less-than / greater-than for "at least N".
type UnicodeLevel uint8

// UnicodeLevel constants describe the Unicode capability tier of the
// terminal:
//
//   - [UnicodeASCII]: only 7-bit ASCII renders reliably. Themes
//     should stick to "+", "-", "|" for box drawing.
//   - [UnicodeWide]: Unicode wide characters render correctly,
//     including box-drawing, list bullets, and arrows. The
//     framework's defaults assume this tier.
//   - [UnicodeEmoji]: emoji and other multi-codepoint sequences
//     render correctly. Themes that want to use emoji glyphs (✓ ❌
//     🔍) gate them on this tier.
const (
	UnicodeASCII UnicodeLevel = iota
	UnicodeWide
	UnicodeEmoji
)
