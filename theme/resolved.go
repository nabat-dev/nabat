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
	"charm.land/glamour/v2/ansi"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/alecthomas/chroma/v2"
)

// ResolvedTheme is the final, immutable result of resolving a [Theme]
// against a [Capabilities] snapshot via [Theme.Resolve]. Consumers (the
// nabat root package, the logging adapter, third-party extensions)
// hold a ResolvedTheme and query it by [Token] or by accessor.
//
// ResolvedTheme is safe for concurrent use after construction; nothing
// on it mutates after [Theme.Resolve] returns. Multiple goroutines
// may call any accessor on the same ResolvedTheme without
// synchronization. Slice and map results are returned as defensive
// copies (e.g. [ResolvedTheme.Tokens]) or are read-only data shared by
// design (e.g. the embedded [chroma.Style] pointer).
//
// The zero value of ResolvedTheme is the "empty" theme: every Style
// call returns the zero [lipgloss.Style] (which lipgloss renders as
// the terminal default), [Chroma] and [Glamour] return nil
// (consumers fall through to the upstream library defaults), and huh
// uses its own defaults. App.New produces a non-zero ResolvedTheme
// even when the user supplies no theme — the empty result here is a
// fallback for tests and for code that constructs a ResolvedTheme
// directly.
//
// Phase 3 of the theme redesign collapsed the three-accessor
// owned/named/cascade pattern into single accessors per integration:
// [Chroma], [Glamour], [Huh]. The cascade is folded once at
// [Theme.Resolve] time, so callers see one already-resolved value
// without having to re-implement the precedence rule.
type ResolvedTheme struct {
	name    string
	variant Variant

	tokens  map[Token]lipgloss.Style
	aliases map[Token]Token

	chromaStyle  *chroma.Style
	glamourStyle *ansi.StyleConfig
	huhTheme     huh.Theme

	listEnum    list.Enumerator
	tableBorder lipgloss.Border
}

// Name returns the theme name from [Theme.Name] (or the registry
// parser for manifest-defined themes). The empty string indicates a
// nameless theme — typically the zero ResolvedTheme used as a fallback.
func (r ResolvedTheme) Name() string { return r.name }

// Variant returns the variant [Theme.Resolve] picked. The zero value
// [VariantUnset] means the source theme made no declaration; consumers
// should treat it as compatible with every [Capabilities] snapshot.
//
// Use Variant for advisory diagnostics (e.g. logging when a dark
// theme is paired with what looks like a light terminal); do NOT
// short-circuit theme application based on it — [Theme.Resolve] has
// already picked a palette compatible with the supplied capabilities.
func (r ResolvedTheme) Variant() Variant { return r.variant }

// Style returns the [lipgloss.Style] registered for token t, walking
// the alias chain when t has no direct entry. The chain is the merge
// of [DefaultAliases] and the resolved palette's [Palette.Aliases];
// [Theme.Resolve] folds them at construction time so [Style] never
// touches the package-level default map.
//
// Lookup order: tokens[t] -> tokens[aliases[t]] -> tokens[aliases[aliases[t]]]
// -> ... When the chain bottoms out (no direct entry, no further
// alias) Style returns the zero [lipgloss.Style] — lipgloss renders
// it as the terminal default, which is the right fallback for "this
// theme didn't customize this slot".
//
// Cycle safety: Style maintains a per-call seen set so a malformed
// alias chain (one that loops on itself) returns the zero style
// instead of looping forever. [Theme.Resolve] also validates aliases
// at construction time, so a cycle is detected and reported there
// before [Style] ever sees it.
func (r ResolvedTheme) Style(t Token) lipgloss.Style {
	if r.tokens == nil {
		return lipgloss.Style{}
	}
	if s, ok := r.tokens[t]; ok {
		return s
	}
	if r.aliases == nil {
		return lipgloss.Style{}
	}
	seen := map[Token]bool{t: true}
	for cur := r.aliases[t]; cur != "" && !seen[cur]; cur = r.aliases[cur] {
		seen[cur] = true
		if s, ok := r.tokens[cur]; ok {
			return s
		}
	}
	return lipgloss.Style{}
}

// Tokens returns the set of tokens this theme has explicitly set. The
// returned slice is freshly allocated; callers may sort or modify it
// without affecting the theme. Tokens is intended for diagnostics
// (printing what a theme covers) and for extensions that want to skip
// tokens the theme did not opt into.
func (r ResolvedTheme) Tokens() []Token {
	if len(r.tokens) == 0 {
		return nil
	}
	out := make([]Token, 0, len(r.tokens))
	for k := range r.tokens {
		out = append(out, k)
	}
	return out
}

// Chroma returns the resolved [*chroma.Style] for syntax highlighting.
// [Theme.Resolve] folds the owned-style / named-preset / variant-
// default cascade into this single value at construction time so
// consumers (markdown highlighter, structured logger, third-party
// extensions) never have to rebuild the precedence rule. nil means
// "no syntax highlighting" — the upstream chroma fallback applies at
// render time.
func (r ResolvedTheme) Chroma() *chroma.Style { return r.chromaStyle }

// Glamour returns the resolved [*ansi.StyleConfig] for markdown
// rendering. [Theme.Resolve] folds the owned-style / capability-aware-
// factory / named-preset / capability-default cascade into this single
// value at construction time. nil means "use glamour's own default
// styling at render time".
func (r ResolvedTheme) Glamour() *ansi.StyleConfig { return r.glamourStyle }

// Huh returns the [huh.Theme] from [Palette.Huh] (or the
// [HuhFromTokens] fallback derived from the palette's tokens when
// that field was nil). Consumers pass the result to form.WithTheme
// or spinner.New().Theme.
func (r ResolvedTheme) Huh() huh.Theme { return r.huhTheme }

// ListEnumerator returns the default list enumerator. The zero
// ResolvedTheme returns nil, which Context.List interprets as the
// lipgloss default ([list.Bullet]); [Theme.Resolve] always produces
// a non-nil value.
func (r ResolvedTheme) ListEnumerator() list.Enumerator { return r.listEnum }

// TableBorder returns the default table border. The zero ResolvedTheme
// returns the zero [lipgloss.Border] (no characters drawn);
// [Theme.Resolve] always produces a usable border
// ([lipgloss.NormalBorder] when none was set).
func (r ResolvedTheme) TableBorder() lipgloss.Border { return r.tableBorder }
