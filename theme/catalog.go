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

// embedded holds schema/v1.json and data/*.json baked into the binary.
//
//go:embed schema/*.json data/*.json
var embedded embed.FS

const (
	schemaPath = "schema/v1.json"
	dataDir    = "data"
)

// rawCatalog stores embedded manifest bytes keyed by registry name
// (filename without .json). Read-only after init; concurrent reads are
// safe without a lock.
var rawCatalog = map[string][]byte{}

// parsedCatalog memoizes [manifest.Parse] results keyed by registry name.
var parsedCatalog sync.Map // map[string]Theme

// manifestMetaCache memoizes [Manifest] projections. Entries are never
// invalidated; [Manifest] still returns a defensive copy each call.
var (
	manifestMetaMu    sync.RWMutex
	manifestMetaCache = map[string]Metadata{}
)

// init loads raw embedded manifest bytes. It does not parse JSON;
// parsing is deferred to [Get]. A read failure panics (broken build).
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

// Get returns the [Theme] registered under name. On a miss the error
// lists every available name:
//
//	t, err := theme.Get("draculaa") // typo
//	// err: nabat/theme: no theme named "draculaa"; available: [...]
//
// The first call parses and caches the embedded manifest; later calls
// return a cloned cached [Theme]. Parse errors surface on first access.
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

// themeFromCompiled converts a [manifest.Compiled] into a [Theme].
// Lives here so the parser never imports theme.
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
			BorderColor:      c.PromptKnobs.BorderColor,
		}
	}

	return Theme{
		Name:        c.Name,
		Default:     Variant(c.Default),
		PromptKnobs: knobs,
		Variants:    variants,
	}
}

// paletteFromCompiledVariant wraps one manifest variant into a [Palette].
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

// Names returns the registered theme names in lexical order.
func Names() []string {
	out := make([]string, 0, len(rawCatalog))
	for k := range rawCatalog {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// All returns a defensive copy of the registry, parsing every manifest.
// Callers may mutate the map without affecting the registry.
//
// Parse errors panic. For per-theme error handling, iterate [Names]
// and call [Get] instead.
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

// Schema returns the embedded JSON Schema for the theme manifest format.
// The slice is a fresh copy on each call. A read failure panics
// (broken build).
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
// [Manifest] returns a fresh value each call; mutating slices in place
// is safe and does not affect the registry.
type Metadata struct {
	// Name is the manifest "name" field and [Get] registry key.
	Name string

	// Description is the manifest "description" field, or empty.
	Description string

	// Default is the fallback variant for [Theme.Resolve]. Empty for
	// single-variant themes.
	Default string

	// Variants is the sorted list of declared variant keys. Always
	// non-empty (schema requires at least one).
	Variants []string

	// TokenNames is the sorted, deduplicated set of token paths any
	// variant declares.
	TokenNames []string
}

// rawManifestMeta is the minimal JSON projection [Manifest] needs.
type rawManifestMeta struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description,omitempty"`
	Default     string                      `json:"default,omitempty"`
	Variants    map[string]rawManifestSlice `json:"variants"`
}

// rawManifestSlice is the per-variant projection [Manifest] needs.
type rawManifestSlice struct {
	Tokens map[string]json.RawMessage `json:"tokens"`
}

// Manifest returns [Metadata] for a built-in theme without resolving it.
// Unknown names return the same "available: [...]" error shape as [Get].
// The returned value is freshly constructed; the decode is memoized.
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

	meta, err := loadManifestMeta(name)
	if err != nil {
		return Metadata{}, err
	}

	manifestMetaMu.Lock()
	manifestMetaCache[name] = meta
	manifestMetaMu.Unlock()

	return copyMetadata(meta), nil
}

// loadManifestMeta decodes embedded manifest JSON for name into
// [Metadata]. Unknown names and decode failures return errors.
func loadManifestMeta(name string) (Metadata, error) {
	data, ok := rawCatalog[name]
	if !ok {
		return Metadata{}, fmt.Errorf("nabat/theme: no theme named %q; available: [%s]", name, strings.Join(Names(), ", "))
	}

	var raw rawManifestMeta
	if unmarshalErr := json.Unmarshal(data, &raw); unmarshalErr != nil {
		return Metadata{}, fmt.Errorf("nabat/theme: extract metadata for %s: %w", name, unmarshalErr)
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
	}, nil
}

// copyMetadata returns a shallow copy of m with fresh TokenNames and
// Variants slices.
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
