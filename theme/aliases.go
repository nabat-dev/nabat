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

// DefaultAliases is the framework fall-through map used when a [Token]
// has no direct entry in the resolved [Palette]. [ResolvedTheme.Style]
// walks the chain until it hits a style or bottoms out at the zero
// [lipgloss.Style].
//
// [Palette.Aliases] overrides entries here; an empty target disables the
// default for that key. Cycles stop at Style time (zero style) and are
// reported at [Theme.Resolve].
var DefaultAliases = map[Token]Token{
	ListEnumerator: TextMuted,
	TreeEnumerator: ListEnumerator,
	TableBorder:    TextMuted,

	ListItem:    TextPrimary,
	TreeItem:    TextPrimary,
	TableCell:   TextPrimary,
	TableHeader: TextTitle,

	SpinnerActive: StatusInfo,
	StatusActive:  StatusInfo,
}

// validateAliasChain reports a cycle in the alias chain starting at
// start. Called from [Theme.Resolve] so authoring mistakes surface at
// load rather than on first [ResolvedTheme.Style] call.
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

// mergeAliases returns [DefaultAliases] overlaid by palette overrides.
// Per-palette entries win; an empty Token disables the default for that
// key. The returned map is a fresh allocation.
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

// cloneAliases returns a fresh copy of an alias map.
func cloneAliases(src map[Token]Token) map[Token]Token {
	out := make(map[Token]Token, len(src))
	maps.Copy(out, src)
	return out
}
