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
	"image/color"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	"nabat.dev/theme"

	xterm "github.com/charmbracelet/x/term"
)

// detectCapabilities builds a [theme.Capabilities] snapshot of the
// rendering surface from an [IOStreams] bundle. Themes branch
// on the result to pick capability-aware colors, choose the right
// glamour preset, and back off when the stream is plain text.
//
// detectCapabilities lives in the nabat root because it depends on
// [IOStreams]; the [theme] package is a leaf and must not pull in IO.
// Tests in the theme package construct [theme.Capabilities] directly
// without calling this function.
//
// detectCapabilities returns a default-dark, no-color, non-interactive
// snapshot when io is nil so callers do not have to nil-check on every
// path. In practice [App.finalize] never invokes it with a nil bundle —
// [config.validate] rejects that earlier — but defending the function
// keeps it usable for tests and tooling.
//
// Detection defaults are conservative: when a fact cannot be measured
// reliably (Width, BackgroundHex, Hyperlinks), the framework reports
// the safer (less-feature) value rather than guessing.
func detectCapabilities(io *IOStreams) theme.Capabilities {
	if io == nil {
		return theme.Capabilities{
			Dark:    true,
			Unicode: detectUnicodeFromEnv(),
		}
	}

	caps := theme.Capabilities{
		Profile:       colorProfileFor(io),
		Interactive:   io.IsStdoutTTY(),
		Dark:          true,
		Width:         io.TerminalWidth(),
		Hyperlinks:    detectHyperlinksFromEnv(),
		Unicode:       detectUnicodeFromEnv(),
		ReducedMotion: detectReducedMotionFromEnv(),
	}

	// HasDarkBackground requires *os.File-typed FDs because it issues
	// a control sequence and reads the response. When either stream is
	// not file-backed (tests, redirected output, custom io.Reader) the
	// detection cannot run; assume dark, which matches the convention
	// used by the rest of the Charm.land ecosystem.
	in, okIn := io.RawIn().(xterm.File)
	out, okOut := io.RawOut().(xterm.File)
	if okIn && okOut {
		caps.Dark = lipgloss.HasDarkBackground(in, out)
		if hex, ok := detectBackgroundHex(in, out); ok {
			caps.BackgroundHex = hex
		}
	}

	return caps
}

// detectBackgroundHex queries the terminal background color via the
// OSC 11 helper that lipgloss already uses. It returns the
// canonicalized "#RRGGBB" form when the query succeeds; the second
// return value is false when the terminal does not support the query
// or the read times out.
//
// Implementation detail: lipgloss exposes a [lipgloss.BackgroundColor]
// helper that returns a [color.Color]. Empty / NoColor results
// indicate no detection happened (terminal silent, query disabled).
func detectBackgroundHex(in, out xterm.File) (string, bool) {
	c, err := lipgloss.BackgroundColor(in, out)
	if err != nil || c == nil {
		return "", false
	}
	if _, isNoColor := c.(lipgloss.NoColor); isNoColor {
		return "", false
	}
	return colorHex(c), true
}

// colorHex formats a [color.Color] as the canonical "#RRGGBB" string
// the [theme.Capabilities.BackgroundHex] field uses. It returns the
// empty string when c is nil — the caller already screens for
// NoColor.
func colorHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	const hex = "0123456789ABCDEF"
	out := []byte{'#', 0, 0, 0, 0, 0, 0}
	out[1] = hex[(r>>12)&0xF]
	out[2] = hex[(r>>8)&0xF]
	out[3] = hex[(g>>12)&0xF]
	out[4] = hex[(g>>8)&0xF]
	out[5] = hex[(b>>12)&0xF]
	out[6] = hex[(b>>8)&0xF]
	return string(out)
}

// detectHyperlinksFromEnv reports whether the terminal advertises
// OSC 8 hyperlink support. There is no universal capability flag, so
// the heuristic walks well-known TERM_PROGRAM / VTE_VERSION /
// TERM_FEATURES indicators that mainstream terminals set.
//
// Conservative: when in doubt, return false. Themes / extensions
// that emit links degrade to raw URL text in that case, which is
// always safe.
func detectHyperlinksFromEnv() bool {
	if v := os.Getenv("TERM_FEATURES"); v != "" {
		for f := range strings.SplitSeq(v, ",") {
			if strings.TrimSpace(f) == "hyperlink" {
				return true
			}
		}
	}
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "ghostty", "vscode":
		return true
	}
	if os.Getenv("VTE_VERSION") != "" {
		// VTE >= 0.50 supports hyperlinks; the env var only set
		// when VTE is the host. Treat the presence of any version
		// as supportive — pre-0.50 VTE is rare in 2026.
		return true
	}
	return false
}

// detectUnicodeFromEnv reports the Unicode capability tier of the
// terminal. Heuristic order:
//
//  1. Explicit env override [NABAT_UNICODE] (ascii / wide / emoji).
//  2. LANG / LC_* containing "UTF-8" -> at least UnicodeWide.
//  3. TERM_PROGRAM signals known to render emoji -> UnicodeEmoji.
//  4. Fallback: UnicodeASCII (the safe choice).
//
// The framework's own table / tree / list paths consult this so
// they substitute ASCII enumerators when the terminal cannot render
// the wide glyphs.
func detectUnicodeFromEnv() theme.UnicodeLevel {
	switch strings.ToLower(os.Getenv("NABAT_UNICODE")) {
	case "ascii":
		return theme.UnicodeASCII
	case "wide":
		return theme.UnicodeWide
	case "emoji":
		return theme.UnicodeEmoji
	}

	utf8 := false
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := strings.ToLower(os.Getenv(key))
		if v == "" {
			continue
		}
		if strings.Contains(v, "utf-8") || strings.Contains(v, "utf8") {
			utf8 = true
			break
		}
	}
	if !utf8 {
		return theme.UnicodeASCII
	}

	// Emoji-capable terminals: a small allow-list of TERM_PROGRAM
	// values known to render multi-codepoint sequences correctly.
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app", "WezTerm", "ghostty", "Apple_Terminal", "vscode":
		return theme.UnicodeEmoji
	}
	return theme.UnicodeWide
}

// detectReducedMotionFromEnv reports whether the framework should
// suppress animations. The env vars checked here mirror the
// accessibility flags adopted by other CLI ecosystems
// (Spectre.Console's NO_MOTION, freedesktop's REDUCE_MOTION proposal,
// the do-not-blink convention).
//
// Any non-empty truthy value enables the flag; "0" / "false" / "no"
// keep it off.
func detectReducedMotionFromEnv() bool {
	for _, key := range []string{"NABAT_REDUCED_MOTION", "REDUCE_MOTION", "NO_MOTION"} {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			continue
		}
		switch strings.ToLower(v) {
		case "0", "false", "no", "off":
			continue
		}
		return true
	}
	return false
}

// colorProfileFor returns the colorprofile.Profile chosen for the
// IOStreams' primary output. It is exported in spirit (the App builds
// it once and stores it on the resolved theme) but not on the IOStreams
// type because the bundle already encodes the same information through
// [IOStreams.ColorEnabled]; this helper is the minimal escape
// hatch that lets the theme layer ask "what's the actual profile?".
//
// The implementation re-detects against the raw output stream rather
// than reading a cached field on IOStreams because IOStreams does not
// expose its outProfile publicly. The cost is one syscall per App.New;
// caching at the app layer would not be safer (tests often swap the
// IOStreams entirely).
func colorProfileFor(io *IOStreams) colorprofile.Profile {
	if !io.IsStdoutTTY() {
		return colorprofile.NoTTY
	}
	return colorprofile.Detect(io.RawOut(), nil)
}
