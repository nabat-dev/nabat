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
)

// DefaultAliases is the framework-owned fall-through map consulted by
// [ResolvedTheme.Style] when a [Token] has no direct entry in the
// resolved [Palette]. The chain is: lookup t -> if missing, try
// DefaultAliases[t] -> if missing, try DefaultAliases[that] -> ...
// When the chain bottoms out without a hit, [ResolvedTheme.Style]
// returns the zero [lipgloss.Style] (terminal default), preserving
// the existing "unset token = terminal default" contract.
//
// The defaults are chosen so the most common authoring pattern —
// "I picked four status colors, two text colors, and one muted
// color, and that's all I want to think about" — produces a
// fully-themed CLI without forcing every theme to repeat the same
// "tree.enumerator -> muted" / "table.border -> muted" mappings.
//
// Per-theme [Palette.Aliases] overrides any entry here; cycles are
// detected at [ResolvedTheme.Style] time and treated as "stop here,
// return the zero style".
var DefaultAliases = map[Token]Token{
	// Chrome markers all collapse to TextMuted by default. Authors
	// who want to differentiate (say, brighter list bullets than
	// table borders) override the specific token.
	ListEnumerator: TextMuted,
	TreeEnumerator: ListEnumerator,
	TableBorder:    TextMuted,

	// Item / cell / row text shares TextPrimary (or TextSecondary) by
	// default. Tables are slightly different: data cells read like
	// values, header cells read like titles.
	ListItem:    TextPrimary,
	TreeItem:    TextPrimary,
	TableCell:   TextPrimary,
	TableHeader: TextTitle,
}

// validateAliasChain walks the chain from t through aliases, ensuring
// no cycle exists and every linked token is non-empty. It is called
// from [Theme.Resolve] over the merged alias map so authoring mistakes
// (a manifest that links its alias chain into a loop) surface at
// theme load rather than via a quiet stack-walk-bound at first
// [ResolvedTheme.Style] call.
//
// Returns nil when the chain is acyclic; otherwise an error naming
// the offending token and the cycle path.
func validateAliasChain(start Token, aliases map[Token]Token) error {
	if aliases == nil {
		return nil
	}
	seen := map[Token]bool{start: true}
	order := []Token{start}
	for cur := aliases[start]; cur != ""; cur = aliases[cur] {
		if seen[cur] {
			order = append(order, cur)
			return fmt.Errorf("token alias cycle detected: %v", order)
		}
		seen[cur] = true
		order = append(order, cur)
	}
	return nil
}

// mergeAliases returns the alias map [ResolvedTheme.Style] consults
// for a given palette: the framework defaults overlaid by any
// per-palette overrides. Per-palette entries win; an explicit
// per-palette mapping to the empty Token disables the framework
// default for that key (so authors can opt out).
//
// The returned map is a fresh allocation — mutating it has no effect
// on either input.
func mergeAliases(palette map[Token]Token) map[Token]Token {
	if len(palette) == 0 {
		return cloneAliases(DefaultAliases)
	}
	out := cloneAliases(DefaultAliases)
	for k, v := range palette {
		if v == "" {
			delete(out, k)
			continue
		}
		out[k] = v
	}
	return out
}

// cloneAliases returns a fresh copy of an alias map. Used so the
// resolved theme owns its own map and is safe to share across
// goroutines without callers worrying about [DefaultAliases]
// mutation (which would be a bug anywhere).
func cloneAliases(src map[Token]Token) map[Token]Token {
	out := make(map[Token]Token, len(src))
	maps.Copy(out, src)
	return out
}
