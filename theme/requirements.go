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
// [ResolvedTheme]. Extensions and the framework itself produce
// Requirement values so the app can detect, at construction time,
// when an installed theme is missing tokens a consumer needs.
//
// The Consumer field identifies who needs the tokens (typically the
// extension name, "logging extension", or "core help renderer") and
// is folded into the diagnostic message so users can act on the
// signal: "set these tokens in your manifest, or pick a different
// theme."
//
// Requirement is plain data; pass it by value. The framework treats
// the Tokens slice as read-only.
type Requirement struct {
	Consumer string
	Tokens   []Token
}

// Require returns a [Requirement] for the supplied consumer name and
// the tokens it reads. It is the canonical constructor; using a
// struct literal works too but Require keeps call sites tidy:
//
//	func (e *Extension) ThemeRequires() theme.Requirement {
//	    return theme.Require("logging extension",
//	        theme.StatusInfo, theme.StatusWarning, theme.StatusError,
//	        theme.AccentPrimary, theme.TextPrimary,
//	    )
//	}
func Require(consumer string, tokens ...Token) Requirement {
	return Requirement{Consumer: consumer, Tokens: tokens}
}

// CoreRequirements returns the [Requirement] entries the nabat root
// package's own output / help / structured paths read from a
// [ResolvedTheme]. The framework includes this list automatically
// when validating; extensions only need to declare the tokens they
// add on top.
//
// Adding a new core consumer (a new Status*, Text*, etc.) means
// adding to the right Requirement here so missing-token diagnostics
// stay accurate.
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

// HasToken reports whether token t resolves to a non-zero
// [lipgloss.Style] on this [ResolvedTheme] — either directly via
// [Palette.Tokens] or transitively through the alias chain. It is
// the predicate the framework uses for requirement validation;
// consumers querying styles still call [ResolvedTheme.Style], which
// returns the zero style on a miss.
//
// HasToken treats only "set to a non-zero style" as covered. A token
// explicitly set to the zero [lipgloss.Style] is reported as covered
// because the manifest author opted in (the only way to land a zero
// style is an explicit empty styleSpec).
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

// MissingTokens returns the tokens from req that this [ResolvedTheme]
// does not cover (neither directly nor via the alias chain). The
// result is sorted lexically so error messages are deterministic.
//
// An empty return means the requirement is fully satisfied.
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

// CheckRequirements applies every [Requirement] in reqs to this
// [ResolvedTheme] and returns one error per consumer whose tokens are
// not fully covered. The errors are joined; nil means every
// consumer's requirement is satisfied.
//
// Diagnostic format:
//
//	theme "minimal" is missing tokens required by:
//	  - logging extension: status.info, status.warning
//	  - core help renderer: text.title
//
// The framework calls CheckRequirements at App.finalize time. The
// resulting error either blocks construction (when the strict mode
// is on) or is rendered as a warning to stderr (the default).
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
