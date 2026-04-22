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
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// TestWithThemeOverrideTweaksOneSlot verifies the canonical use of
// the override Option: pick a built-in theme, then redirect one token
// without writing a custom theme.
func TestWithThemeOverrideTweaksOneSlot(t *testing.T) {
	t.Parallel()

	magenta := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)
	app, err := New("myctl",
		WithTheme(theme.Default),
		WithThemeOverride(theme.StatusError, magenta),
	)
	require.NoError(t, err)
	assert.Equal(t, magenta.GetForeground(),
		app.Theme().Style(theme.StatusError).GetForeground(),
		"WithThemeOverride should redirect the StatusError slot")
}

// TestWithStrictThemeRequirementsBlocksIncompleteTheme verifies the
// strict-mode opt-in surfaces missing-token diagnostics as a hard
// [*ConfigErrors] entry instead of writing them to stderr. The
// custom theme below covers no tokens, so every core consumer
// reports a miss.
func TestWithStrictThemeRequirementsBlocksIncompleteTheme(t *testing.T) {
	t.Parallel()

	bare := theme.Theme{
		Name:    "bare",
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Tokens: map[theme.Token]lipgloss.Style{}},
		},
	}
	_, err := New("myctl",
		WithCustomTheme(bare),
		WithStrictThemeRequirements(),
	)
	require.Error(t, err)
	assert.ErrorContains(t, err, `theme "bare"`)
	assert.ErrorContains(t, err, "missing tokens required by")
}

// TestWithoutStrictThemeRequirementsWarnsOnIncompleteTheme is the
// flip side: the same bare theme installs cleanly without the strict
// option. The diagnostic still fires — we capture stderr to verify —
// but [New] returns no error so production CLIs that intentionally
// ship a sparse theme keep working.
func TestWithoutStrictThemeRequirementsWarnsOnIncompleteTheme(t *testing.T) {
	t.Parallel()

	bare := theme.Theme{
		Name:    "bare",
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Tokens: map[theme.Token]lipgloss.Style{}},
		},
	}
	io, _, _, stderr := testIO()
	_, err := New("myctl",
		WithIO(io),
		WithCustomTheme(bare),
	)
	require.NoError(t, err, "default mode should warn, not error")
	assert.Contains(t, stderr.String(), "warning: theme \"bare\" is missing tokens",
		"stderr should carry the warning when strict mode is off")
}

// TestWithThemeOverridesAppliesAllInOrder covers the batch helper
// and its order-stable composition rule: later overrides for the
// same token win, distinct overrides stack.
func TestWithThemeOverridesAppliesAllInOrder(t *testing.T) {
	t.Parallel()

	red := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	blue := lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF"))
	bold := lipgloss.NewStyle().Bold(true)

	app, err := New("myctl",
		WithTheme(theme.Default),
		WithThemeOverrides(
			theme.SetToken(theme.StatusError, red),  // overridden below
			theme.SetToken(theme.StatusError, blue), // wins
			theme.SetToken(theme.StatusInfo, bold),  // stacks
		),
	)
	require.NoError(t, err)
	rt := app.Theme()
	assert.Equal(t, blue.GetForeground(), rt.Style(theme.StatusError).GetForeground(),
		"later override for the same token should win")
	assert.True(t, rt.Style(theme.StatusInfo).GetBold(),
		"distinct overrides should stack")
}

// TestFinalizeAcceptsTwoAppsWithDifferentOwnedChromaSameName verifies
// that the chroma global registry's collision risk is gone: two Apps
// in the same process can install different chroma.Style values under
// the same theme name without finalize complaining, because finalize
// no longer plants anything in the chroma registry. [Context.highlight]
// reaches the owned style directly via [theme.ResolvedTheme.Chroma].
//
// This test is sequential because the chroma global registry is still
// read once at [theme.Theme.Resolve] time when a palette declares only
// a chroma name, even though we no longer write to it.
func TestFinalizeAcceptsTwoAppsWithDifferentOwnedChromaSameName(t *testing.T) {
	const sharedName = "nabat-test-no-collision-" + "S8"

	ownedA, err := chroma.NewStyle(sharedName, chroma.StyleEntries{
		chroma.NameFunction: "#FF00FF",
	})
	require.NoError(t, err)
	ownedB, err := chroma.NewStyle(sharedName, chroma.StyleEntries{
		chroma.NameFunction: "#00FF00",
	})
	require.NoError(t, err)

	customA := theme.Theme{
		Name:    sharedName,
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Chroma: ownedA},
		},
	}
	customB := theme.Theme{
		Name:    sharedName,
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Chroma: ownedB},
		},
	}

	appA, err := New("myctl", WithCustomTheme(customA))
	require.NoError(t, err)
	appB, err := New("myctl", WithCustomTheme(customB))
	require.NoError(t, err)

	// Each App resolves to its own chroma.Style — finalize did not
	// register either with the chroma registry, so they cannot
	// shadow each other across App boundaries.
	assert.Same(t, ownedA, appA.Theme().Chroma())
	assert.Same(t, ownedB, appB.Theme().Chroma())
}
