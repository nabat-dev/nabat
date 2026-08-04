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
	"image/color"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Prompt is the closed Nabat-native style block for interactive prompts.
// Each field is a [lipgloss.Style] for one slot; the zero style means
// inherit huh's base styling for that slot.
//
// For huh's full surface, set [Palette.Huh] instead; a non-nil Huh wins
// over Prompt. Prefix markers use [lipgloss.Style.SetString] for their
// literal text.
type Prompt struct {
	// Title styles the group / section title above prompts.
	Title lipgloss.Style

	// Description styles explanatory text under each prompt.
	Description lipgloss.Style

	// Cursor styles the text-input cursor.
	Cursor lipgloss.Style

	// Placeholder styles the text-input placeholder.
	Placeholder lipgloss.Style

	// SelectedOption styles the currently selected list item.
	SelectedOption lipgloss.Style

	// UnselectedOption styles items that are not selected.
	UnselectedOption lipgloss.Style

	// SelectedPrefix styles the marker next to the selected item.
	SelectedPrefix lipgloss.Style

	// UnselectedPrefix styles the marker next to non-selected items.
	UnselectedPrefix lipgloss.Style

	// Error styles error indicators and messages.
	Error lipgloss.Style

	// Help styles the keybind footer text.
	Help lipgloss.Style

	// Selector styles the active-row indicator and navigation arrows.
	Selector lipgloss.Style

	// ButtonFocused styles the focused submit / next button.
	ButtonFocused lipgloss.Style

	// ButtonBlurred styles the inactive button.
	ButtonBlurred lipgloss.Style

	// Border is the form / card border. The zero [lipgloss.Border]
	// leaves huh's base border untouched.
	Border lipgloss.Border

	// BorderColor is the focused field left-border foreground. The
	// zero value leaves huh's base border color untouched.
	BorderColor color.Color
}

// PromptKnobs carries theme-wide prompt settings shared across variants.
type PromptKnobs struct {
	// SelectedPrefix is the literal prefix before selected options.
	SelectedPrefix string

	// UnselectedPrefix is the literal prefix before unselected options.
	UnselectedPrefix string

	// Border is the form/card border. The zero [lipgloss.Border]
	// leaves the prompt border unchanged.
	Border lipgloss.Border

	// BorderColor is the focused field left-border foreground. The
	// zero value leaves the prompt border color unchanged.
	BorderColor color.Color
}

// IsZero reports whether k leaves every prompt knob unset.
func (k PromptKnobs) IsZero() bool {
	return k.SelectedPrefix == "" &&
		k.UnselectedPrefix == "" &&
		k.Border == (lipgloss.Border{}) &&
		colorIsUnset(k.BorderColor)
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
	if !colorIsUnset(k.BorderColor) {
		p.BorderColor = k.BorderColor
	}
	return p
}

// IsZero reports whether every field on p is unset. The framework uses
// it to fall back to [PromptFromTokens] when a palette omits Prompt.
//
// A Prompt with only [Border] or [BorderColor] set is non-zero. The
// check inspects fields individually because [lipgloss.Style] is not
// comparable.
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
		p.Border == (lipgloss.Border{}) &&
		colorIsUnset(p.BorderColor)
}

// styleIsZero reports whether s has any overlay-relevant attribute set
// (fg, bg, modifiers, literal payload). Frame/width-only styles count
// as zero because Prompt overlays never read those fields.
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

// Huh returns a [huh.Theme] derived from this Prompt, starting from
// [huh.ThemeBase] and overlaying set fields. Zero-style fields inherit
// the base.
//
// Blurred styling reuses Focused for a stable cross-state look; authors
// who need separate blurred styles set [Palette.Huh] directly. Border
// and BorderColor apply to Focused.Base and Focused.Card when set;
// Blurred fields always use a hidden border so the focused left border
// acts as the focus indicator.
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

		if !colorIsUnset(p.BorderColor) {
			s.Focused.Base = s.Focused.Base.BorderForeground(p.BorderColor)
			s.Focused.Card = s.Focused.Card.BorderForeground(p.BorderColor)
		}

		// Mirror focused styling onto blurred so the form keeps a
		// consistent look between active and inactive states. The
		// previous huhStyle pipeline let authors customize blurred
		// independently via $inherit; the closed Prompt surface
		// trades that knob away for simpler authoring. Authors who
		// need separate blurred styling reach for Palette.Huh.
		s.Blurred = s.Focused
		s.Blurred.Base = s.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())
		s.Blurred.Card = s.Blurred.Card.BorderStyle(lipgloss.HiddenBorder())
		// huh's own themes clear these two indicators on Blurred so an
		// inactive multi-select's pagination arrows don't linger;
		// mirror that here since the copy above would otherwise carry
		// the focused arrows over unchanged.
		s.Blurred.NextIndicator = lipgloss.NewStyle()
		s.Blurred.PrevIndicator = lipgloss.NewStyle()

		if p.Border != (lipgloss.Border{}) {
			s.Focused.Base = s.Focused.Base.BorderStyle(p.Border)
			s.Focused.Card = s.Focused.Card.BorderStyle(p.Border)
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

// PromptFromTokens derives a [Prompt] from a per-token style map. It is
// the fallback when a [Palette] declares neither [Palette.Prompt] nor
// [Palette.Huh].
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
		BorderColor:      colorFromStyle(tokens[AccentPrimary]),
	}
}

// overlayStyle copies set attributes from src onto dst. Unset src
// fields preserve dst so partial Prompt declarations layer on huh base.
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

// colorFromStyle returns the style foreground when set.
func colorFromStyle(s lipgloss.Style) color.Color {
	if fg := s.GetForeground(); !isNoColor(fg) {
		return fg
	}
	return nil
}

// colorIsUnset reports whether c is absent for Prompt border slots.
func colorIsUnset(c color.Color) bool {
	return c == nil || isNoColor(c)
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
