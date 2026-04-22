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

// upstreamPromptAdapters is the **closed** catalog of named huh
// wrappers reachable through the manifest's "huh": "<name>" field.
// Custom themes that want a different look use [theme.Palette.Huh]
// directly (programmatic huh.Theme escape).
//
// The catalog is closed by design: every entry is just a thin wrapper
// around a function the upstream charm.land/huh/v2 package already
// exports, and reproducing those entire themes inline would be
// thousands of lines per manifest. The first time a downstream user
// genuinely needs a fifth wrapper, an entry is one line — but until
// then, three (plus base16 for low-color terminals) is the canonical
// set.
//
// There is no public registration API. The closed catalog is
// deliberate: every styling decision a downstream theme might want
// now has the open theme.Palette.Huh field (programmatic).
var upstreamPromptAdapters = map[string]huh.Theme{
	"charm":      huh.ThemeFunc(huh.ThemeCharm),
	"base16":     huh.ThemeFunc(huh.ThemeBase16),
	"dracula":    huh.ThemeFunc(huh.ThemeDracula),
	"catppuccin": huh.ThemeFunc(huh.ThemeCatppuccin),
}

// lookupPromptAdapter returns the registered adapter for name, or
// false. It is unexported because callers outside this package work
// in terms of manifest names, not the adapter registry directly.
func lookupPromptAdapter(name string) (huh.Theme, bool) {
	t, ok := upstreamPromptAdapters[name]
	return t, ok
}

// PromptAdapterNames returns the sorted set of supported huh
// adapter names. It exists so external schema-vs-loader parity tests
// can confirm that the JSON Schema enum stays in lock-step with the
// catalog the loader actually accepts.
func PromptAdapterNames() []string {
	out := make([]string, 0, len(upstreamPromptAdapters))
	for k := range upstreamPromptAdapters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// errUnknownPromptAdapter formats the standard "did you mean" error
// for a huh field that does not match any catalog entry.
//
// manifest is always the manifest name. variant is optional; pass empty
// when the offending huh value came from a manifest-wide context.
func errUnknownPromptAdapter(manifest, variant, name string) error {
	scope := fmt.Sprintf("manifest %q", manifest)
	if variant != "" {
		scope = fmt.Sprintf("%s variant %q", scope, variant)
	}
	return fmt.Errorf(
		"%s: \"huh\": %q is not a recognized adapter; supported adapters: %v. For custom prompt styling set theme.Palette.Huh directly via nabat.WithCustomTheme",
		scope, name, PromptAdapterNames(),
	)
}
