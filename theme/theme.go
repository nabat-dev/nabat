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
	"maps"

	"charm.land/glamour/v2/ansi"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/alecthomas/chroma/v2"

	glamourstyles "charm.land/glamour/v2/styles"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

// Variant is a theme's intended luminance or TTY context (dark, light,
// or notty). Runtime diagnostics and a future "--theme-variant" override
// read this slot.
//
// The zero value ([VariantUnset]) means the theme did not declare a
// target; consumers should treat it as compatible with every
// [Capabilities] snapshot.
type Variant string

// Variant constants match the manifest "variant" field and
// [Theme.Default] / [Theme.Variants] keys.
const (
	// VariantUnset is the zero value: no target variant declared.
	VariantUnset Variant = ""

	// VariantDark indicates a palette designed for dark terminals.
	VariantDark Variant = "dark"

	// VariantLight indicates a palette designed for light terminals.
	VariantLight Variant = "light"

	// VariantNoTTY indicates a palette designed for non-TTY output.
	VariantNoTTY Variant = "notty"
)

// Theme is declarative CLI styling: [Palette] entries per [Variant],
// a default variant, and cross-variant defaults. [Theme.Resolve] returns
// an immutable [ResolvedTheme]. Safe to copy by value, but the copy shares
// the Variants map; use [Theme.Clone] or [nabat.WithThemeOverride] to
// tweak. Implements [Resolver].
type Theme struct {
	// Name identifies the theme in errors, the catalog key, and
	// [ResolvedTheme.Name]. Built-ins require it; programmatic themes
	// may leave it empty.
	Name string

	// Variants maps each declared [Variant] to its [Palette].
	// An empty map yields a zero [ResolvedTheme] from [Theme.Resolve]
	// (every Style call returns the zero [lipgloss.Style]).
	Variants map[Variant]Palette

	// Default is the variant [Theme.Resolve] picks when [Capabilities]
	// do not match a declared variant. For a single-variant theme the
	// lone key wins even when Default is [VariantUnset].
	Default Variant

	// ListEnum is the default enumerator for list output. Nil resolves
	// to [list.Bullet].
	ListEnum list.Enumerator

	// TableBorder is the default table border. The zero
	// [lipgloss.Border] resolves to [lipgloss.NormalBorder].
	TableBorder lipgloss.Border

	// PromptKnobs are theme-wide prompt settings applied to
	// token-derived and palette prompt styles.
	PromptKnobs PromptKnobs
}

// Palette is the per-variant style data a [Theme] carries.
// [Theme.Resolve] picks one palette and fills nil or empty cascade
// slots with framework defaults.
//
// Chroma and glamour use a three-way cascade (owned value, registered
// name, capability default) folded into one value at resolve time.
type Palette struct {
	// Tokens is the per-token style map. Missing tokens fall through
	// the alias chain ([Aliases] over [DefaultAliases]) before the
	// zero [lipgloss.Style].
	Tokens map[Token]lipgloss.Style

	// Aliases overrides [DefaultAliases] for this palette. A non-empty
	// mapping replaces the default; Aliases[X] = "" disables the
	// default for that key without substituting another.
	Aliases map[Token]Token

	// Chroma is an owned [*chroma.Style]. Non-nil wins over
	// [Palette.ChromaName]; when both are zero, [ChromaFromTokens]
	// supplies the default.
	Chroma *chroma.Style

	// ChromaName is the upstream chroma style name used when
	// [Palette.Chroma] is nil. Unknown names fall through to chroma's
	// own default at render time.
	ChromaName string

	// Glamour is an owned [*ansi.StyleConfig]. Non-nil wins over
	// [Palette.GlamourName] and [Palette.GlamourFor].
	Glamour *ansi.StyleConfig

	// GlamourName is the upstream glamour preset name used when
	// Glamour and GlamourFor are nil. Unknown names fall through to
	// glamour's own default.
	GlamourName string

	// GlamourFor is a capability-aware factory called by [Theme.Resolve]
	// when Glamour and GlamourName are zero. Errors surface from
	// [Theme.ResolveErr] (and are discarded by [Theme.Resolve]).
	GlamourFor func(Capabilities) (*ansi.StyleConfig, error)

	// Prompt is the Nabat-native prompt style block, converted to a
	// [huh.Theme] at resolve time. Zero falls back to
	// [PromptFromTokens]. [Palette.Huh] wins over Prompt when set.
	Prompt Prompt

	// Huh is the [huh.Theme] for interactive prompts. Non-nil wins
	// over [Prompt] and [PromptFromTokens]. Use it when the closed
	// [Prompt] surface is not enough.
	Huh huh.Theme
}

// Resolver resolves a theme against runtime [Capabilities]. Most themes
// declare one [Palette] per [Variant] and use [Theme.Resolve]. Implement
// Resolver directly only when palette choice cannot be expressed that way.
//
// [Theme] implements Resolver.
type Resolver interface {
	Resolve(Capabilities) ResolvedTheme
}

// Clone returns a shallow copy of t with an independent Variants map.
// Palette values inside the map are shared; assigning to
// Clone().Variants[k] does not affect the original.
func (t Theme) Clone() Theme {
	out := t
	out.Variants = maps.Clone(t.Variants)
	return out
}

// Resolve picks a [Variant] for [Capabilities], fills zero [Palette]
// fields with framework defaults, and returns an immutable
// [ResolvedTheme] safe to share across goroutines.
//
// One declared variant wins ([Theme.Default] ignored); otherwise Default
// picks. A zero default with multiple variants yields the zero
// ResolvedTheme. Resolve discards callback and alias-cycle errors; use
// [Theme.ResolveErr] for those. On [Palette.GlamourFor] failure the
// glamour slot stays empty.
func (t Theme) Resolve(c Capabilities) ResolvedTheme {
	rt, _ := t.resolveWithErr(c) //nolint:errcheck // Resolve intentionally drops the error; ResolveErr is the channel for it.
	return rt
}

// ResolveErr behaves like [Theme.Resolve] but also returns errors from
// per-Palette callbacks (today [Palette.GlamourFor]) and alias-cycle
// validation.
//
// On failure ResolveErr still returns a usable [ResolvedTheme] (failed
// glamour or alias slots stay empty) plus a non-nil error. Callers must
// check the error for diagnostics and may still apply the returned theme.
func (t Theme) ResolveErr(c Capabilities) (ResolvedTheme, error) {
	return t.resolveWithErr(c)
}

func (t Theme) resolveWithErr(c Capabilities) (ResolvedTheme, error) {
	variant := t.pickVariant(c)
	palette := t.Variants[variant]

	tokens := make(map[Token]lipgloss.Style, len(palette.Tokens))
	maps.Copy(tokens, palette.Tokens)

	// Chroma cascade: owned style > registered name > token-derived.
	// Fold it into a single value here so the resolved theme doesn't
	// expose three accessors per integration. The result is what
	// consumers like Context.highlight pass directly to the chroma
	// formatter; no global registry lookup at access time.
	chromaStyle := palette.Chroma
	if chromaStyle == nil {
		if palette.ChromaName != "" {
			chromaStyle = chromastyles.Get(palette.ChromaName)
		}
	}
	if chromaStyle == nil {
		chromaStyle = ChromaFromTokens(t.Name, tokens)
	}

	// Glamour cascade: owned style > capability-aware factory >
	// registered name > token-derived default. Same fold as
	// chroma above; the resolved value is the single source of truth
	// for renderMarkdown.
	//
	// Behavior on GlamourFor error: short-circuit. The author
	// explicitly opted into inline glamour by setting GlamourFor;
	// silently falling through to a name preset would mask the
	// failure. Return the error and leave the slot empty; consumers
	// fall through to glamour's own defaults at render time, but
	// the diagnostic surfaces.
	glamourStyle := palette.Glamour
	var resolveErr error
	skipGlamourFallback := false
	if glamourStyle == nil && palette.GlamourFor != nil {
		cfg, err := palette.GlamourFor(c)
		if err != nil {
			resolveErr = fmt.Errorf("theme %q: %w", t.Name, err)
			skipGlamourFallback = true
		} else {
			glamourStyle = cfg
		}
	}
	if glamourStyle == nil && !skipGlamourFallback && palette.GlamourName != "" {
		glamourStyle = glamourstyles.DefaultStyles[palette.GlamourName]
	}
	if glamourStyle == nil && !skipGlamourFallback {
		base := glamourstyles.DefaultStyles[GlamourPreset(variant, c)]
		glamourStyle = GlamourFromTokens(tokens, base)
	}

	// Aliases: merge the framework defaults with any per-palette
	// overrides, then validate the merged chain. A cycle in the
	// merged map (almost certainly an authoring bug) becomes the
	// resolved error so callers see it at App.finalize rather than
	// on the first Style() lookup.
	aliases := mergeAliases(palette.Aliases)
	for tok := range aliases {
		if cycleErr := validateAliasChain(tok, aliases); cycleErr != nil {
			if resolveErr == nil {
				resolveErr = fmt.Errorf("theme %q: %w", t.Name, cycleErr)
			}
			break
		}
	}

	// Huh cascade: Palette.Huh (escape hatch) wins outright; else
	// Palette.Prompt (Nabat-native, common case) is converted; else
	// the token-derived default kicks in. Theme-level PromptKnobs
	// are applied to whichever Prompt path wins before conversion.
	huhTheme := palette.Huh
	if huhTheme == nil {
		knobs := t.PromptKnobs
		if !palette.Prompt.IsZero() {
			huhTheme = knobs.Apply(palette.Prompt).Huh()
		} else {
			huhTheme = knobs.Apply(PromptFromTokens(tokens)).Huh()
		}
	}

	border := lipgloss.NormalBorder()
	if t.TableBorder != (lipgloss.Border{}) {
		border = t.TableBorder
	}
	enum := t.ListEnum
	if enum == nil {
		enum = list.Bullet
	}

	return ResolvedTheme{
		name:         t.Name,
		variant:      variant,
		tokens:       tokens,
		aliases:      aliases,
		chromaStyle:  chromaStyle,
		glamourStyle: glamourStyle,
		huhTheme:     huhTheme,
		listEnum:     enum,
		tableBorder:  border,
	}, resolveErr
}

// pickVariant selects the variant [Theme.Resolve] applies, in order:
// the lone key when only one variant exists; [VariantNoTTY] when
// non-interactive; [VariantDark] or [VariantLight] from
// [Capabilities.Dark]; else [Theme.Default].
func (t Theme) pickVariant(c Capabilities) Variant {
	if len(t.Variants) == 1 {
		for v := range t.Variants {
			return v
		}
	}
	if !c.Interactive {
		if _, ok := t.Variants[VariantNoTTY]; ok {
			return VariantNoTTY
		}
	}
	if c.Dark {
		if _, ok := t.Variants[VariantDark]; ok {
			return VariantDark
		}
	} else {
		if _, ok := t.Variants[VariantLight]; ok {
			return VariantLight
		}
	}
	return t.Default
}

// Validate reports structural problems with the theme.
// It fails when [Theme.Default] names a missing variant, or Default is set
// with an empty [Theme.Variants]. Empty themes without Default are valid.
func (t Theme) Validate() error {
	if len(t.Variants) == 0 {
		if t.Default != VariantUnset {
			return fmt.Errorf("theme %q: Default %q set but no Variants declared", t.Name, t.Default)
		}
		return nil
	}
	if t.Default == VariantUnset {
		return nil // single-variant themes don't need Default; pickVariant handles it
	}
	if _, ok := t.Variants[t.Default]; !ok {
		return fmt.Errorf("theme %q: Default %q does not match any declared variant", t.Name, t.Default)
	}
	return nil
}

// cloneTheme returns a deep-enough copy of t for catalog boundary use.
// Map fields are copied so callers can mutate returned themes without
// affecting cached registry state.
func cloneTheme(t Theme) Theme {
	out := Theme{
		Name:        t.Name,
		Default:     t.Default,
		ListEnum:    t.ListEnum,
		TableBorder: t.TableBorder,
		PromptKnobs: t.PromptKnobs,
	}
	if t.Variants != nil {
		out.Variants = make(map[Variant]Palette, len(t.Variants))
		for v, p := range t.Variants {
			out.Variants[v] = clonePalette(p)
		}
	}
	return out
}
