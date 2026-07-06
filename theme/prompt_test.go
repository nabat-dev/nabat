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
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

func TestPromptFromTokensSetsBorderColor(t *testing.T) {
	t.Parallel()

	tokens := map[theme.Token]lipgloss.Style{
		theme.AccentPrimary: lipgloss.NewStyle().Foreground(lipgloss.Color("#C89B3C")),
	}
	got := theme.PromptFromTokens(tokens)
	require.NotNil(t, got.BorderColor)
	assert.True(t, sameColor(got.BorderColor, lipgloss.Color("#C89B3C")),
		"BorderColor should come from accent.primary foreground")
}

func TestPromptHuhAppliesBorderForeground(t *testing.T) {
	t.Parallel()

	p := theme.Prompt{
		BorderColor: lipgloss.Color("#AABBCC"),
	}
	s := p.Huh().Theme(false)
	assert.True(t, sameColor(s.Focused.Base.GetBorderLeftForeground(), lipgloss.Color("#AABBCC")),
		"focused.base border foreground should match BorderColor")
	assert.True(t, sameColor(s.Focused.Card.GetBorderLeftForeground(), lipgloss.Color("#AABBCC")),
		"focused.card border foreground should match BorderColor")
}

func TestPromptHuhBlurredAlwaysHiddenBorder(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		prompt theme.Prompt
	}{
		{
			name: "token derived border color only",
			prompt: theme.PromptFromTokens(map[theme.Token]lipgloss.Style{
				theme.AccentPrimary: lipgloss.NewStyle().Foreground(lipgloss.Color("#C89B3C")),
			}),
		},
		{
			name: "custom border shape",
			prompt: theme.Prompt{
				BorderColor: lipgloss.Color("#C89B3C"),
				Border:      lipgloss.RoundedBorder(),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := c.prompt.Huh().Theme(false)
			assert.Equal(t, lipgloss.HiddenBorder(), s.Blurred.Base.GetBorderStyle(),
				"blurred.base border should always be hidden")
			assert.Equal(t, lipgloss.HiddenBorder(), s.Blurred.Card.GetBorderStyle(),
				"blurred.card border should always be hidden")
		})
	}
}

func TestPromptHuhBlurredClearsNavIndicators(t *testing.T) {
	t.Parallel()

	p := theme.Prompt{
		Selector: lipgloss.NewStyle().Foreground(lipgloss.Color("#3EB0CC")),
	}
	s := p.Huh().Theme(false)
	assert.True(t, sameColor(s.Focused.NextIndicator.GetForeground(), lipgloss.Color("#3EB0CC")),
		"focused.nextIndicator should pick up Selector color")
	assert.True(t, isNoColorStyle(s.Blurred.NextIndicator),
		"blurred.nextIndicator should be cleared, not mirror focused")
	assert.True(t, isNoColorStyle(s.Blurred.PrevIndicator),
		"blurred.prevIndicator should be cleared, not mirror focused")
}

func isNoColorStyle(s lipgloss.Style) bool {
	_, isNoColor := s.GetForeground().(lipgloss.NoColor)
	return isNoColor
}

func TestPromptKnobsApplyBorderColor(t *testing.T) {
	t.Parallel()

	knobs := theme.PromptKnobs{
		BorderColor: lipgloss.Color("#112233"),
	}
	p := knobs.Apply(theme.Prompt{})
	assert.True(t, sameColor(p.BorderColor, lipgloss.Color("#112233")))
}
