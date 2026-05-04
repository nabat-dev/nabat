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
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Prompt is the Nabat-native style block for interactive prompts. It
// is a closed enum of the slots the framework cares about — closed
// meaning the schema and the manifest authoring experience never
// have to track the upstream huh.Styles struct shape.
//
// Each field is a [lipgloss.Style] applied to one prompt slot. Zero
// (the default lipgloss.Style) means "inherit huh's base styling for
// this slot" — Prompt overlays only the slots the author opted into.
//
// Authors who need huh's full surface drop into the programmatic
// path with a [huh.Theme] of their own, set via [Palette.Huh]
// directly. That escape hatch wins outright when present, so the
// closed Prompt surface and the open huh.Theme escape hatch
// coexist without confusion.
//
// Fields that take a literal text payload (the prefix-style markers)
// use [lipgloss.Style.SetString] to inject the literal in addition
// to the visual style — `theme.SetString` style call sites work the
// same way they did in the old huhStyle path.
type Prompt struct {
	// Title is the group / section title shown above prompts.
	Title lipgloss.Style

	// Description is the explanatory text rendered under each
	// prompt.
	Description lipgloss.Style

	// Cursor styles the text-input cursor.
	Cursor lipgloss.Style

	// Placeholder styles the text-input placeholder copy.
	Placeholder lipgloss.Style

	// SelectedOption styles the currently-selected list item.
	SelectedOption lipgloss.Style

	// UnselectedOption styles the items not currently selected.
	UnselectedOption lipgloss.Style

	// SelectedPrefix styles the marker drawn next to the selected
	// item (often "✓ " or "● ").
	SelectedPrefix lipgloss.Style

	// UnselectedPrefix styles the marker drawn next to non-selected
	// items (often "  " or "○ ").
	UnselectedPrefix lipgloss.Style

	// Error styles error indicators and messages.
	Error lipgloss.Style

	// Help styles the keybind footer text.
	Help lipgloss.Style

	// Selector styles the active-row indicator and navigation
	// arrows (next / prev).
	Selector lipgloss.Style

	// ButtonFocused styles the focused submit / next button.
	ButtonFocused lipgloss.Style

	// ButtonBlurred styles the inactive button.
	ButtonBlurred lipgloss.Style

	// Border applies as the form / card border. The zero
	// [lipgloss.Border] leaves huh's base border untouched.
	Border lipgloss.Border
}

// PromptKnobs carries non-color prompt settings that are convenient to
// share across variants.
type PromptKnobs struct {
	// SelectedPrefix is the literal prefix rendered before selected
	// options (for example "✓ ").
	SelectedPrefix string

	// UnselectedPrefix is the literal prefix rendered before
	// unselected options (for example "  ").
	UnselectedPrefix string

	// Border is the form/card border to apply to prompt output. The
	// zero [lipgloss.Border] leaves the prompt border unchanged.
	Border lipgloss.Border
}

// IsZero reports whether k leaves every prompt knob unset.
func (k PromptKnobs) IsZero() bool {
	return k.SelectedPrefix == "" &&
		k.UnselectedPrefix == "" &&
		k.Border == (lipgloss.Border{})
}

// Apply overlays non-zero knobs from k onto p and returns the result.
func (k PromptKnobs) Apply(p Prompt) Prompt {
	if k.SelectedPrefix != "" {
		p.SelectedPrefix = p.SelectedPrefix.SetString(k.SelectedPrefix)
	}
	if k.UnselectedPrefix != "" {
		p.UnselectedPrefix = p.UnselectedPrefix.SetString(k.UnselectedPrefix)
	}
	if k.Border != (lipgloss.Border{}) {
		p.Border = k.Border
	}
	return p
}

// IsZero reports whether p has any field set. The framework uses
// it to detect "this palette did not declare a prompt" so the
// catalog can fall back to [PromptFromTokens] without having to
// special-case nil.
//
// A Prompt with only [Border] set is non-zero; a Prompt with only
// the zero lipgloss.Border but every style field set is also
// non-zero. The zero Prompt — every style empty AND the zero border
// — is the only "unset" value.
//
// lipgloss.Style is not comparable (it carries slices internally),
// so the check inspects every field through styleIsZero rather than
// using `p == Prompt{}`.
func (p Prompt) IsZero() bool {
	return styleIsZero(p.Title) &&
		styleIsZero(p.Description) &&
		styleIsZero(p.Cursor) &&
		styleIsZero(p.Placeholder) &&
		styleIsZero(p.SelectedOption) &&
		styleIsZero(p.UnselectedOption) &&
		styleIsZero(p.SelectedPrefix) &&
		styleIsZero(p.UnselectedPrefix) &&
		styleIsZero(p.Error) &&
		styleIsZero(p.Help) &&
		styleIsZero(p.Selector) &&
		styleIsZero(p.ButtonFocused) &&
		styleIsZero(p.ButtonBlurred) &&
		p.Border == (lipgloss.Border{})
}

// styleIsZero reports whether s has any visible attribute set. The
// check looks at fg, bg, the boolean modifiers, and the literal
// payload — everything overlayStyle considers when deciding whether
// to overlay onto a base style. A style with only "internal" state
// (a frame size, a width, etc.) is treated as zero too because the
// Prompt overlay path never reads those fields.
func styleIsZero(s lipgloss.Style) bool {
	if !isNoColor(s.GetForeground()) {
		return false
	}
	if !isNoColor(s.GetBackground()) {
		return false
	}
	if s.GetBold() || s.GetItalic() || s.GetUnderline() ||
		s.GetStrikethrough() || s.GetFaint() || s.GetBlink() || s.GetReverse() {
		return false
	}
	if s.Value() != "" {
		return false
	}
	return true
}

// Huh returns a [huh.Theme] derived from this Prompt. The theme
// starts from [huh.ThemeBase] and overlays the supplied fields onto
// the slots huh exposes. Zero-style fields inherit huh's base
// styling.
//
// The mapping mirrors the curated set in [PromptFromTokens] — the
// Focused fields users see most (titles, errors, selection
// indicators, prefixes, text input), the Group surface, and the
// Help footer. The Blurred mirror reuses the focused style for
// stable cross-state appearance; authors who want different blurred
// styling drop into [Palette.Huh] directly.
//
// Border applies to Focused.Base and Focused.Card when non-zero,
// then mirrors onto Blurred.Base / Blurred.Card so the form chrome
// stays consistent.
func (p Prompt) Huh() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		s := huh.ThemeBase(isDark)

		s.Focused.Title = overlayStyle(s.Focused.Title, p.Title)
		s.Focused.NoteTitle = overlayStyle(s.Focused.NoteTitle, p.Title)
		s.Focused.Description = overlayStyle(s.Focused.Description, p.Description)
		s.Focused.ErrorIndicator = overlayStyle(s.Focused.ErrorIndicator, p.Error)
		s.Focused.ErrorMessage = overlayStyle(s.Focused.ErrorMessage, p.Error)
		s.Focused.SelectSelector = overlayStyle(s.Focused.SelectSelector, p.Selector)
		s.Focused.MultiSelectSelector = overlayStyle(s.Focused.MultiSelectSelector, p.Selector)
		s.Focused.NextIndicator = overlayStyle(s.Focused.NextIndicator, p.Selector)
		s.Focused.PrevIndicator = overlayStyle(s.Focused.PrevIndicator, p.Selector)
		s.Focused.Option = overlayStyle(s.Focused.Option, p.UnselectedOption)
		s.Focused.SelectedOption = overlayStyle(s.Focused.SelectedOption, p.SelectedOption)
		s.Focused.UnselectedOption = overlayStyle(s.Focused.UnselectedOption, p.UnselectedOption)
		s.Focused.SelectedPrefix = overlayStyle(s.Focused.SelectedPrefix, p.SelectedPrefix)
		s.Focused.UnselectedPrefix = overlayStyle(s.Focused.UnselectedPrefix, p.UnselectedPrefix)
		s.Focused.FocusedButton = overlayStyle(s.Focused.FocusedButton, p.ButtonFocused)
		s.Focused.BlurredButton = overlayStyle(s.Focused.BlurredButton, p.ButtonBlurred)
		s.Focused.Next = overlayStyle(s.Focused.Next, p.ButtonFocused)
		s.Focused.TextInput.Cursor = overlayStyle(s.Focused.TextInput.Cursor, p.Cursor)
		s.Focused.TextInput.Placeholder = overlayStyle(s.Focused.TextInput.Placeholder, p.Placeholder)
		s.Focused.TextInput.Prompt = overlayStyle(s.Focused.TextInput.Prompt, p.Title)
		s.Focused.TextInput.Text = overlayStyle(s.Focused.TextInput.Text, p.UnselectedOption)

		// Mirror focused styling onto blurred so the form keeps a
		// consistent look between active and inactive states. The
		// previous huhStyle pipeline let authors customize blurred
		// independently via $inherit; the closed Prompt surface
		// trades that knob away for simpler authoring. Authors who
		// need separate blurred styling reach for Palette.Huh.
		s.Blurred = s.Focused
		if p.Border != (lipgloss.Border{}) {
			s.Focused.Base = s.Focused.Base.BorderStyle(p.Border)
			s.Focused.Card = s.Focused.Card.BorderStyle(p.Border)
			s.Blurred.Base = s.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
			s.Blurred.Card = s.Blurred.Card.BorderStyle(lipgloss.HiddenBorder())
		}

		s.Group.Title = s.Focused.Title
		s.Group.Description = s.Focused.Description

		s.Help.ShortKey = overlayStyle(s.Help.ShortKey, p.Help)
		s.Help.ShortDesc = overlayStyle(s.Help.ShortDesc, p.Description)
		s.Help.ShortSeparator = overlayStyle(s.Help.ShortSeparator, p.Help)
		s.Help.FullKey = overlayStyle(s.Help.FullKey, p.Help)
		s.Help.FullDesc = overlayStyle(s.Help.FullDesc, p.Description)
		s.Help.FullSeparator = overlayStyle(s.Help.FullSeparator, p.Help)
		s.Help.Ellipsis = overlayStyle(s.Help.Ellipsis, p.Help)

		return s
	})
}

// PromptFromTokens derives a [Prompt] from a per-token style map.
// It is the fallback the catalog applies when a [Palette] declares
// neither [Palette.Prompt] nor [Palette.Huh], so every theme — even
// bare-bones programmatic ones that only declare token colors —
// gets a usable interactive surface for free, themed in the same
// colors as the rest of the CLI.
//
// The mapping is intentionally narrow and stable; new prompt slots
// added to [Prompt] in the future should map here too so the
// "tokens are enough" promise keeps holding.
func PromptFromTokens(tokens map[Token]lipgloss.Style) Prompt {
	return Prompt{
		Title:            tokens[TextTitle],
		Description:      tokens[TextSecondary],
		Cursor:           tokens[StatusInfo],
		Placeholder:      tokens[TextMuted],
		SelectedOption:   tokens[StatusSuccess],
		UnselectedOption: tokens[TextPrimary],
		SelectedPrefix:   tokens[StatusSuccess],
		UnselectedPrefix: tokens[TextMuted],
		Error:            tokens[StatusError],
		Help:             tokens[AccentPrimary],
		Selector:         tokens[StatusInfo],
		ButtonFocused:    tokens[AccentPrimary],
		ButtonBlurred:    tokens[TextMuted],
	}
}

// overlayStyle copies the foreground / background / bold / italic /
// underline / strikethrough / faint flags from src onto dst when src
// has them set. Unset fields preserve dst's value, so partial
// Prompt declarations layer cleanly on top of huh's base styling.
//
// This helper is used by [Prompt.Huh] for every slot mapping; it's
// the one place the "src wins iff src is set" rule lives.
func overlayStyle(dst, src lipgloss.Style) lipgloss.Style {
	if styleIsZero(src) {
		return dst
	}
	out := dst
	if fg := src.GetForeground(); !isNoColor(fg) {
		out = out.Foreground(fg)
	}
	if bg := src.GetBackground(); !isNoColor(bg) {
		out = out.Background(bg)
	}
	if src.GetBold() {
		out = out.Bold(true)
	}
	if src.GetItalic() {
		out = out.Italic(true)
	}
	if src.GetUnderline() {
		out = out.Underline(true)
	}
	if src.GetStrikethrough() {
		out = out.Strikethrough(true)
	}
	if src.GetFaint() {
		out = out.Faint(true)
	}
	if str := src.Value(); str != "" {
		out = out.SetString(str)
	}
	return out
}

// isNoColor returns true when c is the typed lipgloss.NoColor
// sentinel that lipgloss returns for unset color slots. The
// comparison must be against the typed sentinel (not nil) because
// Get* never returns a Go nil interface.
func isNoColor(c any) bool {
	if c == nil {
		return true
	}
	_, ok := c.(lipgloss.NoColor)
	return ok
}
