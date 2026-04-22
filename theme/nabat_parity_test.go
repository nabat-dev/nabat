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

package theme_test

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// TestNabatChromaParityFromManifest exercises the nabat theme after its
// chroma styling moved out of chroma_style_nabat.go into the manifest.
// Each entry is the same chroma TokenType / hex / modifier that the
// deleted Go file declared; if the manifest drifts from those values
// the test fails immediately.
func TestNabatChromaParityFromManifest(t *testing.T) {
	t.Parallel()

	th, err := theme.Get("nabat")
	require.NoError(t, err)
	rt, err := th.ResolveErr(theme.Capabilities{Dark: true, Interactive: true})
	require.NoError(t, err)

	chromaStyle := rt.Chroma()
	require.NotNil(t, chromaStyle, "nabat manifest must produce an owned chroma style after PR5")

	want := map[chroma.TokenType]struct {
		fg   string
		bold bool
	}{
		chroma.NameFunction:    {fg: "#3EB0CC"},
		chroma.NameBuiltin:     {fg: "#C89B3C"},
		chroma.LiteralString:   {fg: "#7EC87E"},
		chroma.LiteralNumber:   {fg: "#E8A83A"},
		chroma.GenericDeleted:  {fg: "#E05454"},
		chroma.GenericInserted: {fg: "#7EC87E"},
		chroma.Keyword:         {fg: "#D4A853", bold: true},
	}
	for tt, w := range want {
		entry := chromaStyle.Get(tt)
		got := entry.Colour.String()
		assert.Truef(t, sameHexInsensitive(got, w.fg), "%v fg = %q, want %q", tt, got, w.fg)
		if w.bold {
			assert.Equalf(t, chroma.Yes, entry.Bold, "%v expected bold", tt)
		}
	}
}

// TestNabatPromptParityFromManifest verifies that the nabat manifest's
// promptStyle block lands on the resolved huh.Theme with the expected
// per-slot colors. Phase 8 collapsed the old huh-shaped manifest tree
// into the framework-owned [theme.Prompt] struct, so the parity check
// runs against the slots Prompt.Huh actually populates — not the
// hierarchical focused/blurred surface the legacy manifest used.
//
// The Blurred mirror is verified separately so anyone removing the
// "blurred = focused" mirror in Prompt.Huh sees the regression.
func TestNabatPromptParityFromManifest(t *testing.T) {
	t.Parallel()

	th, err := theme.Get("nabat")
	require.NoError(t, err)
	rt, err := th.ResolveErr(theme.Capabilities{Dark: true, Interactive: true})
	require.NoError(t, err)

	huhTheme := rt.Huh()
	require.NotNil(t, huhTheme, "nabat manifest must produce a huh theme after P8")
	s := huhTheme.Theme(true)

	cases := []struct {
		name string
		got  color.Color
		want color.Color
	}{
		{"focused.title fg", s.Focused.Title.GetForeground(), lipgloss.Color("#D4A853")},
		{"focused.description fg", s.Focused.Description.GetForeground(), lipgloss.Color("#D5CDC2")},
		{"focused.errorIndicator fg", s.Focused.ErrorIndicator.GetForeground(), lipgloss.Color("#E05454")},
		{"focused.selectSelector fg", s.Focused.SelectSelector.GetForeground(), lipgloss.Color("#3EB0CC")},
		{"focused.selectedOption fg", s.Focused.SelectedOption.GetForeground(), lipgloss.Color("#7EC87E")},
		{"focused.selectedPrefix fg", s.Focused.SelectedPrefix.GetForeground(), lipgloss.Color("#7EC87E")},
		{"focused.unselectedPrefix fg", s.Focused.UnselectedPrefix.GetForeground(), lipgloss.Color("#8A7E72")},
		{"focused.textInput.cursor fg", s.Focused.TextInput.Cursor.GetForeground(), lipgloss.Color("#3EB0CC")},
		{"focused.focusedButton fg", s.Focused.FocusedButton.GetForeground(), lipgloss.Color("#C89B3C")},
		{"focused.blurredButton fg", s.Focused.BlurredButton.GetForeground(), lipgloss.Color("#8A7E72")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Truef(t, sameColor(c.got, c.want), "%s: got %v, want %v", c.name, c.got, c.want)
		})
	}

	assert.True(t, s.Focused.FocusedButton.GetBold(),
		"focused.focusedButton bold should propagate from accent.primary")

	// SetString text propagates onto the prefix slot.
	assert.Equal(t, "✓ ", s.Focused.SelectedPrefix.Value(), "focused.selectedPrefix text")
	assert.Equal(t, "• ", s.Focused.UnselectedPrefix.Value(), "focused.unselectedPrefix text")

	// Border override applies to focused.base and is mirrored to
	// blurred.base (with the framework's hidden-border treatment so
	// the inactive form does not visually compete).
	assert.Equal(t, lipgloss.RoundedBorder(), s.Focused.Base.GetBorderStyle(),
		"focused.base border should match promptKnobs.border")
	assert.Equal(t, lipgloss.HiddenBorder(), s.Blurred.Base.GetBorderStyle(),
		"blurred.base border should be hidden when promptStyle.border is set")

	// Prompt.Huh deliberately mirrors focused -> blurred so the
	// inactive state stays on-palette without the manifest having to
	// declare a separate per-state surface.
	assert.Truef(t, sameColor(s.Blurred.Title.GetForeground(), lipgloss.Color("#D4A853")),
		"blurred.title should mirror focused.title color, got %v", s.Blurred.Title.GetForeground())
}

// sameColor compares two [color.Color] values by their RGBA components.
// [lipgloss.Color] values produced from equal hex strings compare equal
// this way without depending on internal representation.
func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// sameHexInsensitive compares two "#rrggbb" strings without caring
// about case. chroma's Colour.String() lowercases the hex; the
// manifest authors uppercase. Both encode the same color.
func sameHexInsensitive(a, b string) bool {
	return strings.EqualFold(a, b)
}
