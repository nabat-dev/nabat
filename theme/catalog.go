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
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"

	"nabat.dev/theme/internal/manifest"
)

// embedded holds files baked into the binary at build time:
//
//   - schema/v1.json: the JSON Schema describing the manifest format
//     (also exposed via [Schema]).
//   - data/*.json: the catalog of built-in theme manifests parsed
//     lazily on first [Get] / [All] call.
//
//go:embed schema/*.json data/*.json
var embedded embed.FS

const (
	schemaPath = "schema/v1.json"
	dataDir    = "data"
)

// rawCatalog stores the embedded manifest bytes keyed by registry name
// (filename without .json). It is populated once at init from the
// embedded filesystem; each entry is a slice header pointing at bytes
// already in the binary, so init does no JSON parsing.
//
// The catalog is read-only after init; concurrent reads are safe and
// no lock is needed for lookups into rawCatalog itself.
var rawCatalog = map[string][]byte{}

// parsedCatalog memoizes the result of [manifest.Parse] for each
// registry name. The first [Get] for a name parses the manifest and
// stores the resulting [Theme]; subsequent calls return the cached
// value without re-parsing.
//
// Using [sync.Map] keeps the lookup path lock-free in the common case
// (cached hit) while still serializing the parse on first miss.
var parsedCatalog sync.Map // map[string]Theme

// manifestMetaCache memoizes the JSON-decoded projection used by
// [Manifest]. The cache key is the registry name; the value is the
// canonical [Metadata] for that name. Cached entries are populated on
// first [Manifest] call and never invalidated — the underlying
// embedded bytes are immutable for the lifetime of the process.
//
// [Manifest] still returns a defensive copy on every call so a caller
// that mutates [Metadata.TokenNames] in place does not corrupt the
// cache for the next caller.
var (
	manifestMetaMu    sync.RWMutex
	manifestMetaCache = map[string]Metadata{}
)

// init walks the embedded data/ directory at process start and stores
// the raw bytes for every *.json manifest under its base filename
// (without extension). It does NOT parse the JSON — parsing is
// deferred to [Get] so importing the package costs nothing for
// callers that only need [Token] constants or [Style] lookups.
//
// A read failure here panics. The catalog is part of the binary; a
// missing or unreadable manifest file is a broken build, and surfacing
// it as a runtime error from nabat.New later would just hide the bug.
// The drift test in catalog_test.go ensures every name constant lines
// up with a stored manifest, so a missing or renamed file fails CI
// before any user runs the binary.
func init() {
	entries, err := fs.ReadDir(embedded, dataDir)
	if err != nil {
		panic("nabat/theme: read embedded data dir: " + err.Error())
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".json")
		data, readErr := fs.ReadFile(embedded, dataDir+"/"+e.Name())
		if readErr != nil {
			panic("nabat/theme: read manifest " + e.Name() + ": " + readErr.Error())
		}
		rawCatalog[name] = data
	}
}

// Get returns the [Theme] registered under name. The error lists every
// available name on a miss, so callers and CLI users get an actionable
// diagnosis rather than just "not found":
//
//	t, err := theme.Get("draculaa") // typo
//	// err: nabat/theme: no theme named "draculaa"; available: [default minimal charm dracula catppuccin-mocha nabat]
//
// The first [Get] for a name parses the embedded manifest and caches
// the result; subsequent calls return the cached value without
// re-parsing. Parse errors surface here on first access; once cached,
// later callers always receive the parsed [Theme] without re-checking.
func Get(name string) (Theme, error) {
	if cached, ok := parsedCatalog.Load(name); ok {
		t, typeOk := cached.(Theme)
		if !typeOk {
			// Defensive: only Theme values land in parsedCatalog
			// (themeFromCompiled returns Theme exclusively), so this
			// branch is unreachable in practice. Treating it as a
			// cache miss lets Get re-parse rather than panic.
			parsedCatalog.Delete(name)
		} else {
			return cloneTheme(t), nil
		}
	}
	data, ok := rawCatalog[name]
	if !ok {
		return Theme{}, fmt.Errorf("nabat/theme: no theme named %q; available: [%s]", name, strings.Join(Names(), ", "))
	}
	compiled, err := manifest.Parse(data)
	if err != nil {
		return Theme{}, fmt.Errorf("nabat/theme: parse manifest %q: %w", name, err)
	}
	t := themeFromCompiled(compiled)
	parsedCatalog.Store(name, t)
	return cloneTheme(t), nil
}

// themeFromCompiled converts a [manifest.Compiled] into a [Theme]
// struct value. Each entry in the manifest's "variants" map becomes a
// [Palette] under the corresponding [Variant] key.
//
// This conversion is a mechanical boundary copy: token and alias maps
// become theme-token maps, integration preset names are copied onto the
// palette fields, and root-level prompt knobs become [Theme.PromptKnobs].
//
// This conversion lives in the theme package (not in manifest) so the
// parser stays free of any imports back to theme — that's the
// invariant that breaks the dependency cycle between the registry
// and the parser.
func themeFromCompiled(c *manifest.Compiled) Theme {
	variants := make(map[Variant]Palette, len(c.Variants))
	for variantKey, cv := range c.Variants {
		variants[Variant(variantKey)] = paletteFromCompiledVariant(cv)
	}

	var knobs PromptKnobs
	if c.PromptKnobs != nil {
		knobs = PromptKnobs{
			SelectedPrefix:   c.PromptKnobs.SelectedPrefix,
			UnselectedPrefix: c.PromptKnobs.UnselectedPrefix,
			Border:           c.PromptKnobs.Border,
		}
	}

	return Theme{
		Name:        c.Name,
		Default:     Variant(c.Default),
		PromptKnobs: knobs,
		Variants:    variants,
	}
}

// paletteFromCompiledVariant wraps a single manifest variant payload
// into a [Palette]. It is split out so [themeFromCompiled] reads as a
// straight per-variant fan-out.
func paletteFromCompiledVariant(cv *manifest.CompiledVariant) Palette {
	tokens := make(map[Token]lipgloss.Style, len(cv.Tokens))
	for k, v := range cv.Tokens {
		tokens[Token(k)] = v
	}

	var aliases map[Token]Token
	if len(cv.Aliases) > 0 {
		aliases = make(map[Token]Token, len(cv.Aliases))
		for k, v := range cv.Aliases {
			aliases[Token(k)] = Token(v)
		}
	}

	p := Palette{
		Tokens:      tokens,
		Aliases:     aliases,
		ChromaName:  cv.ChromaName,
		GlamourName: cv.GlamourName,
		Huh:         cv.HuhTheme,
	}
	return p
}

// Names returns the registered theme names in lexical order. It is
// suitable for shell completion, documentation, and the
// "available: [...]" segment of error messages built elsewhere.
func Names() []string {
	out := make([]string, 0, len(rawCatalog))
	for k := range rawCatalog {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// All returns a defensive copy of the registry, parsing every manifest
// on the way out. It is intended for tools that want to iterate the
// catalog (for example a docs generator rendering one section per
// theme); callers may mutate the returned map without affecting the
// registry.
//
// Manifest parse errors panic — the catalog is part of the binary,
// and a tool walking every theme cannot meaningfully recover from a
// half-broken catalog. Callers that want graceful per-theme error
// handling should iterate [Names] and call [Get] in a loop instead.
func All() map[string]Theme {
	out := make(map[string]Theme, len(rawCatalog))
	for name := range rawCatalog {
		t, err := Get(name)
		if err != nil {
			panic("nabat/theme: All: " + err.Error())
		}
		out[name] = t
	}
	return out
}

// Schema returns the embedded JSON Schema describing the theme manifest
// format. Tools that need to validate manifests (a future
// `nabat themes validate` subcommand, a static-site generator, etc.)
// can serve these bytes directly without re-hosting.
//
// The returned slice is a fresh copy on each call; callers may mutate
// it without affecting subsequent calls.
func Schema() []byte {
	b, err := fs.ReadFile(embedded, schemaPath)
	if err != nil {
		// The schema is embedded at compile time. A read failure here
		// means the binary was built incorrectly; there is nothing
		// useful to do at runtime besides surface the bug loudly.
		panic("nabat/theme: " + err.Error())
	}
	return b
}

// Metadata describes a built-in manifest without invoking [Theme.Resolve].
// A future `nabat themes list` subcommand and any documentation
// generator (or IDE plugin) can introspect what a theme advertises —
// its name, default variant, the set of declared variants, and the
// set of tokens any variant covers — by calling [Manifest] rather
// than resolving the theme against a fabricated [Capabilities].
//
// Metadata is read-only and constructed fresh on each [Manifest] call;
// mutating the returned value (for example sorting TokenNames or
// Variants in place) is safe and does not affect the registry.
type Metadata struct {
	// Name is the manifest's "name" field, identical to the registry
	// key used by [Get].
	Name string

	// Description is the manifest's "description" field. Empty when
	// the manifest omitted it.
	Description string

	// Default is the manifest's "default" field — the variant
	// [Theme.Resolve] falls back to when capabilities don't pin a
	// pick. Empty for single-variant themes.
	Default string

	// Variants is the sorted list of variant keys this manifest
	// declares ("dark" / "light" / "notty"). Always non-empty (the
	// schema requires at least one entry).
	Variants []string

	// TokenNames is the sorted, deduplicated set of token paths any
	// variant declares under its "tokens" map. Tools surfacing
	// "what does this theme cover?" use this without having to
	// re-aggregate per variant.
	TokenNames []string
}

// rawManifestMeta is the minimal JSON projection [Manifest] needs.
// Decoding only these fields keeps the accessor cheap and decouples
// it from the full manifest schema (chromaStyle / glamourStyle /
// huhStyle parsing lives in internal/manifest and stays there).
type rawManifestMeta struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Default     string                      `json:"default,omitempty"`
	Variants    map[string]rawManifestSlice `json:"variants"`
}

// rawManifestSlice is the per-variant projection [Manifest] needs.
// Only "tokens" matters here — every other field is parser detail
// the introspection accessor doesn't surface.
type rawManifestSlice struct {
	Tokens map[string]json.RawMessage `json:"tokens"`
}

// Manifest returns the [Metadata] for a built-in theme without
// invoking the recipe. Use it in tooling — `nabat themes list`,
// shell-completion descriptions, documentation generators — that needs
// to inspect what a theme advertises rather than apply it.
//
// The error mirrors [Get]: an unknown name returns an actionable
// "available: [...]" listing so CLI users can recover from typos
// without consulting the docs.
//
// The returned [Metadata] is freshly constructed on each call; callers
// may sort or modify TokenNames in place without affecting subsequent
// calls or the registry. The underlying decode is memoized internally
// so repeated calls (e.g. from a `themes list` subcommand iterating
// every name) do not re-parse the embedded JSON.
func Manifest(name string) (Metadata, error) {
	if _, ok := rawCatalog[name]; !ok {
		return Metadata{}, fmt.Errorf("nabat/theme: no theme named %q; available: [%s]", name, strings.Join(Names(), ", "))
	}

	manifestMetaMu.RLock()
	cached, ok := manifestMetaCache[name]
	manifestMetaMu.RUnlock()
	if ok {
		return copyMetadata(cached), nil
	}

	meta := loadManifestMeta(name)

	manifestMetaMu.Lock()
	manifestMetaCache[name] = meta
	manifestMetaMu.Unlock()

	return copyMetadata(meta), nil
}

// loadManifestMeta reads and decodes the embedded manifest JSON for
// name into a canonical [Metadata]. It is only called from [Manifest]
// on a cache miss; the result is stored in [manifestMetaCache] and
// reused for every subsequent call.
//
// A failure here is treated as a build-time invariant break, the same
// way init() handles a missing or malformed manifest: panic loudly
// rather than return a misleading partial value.
func loadManifestMeta(name string) Metadata {
	data, ok := rawCatalog[name]
	if !ok {
		panic("nabat/theme: loadManifestMeta: unknown name " + name)
	}

	var raw rawManifestMeta
	if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
		panic("nabat/theme: extract metadata for " + name + ": " + unmarshalErr.Error())
	}

	variants := make([]string, 0, len(raw.Variants))
	tokenSet := map[string]struct{}{}
	for vKey, vSlice := range raw.Variants {
		variants = append(variants, vKey)
		for tk := range vSlice.Tokens {
			tokenSet[tk] = struct{}{}
		}
	}
	sort.Strings(variants)

	tokens := make([]string, 0, len(tokenSet))
	for tk := range tokenSet {
		tokens = append(tokens, tk)
	}
	sort.Strings(tokens)

	return Metadata{
		Name:        raw.Name,
		Description: raw.Description,
		Default:     raw.Default,
		Variants:    variants,
		TokenNames:  tokens,
	}
}

// copyMetadata returns a defensive shallow copy of m with fresh
// TokenNames and Variants slices. Callers of [Manifest] are documented
// to be free to sort or otherwise mutate the slices in place; the
// cache must not observe those mutations.
func copyMetadata(m Metadata) Metadata {
	out := m
	if m.TokenNames != nil {
		out.TokenNames = append([]string(nil), m.TokenNames...)
	}
	if m.Variants != nil {
		out.Variants = append([]string(nil), m.Variants...)
	}
	return out
}
