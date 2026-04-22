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

package theme_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// TestBundledManifestTokenKeysMatchDefault verifies every embedded JSON
// manifest uses the same token key set as [theme/data/default.json] per
// variant kind (dark / light / notty) whenever that variant is present.
func TestBundledManifestTokenKeysMatchDefault(t *testing.T) {
	t.Parallel()

	defaultKeys := loadCanonicalTokenKeys(t)
	const dataDir = "data"
	entries, err := os.ReadDir(dataDir)
	require.NoError(t, err)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// #nosec G304 -- reads theme/data/*.json only (same pattern as catalog_test).
			raw, readErr := os.ReadFile(filepath.Join(dataDir, e.Name()))
			require.NoError(t, readErr)

			var doc struct {
				Variants map[string]struct {
					Tokens map[string]json.RawMessage `json:"tokens"`
				} `json:"variants"`
			}
			require.NoError(t, json.Unmarshal(raw, &doc))

			for _, variant := range []struct {
				key   theme.Variant
				label string
			}{
				{theme.VariantDark, "dark"},
				{theme.VariantLight, "light"},
				{theme.VariantNoTTY, "notty"},
			} {
				vp, ok := doc.Variants[variant.label]
				if !ok {
					continue
				}
				want, ok := defaultKeys[variant.key]
				require.True(t, ok, "default theme must define variant %q", variant.label)

				got := keysOf(vp.Tokens)
				require.Equal(t, want, got,
					"theme %q variant %q token keys must match default.json", name, variant.label)
			}
		})
	}
}

func loadCanonicalTokenKeys(t *testing.T) map[theme.Variant][]string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("data", "default.json"))
	require.NoError(t, err)

	var doc struct {
		Variants map[string]struct {
			Tokens map[string]json.RawMessage `json:"tokens"`
		} `json:"variants"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))

	out := make(map[theme.Variant][]string)
	for _, tc := range []struct {
		variant theme.Variant
		label   string
	}{
		{theme.VariantDark, "dark"},
		{theme.VariantLight, "light"},
		{theme.VariantNoTTY, "notty"},
	} {
		vp, ok := doc.Variants[tc.label]
		require.True(t, ok, "default.json must define variant %q", tc.label)
		out[tc.variant] = keysOf(vp.Tokens)
	}
	return out
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
