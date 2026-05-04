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

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"

	glamourstyles "charm.land/glamour/v2/styles"
)

// GlamourFromTokens derives a glamour style from semantic tokens.
// The base style is cloned so callers can pass one of glamour's
// built-in dark/light/notty presets and keep untouched slots intact.
func GlamourFromTokens(tokens map[Token]lipgloss.Style, base *ansi.StyleConfig) *ansi.StyleConfig {
	if base == nil {
		base = &glamourstyles.DarkStyleConfig
	}
	cfg := cloneGlamourConfig(base)

	heading := tokens[TextTitle]
	link := tokens[TextLink]
	secondary := tokens[TextSecondary]
	primary := tokens[TextPrimary]
	codeSurface := tokens[CodeSurface]

	applyHeadingStyle(&cfg, heading)
	applyPrimitive(&cfg.Link, link, false)
	applyPrimitive(&cfg.LinkText, link, false)
	applyPrimitive(&cfg.BlockQuote.StylePrimitive, secondary, false)
	applyPrimitive(&cfg.Emph, secondary, false)
	applyPrimitive(&cfg.Strong, primary, true)
	applyPrimitive(&cfg.Code.StylePrimitive, primary, false)
	applyCodeSurface(&cfg.Code.StylePrimitive, codeSurface)
	return &cfg
}

func cloneGlamourConfig(src *ansi.StyleConfig) ansi.StyleConfig {
	return *src
}

func applyHeadingStyle(cfg *ansi.StyleConfig, s lipgloss.Style) {
	applyPrimitive(&cfg.Heading.StylePrimitive, s, false)
	applyPrimitive(&cfg.H1.StylePrimitive, s, false)
	applyPrimitive(&cfg.H2.StylePrimitive, s, false)
	applyPrimitive(&cfg.H3.StylePrimitive, s, false)
	applyPrimitive(&cfg.H4.StylePrimitive, s, false)
	applyPrimitive(&cfg.H5.StylePrimitive, s, false)
	applyPrimitive(&cfg.H6.StylePrimitive, s, false)
}

func applyPrimitive(dst *ansi.StylePrimitive, s lipgloss.Style, forceBold bool) {
	if hex, ok := colorHexUpper(s.GetForeground()); ok {
		dst.Color = new(hex)
	}
	if hex, ok := colorHexUpper(s.GetBackground()); ok {
		dst.BackgroundColor = new(hex)
	}
	if s.GetBold() || forceBold {
		dst.Bold = new(true)
	}
	if s.GetItalic() {
		dst.Italic = new(true)
	}
	if s.GetUnderline() {
		dst.Underline = new(true)
	}
	if s.GetStrikethrough() {
		dst.CrossedOut = new(true)
	}
	if s.GetFaint() {
		dst.Faint = new(true)
	}
	if s.GetBlink() {
		dst.Blink = new(true)
	}
	if s.GetReverse() {
		dst.Inverse = new(true)
	}
}

func applyCodeSurface(dst *ansi.StylePrimitive, surface lipgloss.Style) {
	if hex, ok := colorHexUpper(surface.GetBackground()); ok {
		dst.BackgroundColor = new(hex)
		return
	}
	if hex, ok := colorHexUpper(surface.GetForeground()); ok {
		dst.BackgroundColor = new(hex)
	}
}

// ChromaFromTokens derives a chroma style from semantic tokens.
func ChromaFromTokens(name string, tokens map[Token]lipgloss.Style) *chroma.Style {
	if name == "" {
		name = "nabat-derived"
	}
	entries := chroma.StyleEntries{
		chroma.Background:      chromaEntry(tokens[TextPrimary], tokens[CodeSurface]),
		chroma.Text:            chromaEntry(tokens[TextPrimary], lipgloss.Style{}),
		chroma.Comment:         chromaEntry(tokens[TextMuted].Italic(true), lipgloss.Style{}),
		chroma.Keyword:         chromaEntry(tokens[TextTitle].Bold(true), lipgloss.Style{}),
		chroma.NameFunction:    chromaEntry(tokens[StatusInfo], lipgloss.Style{}),
		chroma.NameBuiltin:     chromaEntry(tokens[AccentPrimary], lipgloss.Style{}),
		chroma.LiteralString:   chromaEntry(tokens[StatusSuccess], lipgloss.Style{}),
		chroma.LiteralNumber:   chromaEntry(tokens[StatusWarning], lipgloss.Style{}),
		chroma.Operator:        chromaEntry(tokens[TextSecondary], lipgloss.Style{}),
		chroma.Punctuation:     chromaEntry(tokens[TextSecondary], lipgloss.Style{}),
		chroma.GenericDeleted:  chromaEntry(tokens[StatusError], lipgloss.Style{}),
		chroma.GenericInserted: chromaEntry(tokens[StatusSuccess], lipgloss.Style{}),
	}
	style, err := chroma.NewStyle(name, entries)
	if err != nil {
		return nil
	}
	return style
}

func chromaEntry(fgStyle, bgStyle lipgloss.Style) string {
	var parts []string
	if fgStyle.GetBold() {
		parts = append(parts, "bold")
	}
	if fgStyle.GetItalic() {
		parts = append(parts, "italic")
	}
	if fgStyle.GetUnderline() {
		parts = append(parts, "underline")
	}
	if hex, ok := colorHexUpper(fgStyle.GetForeground()); ok {
		parts = append(parts, hex)
	}
	bgHex, ok := colorHexUpper(bgStyle.GetBackground())
	if !ok {
		bgHex, ok = colorHexUpper(bgStyle.GetForeground())
	}
	if ok {
		parts = append(parts, "bg:"+bgHex)
	}
	return strings.Join(parts, " ")
}

func colorHexUpper(c color.Color) (string, bool) {
	if c == nil {
		return "", false
	}
	if _, ok := c.(lipgloss.NoColor); ok {
		return "", false
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8), true
}
