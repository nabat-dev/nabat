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
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
	"nabat.dev/theme/internal/manifest"
)

// publicSchemaURL pins the URL that manifests reference via $schema. It is
// duplicated here (rather than read out of Schema()) so accidental rename
// of the URL inside the document fails the build on this side too.
const publicSchemaURL = "https://nabat.dev/schemas/theme/v1.json"

// TestSchemaCompiles ensures the embedded schema document is itself a
// well-formed JSON Schema (draft 2020-12). The compiler walks every
// keyword, $ref, and $defs entry; any malformed constraint surfaces here
// rather than as a confusing failure inside TestManifestsMatchSchema.
func TestSchemaCompiles(t *testing.T) {
	t.Parallel()

	_, err := compileSchema(t)
	require.NoError(t, err, "compile embedded schema")
}

// TestSchemaIDMatchesPublicURL guards the public URL contract. Renaming
// the schema file or the constant must break the build, not editor
// validation in the wild for downstream theme authors who already
// reference https://nabat.dev/schemas/theme/v1.json.
func TestSchemaIDMatchesPublicURL(t *testing.T) {
	t.Parallel()

	var doc struct {
		ID string `json:"$id"`
	}
	require.NoError(t, json.Unmarshal(theme.Schema(), &doc))
	assert.Equal(t, publicSchemaURL, doc.ID)
}

// TestSchemaHuhEnumMatchesLoader keeps the JSON Schema's "huh"
// enum in lock-step with the loader's closed adapter catalog.
// Without this guard, drift would let a manifest pass schema
// validation in editors only to fail at registry init with
// errUnknownPromptAdapter.
func TestSchemaHuhEnumMatchesLoader(t *testing.T) {
	t.Parallel()

	var doc struct {
		Defs struct {
			Variant struct {
				Properties struct {
					Huh struct {
						Enum []string `json:"enum"`
					} `json:"huh"`
				} `json:"properties"`
			} `json:"variant"`
		} `json:"$defs"`
	}
	require.NoError(t, json.Unmarshal(theme.Schema(), &doc))
	got := append([]string(nil), doc.Defs.Variant.Properties.Huh.Enum...)
	want := manifest.PromptAdapterNames()
	sort.Strings(got)
	sort.Strings(want)
	assert.Equal(t, want, got)
}

// TestManifestsMatchSchema validates every JSON manifest under data/
// against the embedded schema. Each manifest is validated as a
// subtest so a single bad file does not hide the others; when none of
// the manifests have landed yet the test skips so the schema-level
// tests can still run independently.
func TestManifestsMatchSchema(t *testing.T) {
	t.Parallel()

	s, err := compileSchema(t)
	require.NoError(t, err, "compile embedded schema")

	const localDataDir = "data"
	entries, err := os.ReadDir(localDataDir)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("%s/ directory not present", localDataDir)
	}
	require.NoError(t, err, "read data dir")

	manifestCount := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		manifestCount++
		name := strings.TrimSuffix(e.Name(), ".json")
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			data, readErr := os.ReadFile(filepath.Join(localDataDir, e.Name()))
			require.NoError(t, readErr, "read manifest")

			doc, parseErr := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			require.NoError(t, parseErr, "parse manifest")

			assert.NoError(t, s.Validate(doc), "schema mismatch")
		})
	}
	if len(entries) > 0 && manifestCount == 0 {
		t.Skipf("%s/ contains no .json files yet", localDataDir)
	}
}

// TestConstsHaveManifests is the drift test that ties the untyped
// string constants in names.go to embedded manifest files. Adding a
// new name constant without shipping a data/<name>.json file (or vice
// versa) fails the build, not nabat.New at runtime.
//
// Constants added to names.go must be appended to this list; the test
// is unable to introspect them via reflection because they are untyped
// strings.
func TestConstsHaveManifests(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		theme.Default,
		theme.Minimal,
		theme.Charm,
		theme.Dracula,
		theme.Gruvbox,
		theme.CatppuccinLatte,
		theme.CatppuccinFrappe,
		theme.CatppuccinMacchiato,
		theme.CatppuccinMocha,
		theme.Nabat,
		theme.Nord,
		theme.Solarized,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			th, err := theme.Get(name)
			require.NoError(t, err, "constant %q", name)
			require.NotNil(t, th, "constant %q: Get returned nil theme", name)
		})
	}
}

// TestEveryThemeResolves applies each registered theme against a
// representative set of [theme.Capabilities] and verifies it produces
// a usable [theme.ResolvedTheme] without errors. This catches regressions
// in the manifest loader (unknown chroma name, broken huh adapter
// reference, primitive miss) for every theme in the catalog at once.
func TestEveryThemeResolves(t *testing.T) {
	t.Parallel()

	caps := []theme.Capabilities{
		{Dark: true, Interactive: true},
		{Dark: false, Interactive: true},
		{Dark: true, Interactive: false},
		{Dark: false, Interactive: false},
	}
	for name, th := range theme.All() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, c := range caps {
				_, err := th.ResolveErr(c)
				assert.NoErrorf(t, err, "theme %q failed to resolve for caps=%+v", name, c)
			}
		})
	}
}

// TestRegistryGetUnknownReturnsActionableError verifies the not-found
// path tells the user what they could have written. A short test here
// avoids regressions if someone reformats Get and accidentally drops
// the catalog list from the message.
func TestRegistryGetUnknownReturnsActionableError(t *testing.T) {
	t.Parallel()

	_, err := theme.Get("definitely-not-a-theme")
	require.Error(t, err)
	assert.ErrorContains(t, err, "no theme named")
	assert.ErrorContains(t, err, "available:")
	assert.ErrorContains(t, err, theme.Default)
}

// TestNamesIsSorted catches accidental ordering changes in Names()
// that would break shell-completion stability and snapshot tests in
// downstream consumers.
func TestNamesIsSorted(t *testing.T) {
	t.Parallel()

	got := theme.Names()
	want := append([]string(nil), got...)
	sort.Strings(want)
	assert.Equal(t, want, got, "Names() not sorted")
}

// TestAllReturnsDefensiveCopy guarantees mutations to the returned map
// do not affect the registry. Without this, a downstream tool clearing
// the map would silently break later Get calls.
func TestAllReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	all := theme.All()
	require.Contains(t, all, theme.Default, "All() missing %q", theme.Default)
	delete(all, theme.Default)

	_, err := theme.Get(theme.Default)
	assert.NoError(t, err, "registry mutated by All() caller")
}

// TestAllReturnsDefensiveCopyForNestedMaps guarantees callers cannot
// mutate nested map fields (Variants/Aliases/Tokens) through values
// returned by All().
func TestAllReturnsDefensiveCopyForNestedMaps(t *testing.T) {
	t.Parallel()

	first := theme.All()
	th, ok := first[theme.Default]
	require.True(t, ok, "All() missing %q", theme.Default)

	v := firstVariantKey(t, th)
	p := th.Variants[v]
	if p.Aliases == nil {
		p.Aliases = map[theme.Token]theme.Token{}
	}
	p.Aliases[theme.Token("mutated.alias")] = theme.TextPrimary
	th.Variants[v] = p
	first[theme.Default] = th

	second, err := theme.Get(theme.Default)
	require.NoError(t, err)
	secondPalette := second.Variants[v]
	_, leaked := secondPalette.Aliases[theme.Token("mutated.alias")]
	assert.False(t, leaked, "nested alias mutation leaked into catalog state")
}

// TestGetReturnsDefensiveCopyForNestedMaps ensures mutating the Theme
// returned by Get() does not corrupt the cached registry entry used by
// later Get()/All() calls.
func TestGetReturnsDefensiveCopyForNestedMaps(t *testing.T) {
	t.Parallel()

	first, err := theme.Get(theme.Default)
	require.NoError(t, err)

	v := firstVariantKey(t, first)
	p := first.Variants[v]
	if p.Aliases == nil {
		p.Aliases = map[theme.Token]theme.Token{}
	}
	p.Aliases[theme.Token("mutated.from.get")] = theme.TextPrimary
	first.Variants[v] = p

	second, err := theme.Get(theme.Default)
	require.NoError(t, err)
	secondPalette := second.Variants[v]
	_, leaked := secondPalette.Aliases[theme.Token("mutated.from.get")]
	assert.False(t, leaked, "Get() returned shared nested map state")
}

// TestDefaultThemeResolvesAcrossAllCapabilities verifies that default.json
// resolves to a usable [theme.ResolvedTheme] without errors across every
// Capabilities permutation. The dark/light/notty variants collectively cover
// all four combinations so pickVariant always finds a declared variant.
func TestDefaultThemeResolvesAcrossAllCapabilities(t *testing.T) {
	t.Parallel()

	th, err := theme.Get(theme.Default)
	require.NoError(t, err)

	for _, c := range []theme.Capabilities{
		{Dark: true, Interactive: true},
		{Dark: false, Interactive: true},
		{Dark: true, Interactive: false},
		{Dark: false, Interactive: false},
	} {
		t.Run("dark="+boolStr(c.Dark)+"_interactive="+boolStr(c.Interactive), func(t *testing.T) {
			t.Parallel()

			rt, resolveErr := th.ResolveErr(c)
			require.NoError(t, resolveErr)
			assert.Equal(t, theme.Default, rt.Name())
			assert.True(t, rt.Style(theme.StatusSuccess).GetBold(),
				"StatusSuccess.Bold not set; default theme should have populated it")
			assert.NotNil(t, rt.Huh(),
				"Huh() is nil; HuhFromTokens fallback should have populated it")
		})
	}
}

// TestManifestVariantPropagatesToResolvedTheme verifies that the
// variant [Theme.Resolve] picks for a given Capabilities snapshot is
// one of the variants the manifest actually declares. This is the
// round-trip a future `--theme-variant` flag depends on: pick a
// declared variant, surface it on the resolved theme.
func TestManifestVariantPropagatesToResolvedTheme(t *testing.T) {
	t.Parallel()

	th, err := theme.Get(theme.Default)
	require.NoError(t, err)
	r, err := th.ResolveErr(theme.Capabilities{Dark: true, Interactive: true})
	require.NoError(t, err)
	assert.NotEqual(t, theme.VariantUnset, r.Variant(),
		"Variant should be set from the manifest")

	// The default manifest declares "dark"; confirm via Manifest()
	// rather than hardcoding so a future palette flip in default.json
	// doesn't break the round-trip test.
	m, err := theme.Manifest(theme.Default)
	require.NoError(t, err)
	assert.Contains(t, m.Variants, string(r.Variant()),
		"resolved variant must be one of the manifest's declared variants")
}

// TestManifestReturnsCatalogMetadata verifies the accessor surfaces
// every public manifest field — name, description, declared variants,
// tokens — without invoking [Theme.Resolve]. A future
// `nabat themes list` subcommand depends on this read-only path;
// resolving themes just to list them would force callers to fabricate
// a [theme.Capabilities].
func TestManifestReturnsCatalogMetadata(t *testing.T) {
	t.Parallel()

	m, err := theme.Manifest(theme.Default)
	require.NoError(t, err)
	assert.Equal(t, theme.Default, m.Name)
	assert.NotEmpty(t, m.Variants, "Variants should not be empty; manifest declares at least one variant")
	assert.IsIncreasing(t, m.Variants, "Variants must be sorted")
	assert.NotEmpty(t, m.TokenNames, "TokenNames should not be empty; default theme declares tokens")
	assert.IsIncreasing(t, m.TokenNames, "TokenNames must be sorted")
}

// TestManifestUnknownNameReturnsActionableError mirrors the Get error
// shape so tooling that switches between Get and Manifest can rely on
// the same diagnostics.
func TestManifestUnknownNameReturnsActionableError(t *testing.T) {
	t.Parallel()

	_, err := theme.Manifest("definitely-not-a-theme")
	require.Error(t, err)
	assert.ErrorContains(t, err, "no theme named")
	assert.ErrorContains(t, err, "available:")
}

// TestManifestReturnsDefensiveCopy guarantees that callers mutating
// the returned TokenNames slice (sorting, deduping) cannot corrupt
// future calls.
func TestManifestReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	first, err := theme.Manifest(theme.Default)
	require.NoError(t, err)
	if len(first.TokenNames) == 0 {
		t.Skip("default theme declares no tokens")
	}
	first.TokenNames[0] = "MUTATED"

	second, err := theme.Manifest(theme.Default)
	require.NoError(t, err)
	assert.NotEqual(t, "MUTATED", second.TokenNames[0],
		"Manifest returned the same backing slice across calls; mutation leaked")
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func firstVariantKey(t *testing.T, th theme.Theme) theme.Variant {
	t.Helper()
	require.NotEmpty(t, th.Variants, "theme must declare at least one variant")
	for v := range th.Variants {
		return v
	}
	require.FailNow(t, "unreachable: non-empty map had no keys")
	return theme.VariantUnset
}

// compileSchema parses the embedded schema bytes and compiles them via the
// jsonschema/v6 compiler. Both schema-level tests (TestSchemaCompiles and
// TestManifestsMatchSchema) share this helper so a malformed schema
// surfaces in a single place.
func compileSchema(t *testing.T) (*jsonschema.Schema, error) {
	t.Helper()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(theme.Schema()))
	if err != nil {
		return nil, err
	}
	c := jsonschema.NewCompiler()
	if addErr := c.AddResource(publicSchemaURL, doc); addErr != nil {
		return nil, addErr
	}
	return c.Compile(publicSchemaURL)
}
