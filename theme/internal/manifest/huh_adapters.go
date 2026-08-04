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

package manifest

import (
	"fmt"
	"sort"

	"charm.land/huh/v2"
)

// upstreamPromptAdapters is the closed catalog of named huh wrappers
// for the manifest "huh" field. Custom looks use theme.Palette.Huh;
// there is no public registration API.
var upstreamPromptAdapters = map[string]huh.Theme{
	"charm":      huh.ThemeFunc(huh.ThemeCharm),
	"base16":     huh.ThemeFunc(huh.ThemeBase16),
	"dracula":    huh.ThemeFunc(huh.ThemeDracula),
	"catppuccin": huh.ThemeFunc(huh.ThemeCatppuccin),
}

// lookupPromptAdapter returns the registered adapter for name.
func lookupPromptAdapter(name string) (huh.Theme, bool) {
	t, ok := upstreamPromptAdapters[name]
	return t, ok
}

// PromptAdapterNames returns the sorted set of supported huh adapter
// names (for schema-vs-loader parity checks).
func PromptAdapterNames() []string {
	out := make([]string, 0, len(upstreamPromptAdapters))
	for k := range upstreamPromptAdapters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// errUnknownPromptAdapter formats the "unsupported huh adapter" error.
// Pass empty variant when the value is not variant-scoped.
func errUnknownPromptAdapter(manifest, variant, name string) error {
	scope := fmt.Sprintf("manifest %q", manifest)
	if variant != "" {
		scope = fmt.Sprintf("%s variant %q", scope, variant)
	}
	return fmt.Errorf(
		"%s: \"huh\": %q is not a recognized adapter (supported: %v); for custom styling set theme.Palette.Huh via nabat.WithCustomTheme",
		scope, name, PromptAdapterNames(),
	)
}
