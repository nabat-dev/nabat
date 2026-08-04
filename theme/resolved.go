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

// ResolvedTheme is the immutable result of [Theme.Resolve], queried by
// [Token] or accessor. Safe for concurrent use after construction.
// Slice results are defensive copies; shared style pointers are read-only.
// The zero value yields empty styles and nil chroma/glamour.
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

// Name returns the theme name from [Theme.Name]. Empty means a nameless
// theme, typically the zero ResolvedTheme fallback.
func (r ResolvedTheme) Name() string { return r.name }

// Variant returns the variant [Theme.Resolve] picked. [VariantUnset]
// means no declaration; treat it as compatible with every
// [Capabilities] snapshot. Use it for diagnostics, not to skip applying
// the theme (Resolve already picked a compatible palette).
func (r ResolvedTheme) Variant() Variant { return r.variant }

// Style returns the [lipgloss.Style] for token t, walking the alias
// chain when t has no direct entry. The chain is the merge of
// [DefaultAliases] and [Palette.Aliases] folded at [Theme.Resolve] time.
//
// When the chain bottoms out, Style returns the zero [lipgloss.Style]
// (terminal default). A cyclic chain returns the zero style via a
// per-call seen set; [Theme.Resolve] also reports cycles at construction.
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

// Tokens returns the tokens this theme set explicitly. The slice is
// freshly allocated; callers may modify it without affecting the theme.
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
// nil means no highlighting; the upstream chroma fallback applies at
// render time.
func (r ResolvedTheme) Chroma() *chroma.Style { return r.chromaStyle }

// Glamour returns the resolved [*ansi.StyleConfig] for markdown
// rendering. nil means use glamour's own default at render time.
func (r ResolvedTheme) Glamour() *ansi.StyleConfig { return r.glamourStyle }

// Huh returns the [huh.Theme] from [Palette.Huh], or the
// [PromptFromTokens] fallback when that field was nil.
func (r ResolvedTheme) Huh() huh.Theme { return r.huhTheme }

// ListEnumerator returns the default list enumerator. The zero
// ResolvedTheme returns nil ([list.Bullet] at the call site);
// [Theme.Resolve] always produces a non-nil value.
func (r ResolvedTheme) ListEnumerator() list.Enumerator { return r.listEnum }

// TableBorder returns the default table border. The zero ResolvedTheme
// returns the zero [lipgloss.Border]; [Theme.Resolve] always produces
// a usable border ([lipgloss.NormalBorder] when none was set).
func (r ResolvedTheme) TableBorder() lipgloss.Border { return r.tableBorder }
