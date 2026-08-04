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
	"gopherly.dev/termio/colorprofile"

	"nabat.dev/theme"

	xterm "github.com/charmbracelet/x/term"
)

// detectCapabilities builds a [theme.Capabilities] snapshot from an
// [IOStreams] bundle and the pre-detected color profile. It lives in nabat
// because it depends on [IOStreams]; theme tests construct Capabilities
// directly. A nil io yields a default-dark, no-color, non-interactive
// snapshot. Unmeasurable facts (Width, BackgroundHex, Hyperlinks) report
// the safer, less-featured value.
func detectCapabilities(io *IOStreams, profile colorprofile.Profile) theme.Capabilities {
	if io == nil {
		return theme.Capabilities{
			Dark:    true,
			Unicode: detectUnicodeFromEnv(),
		}
	}

	caps := theme.Capabilities{
		Profile:       profile,
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

// detectBackgroundHex queries the terminal background via OSC 11
// ([lipgloss.BackgroundColor]). It returns "#RRGGBB" on success; ok is
// false when the terminal is silent, returns NoColor, or times out.
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
// used by [theme.Capabilities.BackgroundHex]. Returns "" when c is nil.
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

// detectHyperlinksFromEnv reports OSC 8 hyperlink support from
// TERM_FEATURES, TERM_PROGRAM, and VTE_VERSION. Returns false when
// unsure so link emitters degrade to raw URL text.
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
		// as supportive; pre-0.50 VTE is rare in 2026.
		return true
	}
	return false
}

// detectUnicodeFromEnv reports the Unicode capability tier: NABAT_UNICODE
// override, then LANG/LC_* UTF-8 (UnicodeWide), then TERM_PROGRAM emoji
// hosts (UnicodeEmoji), else UnicodeASCII.
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

// detectReducedMotionFromEnv reports whether animations should be
// suppressed. Checks NABAT_REDUCED_MOTION, REDUCE_MOTION, and NO_MOTION;
// truthy values enable it, while "0" / "false" / "no" / "off" keep it off.
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
