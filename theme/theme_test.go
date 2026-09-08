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
	"errors"
	"reflect"
	"strings"
	"testing"

	"charm.land/glamour/v2/ansi"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/alecthomas/chroma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// sameFunc compares two list enumerator function pointers. The
// enumerator type is `func(items list.Items, i int) string`; equal
// pointers identify them as the same closure.
func sameFunc(a, b any) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

// TestThemeResolveSingleVariantPicksOnlyKey covers the "single
// variant" shortcut: themes that ship one Palette resolve to it
// regardless of [Theme.Default].
func TestThemeResolveSingleVariantPicksOnlyKey(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Name: "dracula",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusSuccess: lipgloss.NewStyle().Bold(true),
				},
			},
		},
	}
	got := th.Resolve(theme.Capabilities{})
	assert.Equal(t, "dracula", got.Name())
	assert.True(t, got.Style(theme.StatusSuccess).GetBold())
	assert.Equal(t, theme.VariantDark, got.Variant())
}

// TestThemeResolveDefaultPicksDeclaredVariant covers the multi-
// variant path: when more than one Palette is declared, [Theme.Default]
// drives the pick.
func TestThemeResolveDefaultPicksDeclaredVariant(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Name:    "duo",
		Default: theme.VariantLight,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark:  {Tokens: map[theme.Token]lipgloss.Style{theme.StatusSuccess: lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))}},
			theme.VariantLight: {Tokens: map[theme.Token]lipgloss.Style{theme.StatusSuccess: lipgloss.NewStyle().Foreground(lipgloss.Color("#007F00"))}},
		},
	}
	got := th.Resolve(theme.Capabilities{})
	assert.Equal(t, theme.VariantLight, got.Variant())
	assert.Equal(t, lipgloss.Color("#007F00"), got.Style(theme.StatusSuccess).GetForeground())
}

// TestStyleReturnsZeroForUnsetToken locks the "unset token returns
// zero style" contract. Consumers depend on the zero style rendering
// as the terminal default rather than (Style, ok) tuples.
func TestStyleReturnsZeroForUnsetToken(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Tokens: map[theme.Token]lipgloss.Style{}},
		},
	}
	got := th.Resolve(theme.Capabilities{})
	assert.Equal(t, lipgloss.Style{}, got.Style(theme.StatusError))
}

// TestStyleFollowsDefaultAliasChain verifies that a token unset on
// the palette resolves through [theme.DefaultAliases]. The default
// chain says ListEnumerator -> TextMuted, so a palette that sets
// only TextMuted should produce the same style for ListEnumerator.
func TestStyleFollowsDefaultAliasChain(t *testing.T) {
	t.Parallel()

	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{theme.TextMuted: muted},
			},
		},
	}
	rt := th.Resolve(theme.Capabilities{})
	assert.Equal(t, muted.GetForeground(),
		rt.Style(theme.ListEnumerator).GetForeground(),
		"ListEnumerator should fall through to TextMuted by default")
	assert.Equal(t, muted.GetForeground(),
		rt.Style(theme.TreeEnumerator).GetForeground(),
		"TreeEnumerator should fall through (TreeEnumerator -> ListEnumerator -> TextMuted)")
	assert.Equal(t, muted.GetForeground(),
		rt.Style(theme.TableBorder).GetForeground(),
		"TableBorder should fall through to TextMuted by default")
}

// TestStyleAppliesPerPaletteAliasOverride verifies that a manifest's
// per-palette aliases entry overrides the framework default for that
// key. The example below points list.item at text.secondary instead of
// text.primary (the default).
func TestStyleAppliesPerPaletteAliasOverride(t *testing.T) {
	t.Parallel()

	secondary := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	primary := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.TextSecondary: secondary,
					theme.TextPrimary:   primary,
				},
				Aliases: map[theme.Token]theme.Token{
					theme.ListItem: theme.TextSecondary,
				},
			},
		},
	}
	rt := th.Resolve(theme.Capabilities{})
	assert.Equal(t, secondary.GetForeground(), rt.Style(theme.ListItem).GetForeground(),
		"per-palette alias override should redirect ListItem to TextSecondary")
}

// TestResolveDetectsAliasCycle locks the cycle-detection invariant:
// an alias chain that loops surfaces an error from [Theme.ResolveErr]
// rather than silently looping forever at lookup time. The
// per-call seen set in [ResolvedTheme.Style] is a defense in depth;
// the construction-time check is the primary signal.
func TestResolveDetectsAliasCycle(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Name: "loopy",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.TextMuted: lipgloss.NewStyle(),
				},
				Aliases: map[theme.Token]theme.Token{
					theme.ListEnumerator: theme.TreeEnumerator,
					theme.TreeEnumerator: theme.ListEnumerator,
				},
			},
		},
	}
	_, err := th.ResolveErr(theme.Capabilities{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "loopy")
	assert.ErrorContains(t, err, "cycle")
}

// TestStyleSurvivesAliasCycleAtLookup is the safety-net mirror of
// TestResolveDetectsAliasCycle: even if a malformed map sneaks past
// construction validation, ResolvedTheme.Style must not loop forever.
// The per-call seen set bottoms out and returns the zero style.
func TestStyleSurvivesAliasCycleAtLookup(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Aliases: map[theme.Token]theme.Token{
					theme.ListItem: theme.TreeItem,
					theme.TreeItem: theme.ListItem,
				},
			},
		},
	}
	rt, _ := th.ResolveErr(theme.Capabilities{}) //nolint:errcheck // err is expected here; we still want a usable rt for the lookup test below
	got := rt.Style(theme.ListItem)
	assert.Equal(t, lipgloss.Style{}, got, "Style must terminate and return zero on a cyclic chain")
}

// TestResolveAppliesHuhFromTokensWhenNil exercises the
// [HuhFromTokens] fallback: a Palette with no [Palette.Huh] derives
// a huh.Theme from its tokens so prompts stay on-palette out of the
// box.
func TestResolveAppliesHuhFromTokensWhenNil(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.AccentPrimary: lipgloss.NewStyle().Foreground(lipgloss.Color("#D4A853")),
					theme.TextSecondary: lipgloss.NewStyle().Foreground(lipgloss.Color("#D5CDC2")),
				},
			},
		},
	}
	got := th.Resolve(theme.Capabilities{}).Huh()
	require.NotNil(t, got, "Huh fallback should populate a non-nil huh.Theme")
}

// TestResolveAppliesGlamourPresetWhenAllNil exercises the chain:
// when neither Glamour nor GlamourFor nor GlamourName is set, the
// framework picks via [GlamourPreset] for the variant + capabilities
// and folds the named preset into the resolved [Glamour] value.
func TestResolveAppliesGlamourPresetWhenAllNil(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		variant theme.Variant
		caps    theme.Capabilities
		// For known glamour presets ("dark", "light", "notty"),
		// the resolved Glamour() must be non-nil. For "" (unknown
		// or not selectable), it must be nil.
		wantNil bool
	}{
		{"notty variant always picks notty", theme.VariantNoTTY, theme.Capabilities{Dark: true, Interactive: true}, false},
		{"non-interactive picks notty", theme.VariantDark, theme.Capabilities{Dark: true}, false},
		{"interactive dark picks dark", theme.VariantDark, theme.Capabilities{Dark: true, Interactive: true}, false},
		{"interactive light picks light", theme.VariantLight, theme.Capabilities{Interactive: true}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			th := theme.Theme{
				Variants: map[theme.Variant]theme.Palette{c.variant: {}},
			}
			got := th.Resolve(c.caps).Glamour()
			if c.wantNil {
				assert.Nil(t, got)
			} else {
				assert.NotNil(t, got, "preset for %s+%v should resolve to a non-nil glamour config", c.variant, c.caps)
			}
		})
	}
}

// TestResolveAppliesTokenDerivedChromaWhenUnset exercises the chroma
// fallback: empty Chroma + empty ChromaName resolves through
// [ChromaFromTokens].
func TestResolveAppliesTokenDerivedChromaWhenUnset(t *testing.T) {
	t.Parallel()

	cases := []theme.Variant{theme.VariantDark, theme.VariantLight, theme.VariantNoTTY}
	for _, c := range cases {
		t.Run(string(c), func(t *testing.T) {
			t.Parallel()
			th := theme.Theme{
				Variants: map[theme.Variant]theme.Palette{
					c: {
						Tokens: map[theme.Token]lipgloss.Style{
							theme.TextPrimary: lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")),
							theme.CodeSurface: lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")),
						},
					},
				},
			}
			got := th.Resolve(theme.Capabilities{Interactive: true}).Chroma()
			require.NotNil(t, got)
		})
	}
}

// TestResolveDefaultsForListEnumeratorAndTableBorder verifies that
// zero [Theme.ListEnum] and zero [Theme.TableBorder] resolve to
// [list.Bullet] and [lipgloss.NormalBorder] respectively — the
// framework defaults consumers expect when a theme cares only about
// colors.
func TestResolveDefaultsForListEnumeratorAndTableBorder(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{theme.VariantDark: {}},
	}
	got := th.Resolve(theme.Capabilities{})
	require.NotNil(t, got.ListEnumerator())
	assert.True(t, sameFunc(list.Bullet, got.ListEnumerator()),
		"default list enumerator should be list.Bullet")
	assert.Equal(t, lipgloss.NormalBorder(), got.TableBorder())
}

// TestResolveOverridesListEnumeratorAndTableBorder verifies that
// values explicitly set on [Theme.ListEnum] / [Theme.TableBorder]
// flow through to the resolved theme without being clobbered by the
// framework defaults.
func TestResolveOverridesListEnumeratorAndTableBorder(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		ListEnum:    list.Dash,
		TableBorder: lipgloss.RoundedBorder(),
		Variants:    map[theme.Variant]theme.Palette{theme.VariantDark: {}},
	}
	got := th.Resolve(theme.Capabilities{})
	require.NotNil(t, got.ListEnumerator())
	assert.True(t, sameFunc(list.Dash, got.ListEnumerator()),
		"list enumerator override should be list.Dash")
	assert.Equal(t, lipgloss.RoundedBorder(), got.TableBorder())
}

// TestZeroResolvedThemeIsSafeToQuery confirms that the zero
// [ResolvedTheme] (used as a fallback by tests, by code that
// constructs ResolvedTheme directly, or by an [App] that never had
// a theme installed) supports every accessor without panicking.
func TestZeroResolvedThemeIsSafeToQuery(t *testing.T) {
	t.Parallel()

	var r theme.ResolvedTheme
	assert.Equal(t, "", r.Name())
	assert.Equal(t, theme.VariantUnset, r.Variant())
	assert.Equal(t, lipgloss.Style{}, r.Style(theme.StatusSuccess))
	assert.Empty(t, r.Tokens())
	assert.Nil(t, r.Chroma())
	assert.Nil(t, r.Glamour())
	assert.Nil(t, r.Huh())
}

// TestResolveErrSurfacesGlamourForFailure exercises the
// [Palette.GlamourFor] error path: when the inline glamour callback
// returns an error, [Theme.ResolveErr] surfaces it (wrapped with the
// theme name) while still producing a usable ResolvedTheme.
func TestResolveErrSurfacesGlamourForFailure(t *testing.T) {
	t.Parallel()

	want := errors.New("inline glamour broke")
	th := theme.Theme{
		Name: "broken",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				GlamourFor: func(theme.Capabilities) (*ansi.StyleConfig, error) {
					return nil, want
				},
			},
		},
	}

	rt, err := th.ResolveErr(theme.Capabilities{Dark: true, Interactive: true})
	require.Error(t, err)
	assert.ErrorIs(t, err, want, "ResolveErr should wrap the GlamourFor error")
	assert.Contains(t, err.Error(), "broken", "error should name the theme")

	// The slot stays empty so consumers fall through to glamour's
	// own default rather than crashing.
	assert.Nil(t, rt.Glamour())
}

// TestResolveErrIsNilOnHappyPath confirms ResolveErr returns no
// error when every field is well-formed; ResolveErr must mirror
// Resolve when there are no per-Palette failures so callers can
// always use ResolveErr without losing diagnostic value.
func TestResolveErrIsNilOnHappyPath(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Name: "fine",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Tokens: map[theme.Token]lipgloss.Style{theme.StatusSuccess: lipgloss.NewStyle()}},
		},
	}
	_, err := th.ResolveErr(theme.Capabilities{})
	assert.NoError(t, err)
}

// TestThemeValidateFlagsBadDefault locks the [Theme.Validate]
// invariant: a non-empty [Theme.Default] must reference a declared
// variant. The catalog would surface this at registry-load time;
// programmatic themes opt in by calling Validate themselves.
func TestThemeValidateFlagsBadDefault(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Name:    "wrong",
		Default: theme.VariantLight,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {},
		},
	}
	err := th.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "wrong")
	assert.ErrorContains(t, err, "Default")
}

// TestThemeValidateAllowsZeroVariantsZeroDefault verifies the empty
// theme is structurally valid; zero Variants + zero Default just
// resolves to the zero ResolvedTheme.
func TestThemeValidateAllowsZeroVariantsZeroDefault(t *testing.T) {
	t.Parallel()

	assert.NoError(t, theme.Theme{}.Validate())
}

// TestThemeImplementsResolver pins the type-system relationship: every
// [Theme] satisfies [Resolver] via its Resolve method, which is what
// lets [WithCustomTheme] accept either form without a union type.
func TestThemeImplementsResolver(t *testing.T) {
	t.Parallel()

	var r theme.Resolver = theme.Theme{Variants: map[theme.Variant]theme.Palette{theme.VariantDark: {}}}
	rt := r.Resolve(theme.Capabilities{})
	assert.NotNil(t, rt)
}

// TestTokensReturnsAllSetTokens verifies the diagnostic accessor
// surfaces every token the theme explicitly set; the order is not
// guaranteed (map iteration), so the test uses ElementsMatch.
func TestTokensReturnsAllSetTokens(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusSuccess: lipgloss.NewStyle(),
					theme.StatusError:   lipgloss.NewStyle(),
				},
			},
		},
	}
	got := th.Resolve(theme.Capabilities{}).Tokens()
	assert.ElementsMatch(t, []theme.Token{theme.StatusSuccess, theme.StatusError}, got)
}

// TestChromaPrefersOwnedStyle verifies the resolved-chroma cascade:
// an owned [*chroma.Style] wins over a registered name.
func TestChromaPrefersOwnedStyle(t *testing.T) {
	t.Parallel()

	owned := &chroma.Style{}
	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Chroma: owned, ChromaName: "monokai"},
		},
	}
	rt := th.Resolve(theme.Capabilities{})
	assert.Same(t, owned, rt.Chroma())
}

// TestChromaFallsBackToRegistry verifies the registry path: only
// ChromaName set resolves through chromastyles.Get during
// [Theme.Resolve].
func TestChromaFallsBackToRegistry(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {ChromaName: "monokai"},
		},
	}
	got := th.Resolve(theme.Capabilities{}).Chroma()
	require.NotNil(t, got)
	assert.Equal(t, "monokai", got.Name)
}

// TestGlamourPrefersOwnedStyle mirrors the chroma test for glamour:
// an owned ansi.StyleConfig wins over a named preset.
func TestGlamourPrefersOwnedStyle(t *testing.T) {
	t.Parallel()

	owned := &ansi.StyleConfig{}
	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Glamour: owned, GlamourName: "dark"},
		},
	}
	rt := th.Resolve(theme.Capabilities{})
	assert.Same(t, owned, rt.Glamour())
}

// TestGlamourFallsBackToPreset verifies the named-preset branch:
// only GlamourName set resolves through glamour's DefaultStyles
// during [Theme.Resolve].
func TestGlamourFallsBackToPreset(t *testing.T) {
	t.Parallel()

	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {GlamourName: "dark"},
		},
	}
	got := th.Resolve(theme.Capabilities{Interactive: true}).Glamour()
	require.NotNil(t, got)
}

// TestPaletteCopiedDefensively confirms that [Theme.Resolve] copies
// the Palette token map; mutating the source map after Resolve must
// not affect the resolved theme.
func TestPaletteCopiedDefensively(t *testing.T) {
	t.Parallel()

	src := map[theme.Token]lipgloss.Style{
		theme.StatusSuccess: lipgloss.NewStyle().Bold(true),
	}
	th := theme.Theme{
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {Tokens: src},
		},
	}
	got := th.Resolve(theme.Capabilities{})

	src[theme.StatusSuccess] = lipgloss.NewStyle().Italic(true)
	assert.True(t, got.Style(theme.StatusSuccess).GetBold(),
		"Resolve should snapshot tokens; later mutation leaked")
	assert.False(t, got.Style(theme.StatusSuccess).GetItalic(),
		"Resolve should snapshot tokens; later mutation leaked")
}

// TestTokenConstantsAreDottedLowercase locks the naming convention
// for the well-known token set. Drift here would break manifest
// authors and consumers that hardcode the names.
func TestTokenConstantsAreDottedLowercase(t *testing.T) {
	t.Parallel()

	for _, tok := range []theme.Token{
		theme.StatusSuccess, theme.StatusWarning, theme.StatusError, theme.StatusInfo,
		theme.TextPrimary, theme.TextSecondary, theme.TextTitle, theme.TextLink, theme.TextMuted,
		theme.AccentPrimary, theme.CodeSurface,
		theme.TableBorder, theme.TableHeader, theme.TableCell,
		theme.ListItem, theme.ListEnumerator,
		theme.TreeItem, theme.TreeEnumerator,
	} {
		s := string(tok)
		assert.NotEmpty(t, s)
		assert.Equal(t, strings.ToLower(s), s, "token %q should be lowercase", s)
		assert.Contains(t, s, ".", "token %q should be dotted", s)
	}
}
