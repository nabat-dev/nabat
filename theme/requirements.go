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
	"sort"
	"strings"
)

// Requirement declares the [Token] set a consumer reads from a
// [ResolvedTheme]. Consumer identifies the requester in diagnostics
// (for example "logging extension"). Plain data; pass by value; treat
// Tokens as read-only.
type Requirement struct {
	Consumer string
	Tokens   []Token
}

// Require returns a [Requirement] for consumer and the tokens it reads.
//
// Example:
//
//	return theme.Require("logging extension",
//	    theme.StatusInfo, theme.StatusWarning, theme.StatusError,
//	)
func Require(consumer string, tokens ...Token) Requirement {
	return Requirement{Consumer: consumer, Tokens: tokens}
}

// CoreRequirements returns the [Requirement] entries the nabat root
// package reads from a [ResolvedTheme]. The framework includes this
// list automatically; extensions declare only what they add.
func CoreRequirements() []Requirement {
	return []Requirement{
		{
			Consumer: "core semantic output",
			Tokens: []Token{
				StatusSuccess, StatusWarning, StatusError, StatusInfo,
				AccentPrimary, TextPrimary,
			},
		},
		{
			Consumer: "core help renderer",
			Tokens:   []Token{TextTitle, TextSecondary, AccentPrimary},
		},
		{
			Consumer: "core derived integrations",
			Tokens:   []Token{TextMuted, TextLink, CodeSurface},
		},
		{
			Consumer: "core structured output",
			Tokens: []Token{
				TableBorder, TableHeader, TableCell,
				ListItem, ListEnumerator,
				TreeItem, TreeEnumerator,
			},
		},
	}
}

// HasToken reports whether token t is covered on this [ResolvedTheme],
// either directly or via the alias chain.
//
// A token explicitly set to the zero [lipgloss.Style] counts as covered
// (the author opted in). Consumers that need the style still call
// [ResolvedTheme.Style].
func (r ResolvedTheme) HasToken(t Token) bool {
	if r.tokens == nil {
		return false
	}
	if _, ok := r.tokens[t]; ok {
		return true
	}
	if r.aliases == nil {
		return false
	}
	seen := map[Token]bool{t: true}
	for cur := r.aliases[t]; cur != "" && !seen[cur]; cur = r.aliases[cur] {
		seen[cur] = true
		if _, ok := r.tokens[cur]; ok {
			return true
		}
	}
	return false
}

// MissingTokens returns tokens from req that this [ResolvedTheme] does
// not cover. The result is sorted lexically; empty means fully satisfied.
func (r ResolvedTheme) MissingTokens(req Requirement) []Token {
	if len(req.Tokens) == 0 {
		return nil
	}
	var out []Token
	for _, t := range req.Tokens {
		if !r.HasToken(t) {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// CheckRequirements returns an error listing consumers whose tokens
// this [ResolvedTheme] does not fully cover. nil means all satisfied.
//
// Diagnostic format:
//
//	theme "minimal" is missing tokens required by:
//	  - logging extension: status.info, status.warning
//	  - core help renderer: text.title
func (r ResolvedTheme) CheckRequirements(reqs []Requirement) error {
	if len(reqs) == 0 {
		return nil
	}
	var lines []string
	for _, req := range reqs {
		missing := r.MissingTokens(req)
		if len(missing) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  - %s: %s", req.Consumer, joinTokens(missing)))
	}
	if len(lines) == 0 {
		return nil
	}
	sort.Strings(lines) // deterministic ordering across consumers
	return fmt.Errorf("theme %q is missing tokens required by:\n%s", r.Name(), strings.Join(lines, "\n"))
}

// joinTokens formats a sorted token list as a comma-separated string
// for use inside [CheckRequirements] diagnostics.
func joinTokens(toks []Token) string {
	parts := make([]string, 0, len(toks))
	for _, t := range toks {
		parts = append(parts, string(t))
	}
	return strings.Join(parts, ", ")
}
