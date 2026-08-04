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
	"github.com/alecthomas/chroma/v2"
)

// Override is a per-[Palette] mutation from the Set* helpers. Applied
// to every variant of the underlying [Theme], so a single token tweak
// affects whichever variant [Theme.Resolve] picks.
//
// Prefer the Set* helpers (or nabat.WithThemeOverride) over implementing
// Override directly.
type Override interface {
	apply(*Palette)
}

// overrideFn adapts a func(*Palette) into [Override].
type overrideFn func(*Palette)

func (f overrideFn) apply(p *Palette) { f(p) }

// SetToken returns an [Override] that records s under token t,
// shadowing any existing palette value. Applies to every variant.
func SetToken(t Token, s lipgloss.Style) Override {
	return overrideFn(func(p *Palette) {
		if p.Tokens == nil {
			p.Tokens = map[Token]lipgloss.Style{}
		}
		p.Tokens[t] = s
	})
}

// SetAlias returns an [Override] that sets tok's fall-through target in
// [Palette.Aliases]. An empty target disables the framework default for
// that key.
func SetAlias(tok, target Token) Override {
	return overrideFn(func(p *Palette) {
		if p.Aliases == nil {
			p.Aliases = map[Token]Token{}
		}
		p.Aliases[tok] = target
	})
}

// SetChroma returns an [Override] that sets an owned [*chroma.Style]
// and clears [Palette.ChromaName] so the cascade uses the pointer alone.
func SetChroma(s *chroma.Style) Override {
	return overrideFn(func(p *Palette) {
		p.Chroma = s
		p.ChromaName = ""
	})
}

// SetChromaName returns an [Override] that sets the upstream chroma
// style name and clears [Palette.Chroma].
func SetChromaName(name string) Override {
	return overrideFn(func(p *Palette) {
		p.Chroma = nil
		p.ChromaName = name
	})
}

// SetGlamour returns an [Override] that sets an owned
// [*ansi.StyleConfig] and clears [Palette.GlamourName] and
// [Palette.GlamourFor].
func SetGlamour(s *ansi.StyleConfig) Override {
	return overrideFn(func(p *Palette) {
		p.Glamour = s
		p.GlamourName = ""
		p.GlamourFor = nil
	})
}

// SetGlamourName returns an [Override] that sets the upstream glamour
// preset name and clears [Palette.Glamour] and [Palette.GlamourFor].
func SetGlamourName(name string) Override {
	return overrideFn(func(p *Palette) {
		p.Glamour = nil
		p.GlamourName = name
		p.GlamourFor = nil
	})
}

// SetHuh returns an [Override] that sets the [huh.Theme] for interactive
// prompts. Nil reverts to the [PromptFromTokens] fallback at
// [Theme.Resolve] time.
func SetHuh(h huh.Theme) Override {
	return overrideFn(func(p *Palette) {
		p.Huh = h
	})
}

// With returns a derived [Theme] with overrides applied to every
// variant. The receiver is not modified; the returned theme owns fresh
// per-variant maps. Overrides apply left to right; the right-most
// override of a token wins. With panics if any override is nil.
//
// Example:
//
//	dracula, _ := theme.Get(theme.Dracula)
//	mine := dracula.With(
//	    theme.SetToken(theme.StatusError, magenta),
//	    theme.SetAlias(theme.ListItem, theme.TextSecondary),
//	)
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
		for i, o := range overrides {
			if o == nil {
				panic(fmt.Sprintf("nabat/theme: Theme.With: Override at index %d is nil", i))
			}
			o.apply(&copied)
		}
		out.Variants[v] = copied
	}
	return out
}

// clonePalette returns a copy of src with independent Tokens and
// Aliases maps. Pointer fields are shared; override helpers replace
// them wholesale rather than mutating in place.
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
