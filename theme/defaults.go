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

// HuhFromTokens derives a [huh.Theme] from a per-token style map via
// [PromptFromTokens]. Prefer [PromptFromTokens] when the Nabat-native
// [Prompt] value is useful; this stays for callers that want huh.Theme
// directly.
func HuhFromTokens(tokens map[Token]lipgloss.Style) huh.Theme {
	return PromptFromTokens(tokens).Huh()
}

// GlamourPreset picks the upstream glamour style name for v and c.
// [Theme.Resolve] uses it when glamour slots are empty.
//
//   - [VariantNoTTY] or non-interactive -> "notty"
//   - [Capabilities.Dark] -> "dark"
//   - otherwise -> "light"
func GlamourPreset(v Variant, c Capabilities) string {
	if v == VariantNoTTY || !c.Interactive {
		return "notty"
	}
	if c.Dark {
		return "dark"
	}
	return "light"
}

// ChromaPreset picks the upstream chroma style name for v.
// [Theme.Resolve] uses it when chroma slots are empty.
//
//   - [VariantDark] -> "monokai"
//   - [VariantLight] -> "github"
//   - [VariantNoTTY] / unset -> "" (chroma default at render time)
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
