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
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// HuhFromTokens derives a [huh.Theme] from a per-token style map.
// It is a thin wrapper around [PromptFromTokens] for callers that
// want the legacy "give me a huh.Theme directly" entry point;
// [Theme.Resolve] uses [PromptFromTokens] + [Prompt.Huh] internally
// so the two paths produce identical output.
//
// New code should reach for [PromptFromTokens] (which returns the
// inspectable Nabat-native [Prompt] value) when possible. HuhFromTokens
// stays for ergonomic compatibility with consumers that already
// hold a token map and want a huh.Theme out the other end.
func HuhFromTokens(tokens map[Token]lipgloss.Style) huh.Theme {
	return PromptFromTokens(tokens).Huh()
}

// GlamourPreset picks the upstream glamour style name that fits the
// given variant + capabilities combination. It is the fallback
// [Theme.Resolve] applies when both [Palette.Glamour] and
// [Palette.GlamourFor] are nil and [Palette.GlamourName] is empty.
//
// Mapping:
//
//   - Variant [VariantNoTTY] OR [Capabilities.Interactive] false ->
//     "notty" (plain text).
//   - [Capabilities.Dark] true -> "dark".
//   - Otherwise -> "light".
//
// The catalog and programmatic themes get the same defaults out of
// this one helper.
func GlamourPreset(v Variant, c Capabilities) string {
	if v == VariantNoTTY || !c.Interactive {
		return "notty"
	}
	if c.Dark {
		return "dark"
	}
	return "light"
}

// ChromaPreset picks the upstream chroma style name that fits the
// given variant. It is the fallback [Theme.Resolve] applies when
// [Palette.Chroma] is nil and [Palette.ChromaName] is empty.
//
// Mapping:
//
//   - [VariantDark]  -> "monokai" (lines up with Charm's defaults).
//   - [VariantLight] -> "github"  (high-contrast on light terminals).
//   - [VariantNoTTY] -> ""        (chroma falls back to no styling).
//
// An empty return tells the caller to leave the chroma slot empty and
// let chroma's own default kick in at render time.
func ChromaPreset(v Variant) string {
	switch v {
	case VariantDark:
		return "monokai"
	case VariantLight:
		return "github"
	case VariantNoTTY:
		return ""
	}
	return ""
}
