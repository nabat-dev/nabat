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

// TestRequireBuildsCanonicalRequirement covers the constructor: the
// result carries the consumer name and tokens passed in.
func TestRequireBuildsCanonicalRequirement(t *testing.T) {
	t.Parallel()

	got := theme.Require("logging extension",
		theme.StatusInfo, theme.StatusWarning, theme.StatusError,
	)
	assert.Equal(t, "logging extension", got.Consumer)
	assert.Equal(t, []theme.Token{
		theme.StatusInfo, theme.StatusWarning, theme.StatusError,
	}, got.Tokens)
}

// TestHasTokenDirectMiss verifies the negative path: a token unset
// on the palette and not reachable through the alias chain is
// reported as not covered.
func TestHasTokenDirectMiss(t *testing.T) {
	t.Parallel()

	rt := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Tokens: map[theme.Token]lipgloss.Style{}},
		},
	}.Resolve(theme.Capabilities{})
	assert.False(t, rt.HasToken(theme.StatusError),
		"an unset token with no alias coverage should report as missing")
}

// TestHasTokenAliasReached verifies the positive path through the
// default alias chain: setting only TextMuted should cover
// ListEnumerator (via DefaultAliases[ListEnumerator] = TextMuted).
func TestHasTokenAliasReached(t *testing.T) {
	t.Parallel()

	rt := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.TextMuted: lipgloss.NewStyle(),
				},
			},
		},
	}.Resolve(theme.Capabilities{})
	assert.True(t, rt.HasToken(theme.ListEnumerator),
		"alias chain should mark ListEnumerator as covered when TextMuted is set")
}

// TestMissingTokensReturnsSortedList covers the missing-tokens
// helper: every uncovered token shows up in the result, sorted
// lexically so error messages are deterministic.
func TestMissingTokensReturnsSortedList(t *testing.T) {
	t.Parallel()

	rt := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusInfo: lipgloss.NewStyle(),
				},
			},
		},
	}.Resolve(theme.Capabilities{})

	got := rt.MissingTokens(theme.Require("x",
		theme.StatusError, theme.StatusInfo, theme.StatusWarning,
	))
	assert.Equal(t, []theme.Token{theme.StatusError, theme.StatusWarning}, got,
		"missing tokens must be sorted and exclude covered ones")
}

// TestCheckRequirementsErrorListsConsumers verifies the diagnostic
// shape: the joined error names every consumer with at least one
// missing token and lists the missing tokens per line.
func TestCheckRequirementsErrorListsConsumers(t *testing.T) {
	t.Parallel()

	// Set only StatusInfo so several consumers report misses.
	rt := theme.Theme{
		Name: "tiny",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusInfo: lipgloss.NewStyle(),
				},
			},
		},
	}.Resolve(theme.Capabilities{})

	err := rt.CheckRequirements([]theme.Requirement{
		theme.Require("alpha", theme.StatusError, theme.StatusWarning),
		theme.Require("beta", theme.TextTitle),
		theme.Require("delta", theme.StatusInfo), // covered, omitted
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `theme "tiny"`)
	assert.Contains(t, err.Error(), "alpha:")
	assert.Contains(t, err.Error(), "beta:")
	assert.NotContains(t, err.Error(), "delta:",
		"satisfied consumers should not appear in the diagnostic")
}

// TestCheckRequirementsAllSatisfied returns nil when every consumer
// is covered — the framework path that lets construction succeed
// silently.
func TestCheckRequirementsAllSatisfied(t *testing.T) {
	t.Parallel()

	rt := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusInfo:  lipgloss.NewStyle(),
					theme.StatusError: lipgloss.NewStyle(),
				},
			},
		},
	}.Resolve(theme.Capabilities{})
	assert.NoError(t, rt.CheckRequirements([]theme.Requirement{
		theme.Require("c1", theme.StatusInfo),
		theme.Require("c2", theme.StatusError),
	}))
}

// TestCoreRequirementsCoversWellKnownTokens guards the rule that the
// framework's own consumers declare every well-known token. Drift
// here means a new core consumer (a new Status*, Text*, Table*,
// List*, Tree*) was added to the framework but not to
// CoreRequirements — and its missing-token diagnostic would silently
// stop catching regressions.
func TestCoreRequirementsCoversWellKnownTokens(t *testing.T) {
	t.Parallel()

	covered := map[theme.Token]bool{}
	for _, req := range theme.CoreRequirements() {
		for _, tok := range req.Tokens {
			covered[tok] = true
		}
	}
	for _, want := range []theme.Token{
		theme.StatusSuccess, theme.StatusWarning, theme.StatusError, theme.StatusInfo,
		theme.TextPrimary, theme.TextSecondary, theme.TextTitle, theme.TextLink, theme.TextMuted,
		theme.AccentPrimary, theme.CodeSurface,
		theme.TableBorder, theme.TableHeader, theme.TableCell,
		theme.ListItem, theme.ListEnumerator,
		theme.TreeItem, theme.TreeEnumerator,
	} {
		assert.Truef(t, covered[want], "well-known token %s missing from CoreRequirements()", want)
	}
}
