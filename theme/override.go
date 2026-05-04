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
	"maps"

	"charm.land/glamour/v2/ansi"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
)

// Override is a per-[Palette] mutation produced by the Set* helpers
// in this file. The framework applies overrides to every variant of
// the underlying [Theme] so a one-line "make status.error magenta"
// affects whichever variant [Theme.Resolve] picks at runtime.
//
// Override is an interface (not a function type) so the Set* helpers
// can return concrete typed values that test assertions match against
// without reflection. Third-party code rarely implements Override
// directly; reach for the Set* helpers (and, for [App] users, the
// nabat.WithThemeOverride option) instead.
type Override interface {
	apply(*Palette)
}

// overrideFn adapts a func(*Palette) into the [Override] interface so
// the Set* helpers can stay one-liners. Keeping the type unexported
// preserves the "implement Override only via the helpers" contract
// stated above.
type overrideFn func(*Palette)

func (f overrideFn) apply(p *Palette) { f(p) }

// SetToken returns an [Override] that records s under token t,
// shadowing any value the underlying [Palette] already carries.
// Overrides apply to every variant of the underlying [Theme]; pair
// with the multi-variant resolution that [Theme.Resolve] performs
// to keep "tweak this one slot" trivial regardless of how many
// variants the theme declares.
//
// Use case: a downstream app that picks the bundled Dracula theme
// but wants its own brand color for the error status. One line at
// nabat.New is enough; the Theme value stays declarative.
func SetToken(t Token, s lipgloss.Style) Override {
	return overrideFn(func(p *Palette) {
		if p.Tokens == nil {
			p.Tokens = map[Token]lipgloss.Style{}
		}
		p.Tokens[t] = s
	})
}

// SetAlias returns an [Override] that records src as the fall-through
// target for tok in the palette's [Palette.Aliases] map. Pass an
// empty target to disable an alias (matches the Palette.Aliases
// "empty value disables the framework default" semantics).
func SetAlias(tok, target Token) Override {
	return overrideFn(func(p *Palette) {
		if p.Aliases == nil {
			p.Aliases = map[Token]Token{}
		}
		p.Aliases[tok] = target
	})
}

// SetChroma returns an [Override] that swaps in a different owned
// [*chroma.Style] for syntax highlighting. The override clears
// [Palette.ChromaName] so the resulting cascade resolves through
// the explicit pointer alone — no risk of the registered name
// silently winning back when the override is dropped later.
func SetChroma(s *chroma.Style) Override {
	return overrideFn(func(p *Palette) {
		p.Chroma = s
		p.ChromaName = ""
	})
}

// SetChromaName returns an [Override] that switches the upstream
// chroma style name. Like [SetChroma] it clears the sibling field
// (in this case [Palette.Chroma]) so the cascade picks the named
// preset deterministically.
func SetChromaName(name string) Override {
	return overrideFn(func(p *Palette) {
		p.Chroma = nil
		p.ChromaName = name
	})
}

// SetGlamour returns an [Override] that swaps in a different owned
// [*ansi.StyleConfig] for markdown rendering. The override clears
// [Palette.GlamourName] and [Palette.GlamourFor] so the resulting
// cascade resolves through the explicit pointer alone.
func SetGlamour(s *ansi.StyleConfig) Override {
	return overrideFn(func(p *Palette) {
		p.Glamour = s
		p.GlamourName = ""
		p.GlamourFor = nil
	})
}

// SetGlamourName returns an [Override] that switches the upstream
// glamour preset name. Clears the sibling owned-style and factory
// fields so the cascade picks the named preset deterministically.
func SetGlamourName(name string) Override {
	return overrideFn(func(p *Palette) {
		p.Glamour = nil
		p.GlamourName = name
		p.GlamourFor = nil
	})
}

// SetHuh returns an [Override] that swaps in a different [huh.Theme]
// for interactive prompts. Setting nil reverts to the
// [HuhFromTokens] fallback at [Theme.Resolve] time.
func SetHuh(h huh.Theme) Override {
	return overrideFn(func(p *Palette) {
		p.Huh = h
	})
}

// With returns a derived [Theme] with the supplied overrides applied
// to every variant. The receiver is not modified; the returned theme
// owns fresh per-variant maps so subsequent overrides on the
// receiver (or the original Theme) do not leak across.
//
// Override application order matches the supplied slice (left to
// right), so the right-most override of a token wins. Overrides
// targeting different fields compose freely.
//
// Typical use:
//
//	dracula, _ := theme.Get(theme.Dracula)
//	mine := dracula.With(
//	    theme.SetToken(theme.StatusError, magenta),
//	    theme.SetAlias(theme.ListItem, theme.TextSecondary),
//	)
//	app, _ := nabat.New("myctl", nabat.WithCustomTheme(mine))
func (t Theme) With(overrides ...Override) Theme {
	if len(overrides) == 0 {
		return t
	}
	out := Theme{
		Name:        t.Name,
		Default:     t.Default,
		ListEnum:    t.ListEnum,
		TableBorder: t.TableBorder,
		PromptKnobs: t.PromptKnobs,
		Variants:    make(map[Variant]Palette, len(t.Variants)),
	}
	for v, src := range t.Variants {
		copied := clonePalette(src)
		for _, o := range overrides {
			if o == nil {
				continue
			}
			o.apply(&copied)
		}
		out.Variants[v] = copied
	}
	return out
}

// clonePalette returns a deep-enough copy of src that applying
// [Override] mutations to the result does not leak into the source.
// Map fields (Tokens, Aliases) are copied; pointer fields are not —
// they are immutable by convention (a chroma.Style or huh.Theme
// passed into a Palette is meant to be shared, not edited) and the
// override helpers replace the pointer wholesale rather than
// mutating it in place.
func clonePalette(src Palette) Palette {
	out := Palette{
		Chroma:      src.Chroma,
		ChromaName:  src.ChromaName,
		Glamour:     src.Glamour,
		GlamourName: src.GlamourName,
		GlamourFor:  src.GlamourFor,
		Prompt:      src.Prompt,
		Huh:         src.Huh,
	}
	if src.Tokens != nil {
		out.Tokens = make(map[Token]lipgloss.Style, len(src.Tokens))
		maps.Copy(out.Tokens, src.Tokens)
	}
	if src.Aliases != nil {
		out.Aliases = make(map[Token]Token, len(src.Aliases))
		maps.Copy(out.Aliases, src.Aliases)
	}
	return out
}
