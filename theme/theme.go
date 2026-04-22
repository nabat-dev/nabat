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

// Variant is a theme's intended luminance / TTY context — the
// background brightness or render mode the author designed for. It is
// the slot a future "--theme-variant" override flag and runtime
// diagnostics will read (e.g. "the active theme targets dark
// backgrounds; your terminal looks light") to choose between light
// and dark palette flavors of a brand theme.
//
// The zero Variant ([VariantUnset]) means "the theme did not declare
// a target variant"; consumers should treat it as compatible with
// every Capabilities snapshot.
type Variant string

// Variant constants enumerate the values a manifest's "variant" field
// (and a programmatic [Theme.Default] / map key) accepts. They match
// the strings the manifest schema validates against, so a manifest's
// "variant" field round-trips through this type without a translation
// step.
const (
	// VariantUnset is the zero value: the theme did not declare a
	// target variant. Consumers should not gate behavior on this.
	VariantUnset Variant = ""

	// VariantDark indicates the theme was designed for dark
	// terminals. Use it when the palette assumes a dark background.
	VariantDark Variant = "dark"

	// VariantLight indicates the theme was designed for light
	// terminals. Use it when the palette assumes a light background.
	VariantLight Variant = "light"

	// VariantNoTTY indicates the theme was designed for non-TTY
	// output (logs, CI artifacts). Use it when the palette assumes
	// no ANSI styling at all.
	VariantNoTTY Variant = "notty"
)

// Theme is declarative data describing a complete CLI styling. It
// carries one or more [Palette] entries (one per declared variant),
// the variant the theme defaults to when [Capabilities] do not pin a
// clear pick, and a small set of cross-variant defaults (list
// enumerator, table border).
//
// Themes resolve once per [App] at construction time via [Theme.Resolve],
// which picks a variant, applies framework defaults for any
// per-palette field the author left zero, and returns an immutable
// [ResolvedTheme]. The choice to model themes as data (rather than
// the closure type they used to be) is what lets the catalog return
// inspectable values, lets [Override] build derived themes without
// re-running setup, and keeps the manifest loader a pure data
// pipeline.
//
// Theme is safe to copy by value; however the copy shares the Variants map
// with the original — mutating `copy.Variants[k] = p` affects the source.
// To derive a modified theme with an independent Variants map, use
// [Theme.Clone]. To tweak a single token in a built-in theme without
// constructing a full copy, use [nabat.WithThemeOverride].
//
// Theme implements [Resolver]; pass it anywhere a Resolver is expected.
type Theme struct {
	// Name identifies the theme in error messages, the catalog
	// registry key, and [ResolvedTheme.Name]. Required for built-in
	// themes; programmatic themes may leave it empty.
	Name string

	// Variants maps each declared [Variant] to its [Palette]. Most
	// themes ship a single entry; multi-variant themes carry one
	// palette per dark/light/notty mode and let [Theme.Resolve] pick
	// at runtime.
	//
	// An empty Variants produces a zero [ResolvedTheme] from
	// [Theme.Resolve] — every Style call returns the zero
	// [lipgloss.Style], which lipgloss renders as the terminal
	// default.
	Variants map[Variant]Palette

	// Default is the variant [Theme.Resolve] picks when the runtime
	// [Capabilities] do not point at one of the declared variants.
	// For single-variant themes it must equal the only key in
	// Variants; the zero [VariantUnset] is treated as the lone key
	// when exactly one variant exists.
	Default Variant

	// ListEnum is the default enumerator for [Context.List] output.
	// Themes that want bullets, dashes, or numbers without a
	// per-call override set it once here; the framework picks
	// [list.Bullet] when this is nil.
	ListEnum list.Enumerator

	// TableBorder is the default border drawn by [Context.Table]
	// when no per-call override is supplied. The zero
	// [lipgloss.Border] resolves to [lipgloss.NormalBorder].
	TableBorder lipgloss.Border

	// PromptKnobs are theme-wide non-color prompt settings applied
	// to token-derived and palette prompt styles.
	PromptKnobs PromptKnobs
}

// Palette is the per-variant style data a [Theme] carries. Each
// declared variant maps to one Palette; [Theme.Resolve] picks one and
// fills any nil/empty cascade slot with framework defaults.
//
// The chroma and glamour cascades are intentionally three-way (owned
// value > registered name > capability default) because both upstream
// libraries support either form, and themes commonly mix them — for
// example "use the registered Dracula chroma style but ship a custom
// glamour config". The cascade is folded into a single value at
// [Theme.Resolve] time so [ResolvedTheme] consumers see only the
// resolved result, not the source path.
type Palette struct {
	// Tokens is the per-token style map. Lookups in
	// [ResolvedTheme.Style] hit this map first; tokens the
	// palette omits fall through to the alias chain (per-palette
	// [Aliases] overlaid on [DefaultAliases]) before defaulting to
	// the zero [lipgloss.Style] (terminal default).
	Tokens map[Token]lipgloss.Style

	// Aliases overrides entries in [DefaultAliases] for this
	// palette. A non-empty mapping replaces the framework default;
	// an explicit empty Token value (Aliases[X] = "") disables the
	// framework default for that key without substituting another.
	//
	// Typical use: the manifest's "aliases" field, where a theme
	// author wants list bullets to follow text.secondary instead of
	// text.muted. Most themes leave this nil and inherit the
	// framework defaults wholesale.
	Aliases map[Token]Token

	// Chroma is an owned [*chroma.Style] for syntax highlighting.
	// When non-nil it wins over [Palette.ChromaName]; when both are
	// zero the palette's variant determines the framework default
	// via [ChromaPreset].
	Chroma *chroma.Style

	// ChromaName is the upstream chroma style name (e.g. "monokai",
	// "dracula"). Used when [Palette.Chroma] is nil. Unknown names
	// fall through to chroma's own default at render time.
	ChromaName string

	// Glamour is an owned [*ansi.StyleConfig] for markdown
	// rendering. When non-nil it wins over both
	// [Palette.GlamourName] and [Palette.GlamourFor].
	Glamour *ansi.StyleConfig

	// GlamourName is the upstream glamour preset name (e.g. "dark",
	// "light", "notty"). Used when both Glamour and GlamourFor are
	// nil; unknown names fall through to glamour's own default.
	GlamourName string

	// GlamourFor is the capability-aware factory used by themes
	// (typically the manifest loader's inline "glamourStyle" path)
	// that need to evaluate glamour against the current
	// [Capabilities]. Called by [Theme.Resolve] when both Glamour
	// and GlamourName are zero. The function may return an error
	// that surfaces from [Theme.Resolve].
	GlamourFor func(Capabilities) (*ansi.StyleConfig, error)

	// Prompt is the Nabat-native style block for interactive
	// prompts. The framework converts it to a [huh.Theme] at
	// [Theme.Resolve] time. Zero (the empty Prompt) means the
	// catalog falls back to [PromptFromTokens] using
	// [Palette.Tokens]; setting any field opts into the closed
	// Nabat-native surface and forgoes the framework default.
	//
	// [Palette.Huh] still wins over [Prompt] when set — that's
	// the escape hatch for themes that need huh's full surface.
	Prompt Prompt

	// Huh is the [huh.Theme] used by interactive prompts. When
	// non-nil it wins outright over [Prompt] and the
	// [PromptFromTokens] fallback. The escape hatch is for themes
	// that need huh's full per-state surface (separate focused /
	// blurred styling, custom textInput layout, etc.) — the
	// closed Nabat-native [Prompt] cannot express those.
	Huh huh.Theme
}

// Resolver is the escape hatch for themes whose palette choice depends
// on runtime [Capabilities] in a way that cannot be expressed as
// "one Palette per Variant". Most themes (every built-in, every
// straight programmatic theme) declare one Palette per variant and
// let [Theme.Resolve] pick. The rare cases — for example a theme that
// switches palettes when the color profile is ANSI16 vs TrueColor —
// implement Resolver directly and bypass [Theme] entirely.
//
// [Theme] implements Resolver via its [Theme.Resolve] method, so a
// Theme value can be passed anywhere a Resolver is expected.
type Resolver interface {
	Resolve(Capabilities) ResolvedTheme
}

// Clone returns a shallow copy of t with an independent Variants map.
// Palette values inside the map are shared (not deep-copied); the new
// map itself is separate, so assigning to Clone().Variants[k] does not
// affect the original theme.
func (t Theme) Clone() Theme {
	out := t
	out.Variants = maps.Clone(t.Variants)
	return out
}

// Resolve picks a [Variant] for the supplied [Capabilities], applies
// framework defaults to any zero field on the chosen [Palette], and
// returns an immutable [ResolvedTheme]. The result is safe to share
// across goroutines.
//
// Variant selection:
//
//   - If the theme declares exactly one variant, that one is used and
//     [Theme.Default] is ignored.
//   - Otherwise, [Theme.Default] picks; the zero default plus
//     multiple variants returns the zero ResolvedTheme rather than a
//     guess (multi-variant resolution sharpens in a later phase).
//
// Cascade defaults applied to the chosen Palette:
//
//   - [Palette.Huh] nil  -> derived from [Palette.Tokens] via [PromptFromTokens].
//   - [Palette.Glamour] nil + [Palette.GlamourFor] non-nil -> evaluate
//     against Capabilities; an error is logged via the returned
//     ResolvedTheme's name in the error chain (see error return).
//   - [Palette.Glamour] nil + [Palette.GlamourFor] nil + [Palette.GlamourName]
//     empty -> [GlamourFromTokens] with [GlamourPreset] base.
//   - [Palette.Chroma] nil + [Palette.ChromaName] empty ->
//     [ChromaFromTokens].
//
// Cross-palette defaults applied:
//
//   - [Theme.ListEnum] nil -> [list.Bullet].
//   - [Theme.TableBorder] zero -> [lipgloss.NormalBorder].
//
// Errors:
//
//   - The inline-glamour [Palette.GlamourFor] callback may fail; the
//     error is wrapped with the theme name and returned. Resolve
//     still produces a usable ResolvedTheme in that case (the
//     glamour slot stays empty, falling through to glamour's own
//     defaults at render time) so consumers can surface the error
//     diagnostically without losing the rest of the styling.
func (t Theme) Resolve(c Capabilities) ResolvedTheme {
	rt, _ := t.resolveWithErr(c) //nolint:errcheck // Resolve intentionally drops the error; ResolveErr is the channel for it.
	return rt
}

// ResolveErr behaves like [Theme.Resolve] but also returns the error
// from any per-Palette callback (today only [Palette.GlamourFor]).
// Construction paths that want to surface those failures (App.finalize,
// the catalog loader's own validations) should use this; consumers
// that just need a styling pick the error-eating Resolve.
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
	// formatter — no global registry lookup at access time.
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
	// failure. Return the error and leave the slot empty — consumers
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

// pickVariant picks the variant [Theme.Resolve] should apply. The
// rule is, in order:
//
//  1. Single-variant themes: return the lone key (regardless of
//     [Theme.Default]).
//  2. Non-interactive output: prefer [VariantNoTTY] when declared.
//  3. Dark terminal: prefer [VariantDark] when declared.
//  4. Light terminal: prefer [VariantLight] when declared.
//  5. Fall back to [Theme.Default] when no capability-matched
//     variant exists.
//
// The capability-driven pick is what lets a single multi-variant
// manifest (say "dracula" with both dark and light palettes) flip
// automatically based on the terminal's detected background — no
// `nabat.WithTheme("dracula-light")` needed when the user sits in a
// light terminal.
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

// Validate returns any structural problems with the theme. The catalog
// uses it to surface broken built-in themes at registry load; user
// code rarely needs it directly.
//
// Errors:
//
//   - Default references a variant not declared in [Theme.Variants].
//   - [Theme.Variants] is empty AND [Theme.Default] is not unset
//     (an empty theme is valid; mismatched defaults are not).
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
