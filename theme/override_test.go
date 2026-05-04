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
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"

	"nabat.dev/theme"
)

// TestThemeWithPreservesPrompt is a regression test for a clonePalette
// bug where Palette.Prompt was silently dropped during Theme.With(),
// causing themes with a custom promptStyle to fall back to the
// token-derived default after any override.
func TestThemeWithPreservesPrompt(t *testing.T) {
	t.Parallel()

	customPrompt := theme.Prompt{
		Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000")),
		SelectedPrefix: lipgloss.NewStyle().SetString("→ "),
		Border:         lipgloss.RoundedBorder(),
	}
	src := theme.Theme{
		Name: "src",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusSuccess: lipgloss.NewStyle().Bold(true),
				},
				Prompt: customPrompt,
			},
		},
	}

	derived := src.With(theme.SetToken(theme.StatusError, lipgloss.NewStyle().Italic(true)))

	got := derived.Variants[theme.VariantDark].Prompt
	assert.Equal(t, customPrompt.Title, got.Title, "Title style dropped by clonePalette")
	assert.Equal(t, customPrompt.SelectedPrefix, got.SelectedPrefix, "SelectedPrefix dropped by clonePalette")
	assert.Equal(t, customPrompt.Border, got.Border, "Border dropped by clonePalette")
	assert.False(t, got.IsZero(), "derived palette's Prompt should not be zero")
}

// TestThemeWithDoesNotMutateReceiver verifies that overrides applied
// via With produce a fresh Theme; the receiver's variants must be
// untouched so the catalog's cached Theme stays clean for other
// callers.
func TestThemeWithDoesNotMutateReceiver(t *testing.T) {
	t.Parallel()

	src := theme.Theme{
		Name: "src",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusSuccess: lipgloss.NewStyle().Bold(true),
				},
			},
		},
	}

	override := lipgloss.NewStyle().Italic(true)
	_ = src.With(theme.SetToken(theme.StatusSuccess, override))

	got := src.Variants[theme.VariantDark].Tokens[theme.StatusSuccess]
	assert.True(t, got.GetBold(), "receiver's tokens were mutated by With")
	assert.False(t, got.GetItalic(), "receiver's tokens were mutated by With")
}

// TestThemeWithAppliesAcrossVariants verifies that a single override
// reaches every declared variant — the documented "tweak this one
// slot regardless of which variant resolves at runtime" contract.
func TestThemeWithAppliesAcrossVariants(t *testing.T) {
	t.Parallel()

	src := theme.Theme{
		Name:    "src",
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark:  {Tokens: map[theme.Token]lipgloss.Style{}},
			theme.VariantLight: {Tokens: map[theme.Token]lipgloss.Style{}},
		},
	}

	override := lipgloss.NewStyle().Foreground(lipgloss.Color("#123456"))
	derived := src.With(theme.SetToken(theme.StatusError, override))

	for v, palette := range derived.Variants {
		assert.Equal(t, override, palette.Tokens[theme.StatusError],
			"override missing in variant %q", v)
	}
}
