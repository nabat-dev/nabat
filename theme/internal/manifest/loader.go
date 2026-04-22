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
	"errors"
	"fmt"
	"sort"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"

	glamourstyles "charm.land/glamour/v2/styles"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

var errNilPromptKnobs = errors.New("prompt knobs input is nil")

// Capabilities mirrors [nabat.dev/theme.Capabilities] inside the
// manifest package so the parser can branch on terminal facts without
// importing the theme package (which would create a cycle —
// theme/catalog.go imports this package). The catalog converts at the
// boundary by passing field-by-field copies into [CompiledVariant.GlamourFor].
type Capabilities struct {
	Dark        bool
	Profile     colorprofile.Profile
	Interactive bool
}

// Compiled is the intermediate result [Parse] produces from a
// multi-variant manifest. It carries every value the catalog needs to
// assemble a [nabat.dev/theme.Theme] struct, but contains no references
// back to the theme package itself — that keeps the dependency
// direction strictly downstream (theme -> manifest, never the
// reverse).
//
// Variants are keyed by mode string ("dark" / "light" / "notty") so
// the catalog can map directly to [nabat.dev/theme.Variant] at the
// boundary.
type Compiled struct {
	// Name is the manifest's "name" field, identical to the registry
	// key the catalog stores it under.
	Name string

	// Default is the manifest's "default" field — the variant the
	// catalog should fall back to when the runtime Capabilities do
	// not match any declared variant. Empty for single-variant
	// themes (the lone variant is always picked).
	Default string

	// Variants holds one [CompiledVariant] per mode declared in the
	// manifest's "variants" map.
	Variants map[string]*CompiledVariant

	// PromptKnobs are theme-wide non-color prompt settings.
	PromptKnobs *PromptKnobs
}

// CompiledVariant is the per-variant style data extracted from one
// entry under the manifest's "variants" map. Token names are plain
// strings (the catalog wraps them as [theme.Token] at the boundary);
// chroma / glamour / huh slots carry the selected upstream preset
// names and adapter.
type CompiledVariant struct {
	// Tokens is the resolved per-token style map keyed by dotted
	// token name. Every entry came through the styleResolver so
	// $primitive / $token / inline color refs have already been
	// collapsed into concrete lipgloss styles.
	Tokens map[string]lipgloss.Style

	// Aliases is the per-variant alias overrides as plain strings.
	// The catalog merges these onto theme.DefaultAliases at
	// theme.Theme.Resolve time. An empty value disables the
	// framework default for that key without substituting another;
	// see theme.Palette.Aliases for the full semantics.
	Aliases map[string]string

	// ChromaName is the upstream chroma style name the variant
	// declared via the "chroma" field.
	ChromaName string

	// GlamourName is the upstream glamour preset name the variant
	// declared via the "glamour" field.
	GlamourName string

	// HuhTheme is the framework-owned [huh.Theme] resolved from the
	// per-variant "huh" adapter.
	HuhTheme huh.Theme
}

// PromptKnobs stores theme-wide non-color prompt controls.
type PromptKnobs struct {
	SelectedPrefix   string
	UnselectedPrefix string
	Border           lipgloss.Border
}

// Parse decodes a manifest into a [*Compiled] intermediate. The
// catalog then assembles the final [nabat.dev/theme.Theme] from this
// value, applying capability-aware defaults for any optional field the
// manifest omitted.
//
// Errors from parsing the JSON shape, validating cross-field
// constraints, or resolving primitive / $token references surface here
// at registry-load time. Errors that depend on the runtime
// Capabilities (only the inline glamourStyle path can produce one) are
// surfaced later via [CompiledVariant.GlamourFor] — the catalog
// forwards them through [theme.Theme.ResolveErr].
//
// Parse is the only symbol this package exports beyond the
// [Capabilities] type and the per-parser helpers ([HuhAdapterNames]);
// everything else is an implementation detail of the parser pipeline.
func Parse(data []byte) (*Compiled, error) {
	rt, parseErr := unmarshalManifest(data)
	if parseErr != nil {
		return nil, fmt.Errorf("decode manifest: %w", parseErr)
	}
	if validErr := validateManifest(rt); validErr != nil {
		return nil, validErr
	}

	out := &Compiled{
		Name:     rt.Name,
		Default:  string(rt.Default),
		Variants: make(map[string]*CompiledVariant, len(rt.Variants)),
	}
	if rt.PromptKnobs != nil {
		knobs, err := parsePromptKnobs(rt.PromptKnobs)
		if err != nil {
			return nil, fmt.Errorf("manifest %q: promptKnobs: %w", rt.Name, err)
		}
		out.PromptKnobs = knobs
	}

	for variantKey, slice := range rt.Variants {
		cv, err := parseVariant(rt.Name, string(variantKey), slice)
		if err != nil {
			return nil, fmt.Errorf("manifest %q: variant %q: %w", rt.Name, variantKey, err)
		}
		out.Variants[string(variantKey)] = cv
	}

	return out, nil
}

// parseVariant compiles one entry under the manifest's "variants" map.
func parseVariant(themeName, variantName string, slice rawSlice) (*CompiledVariant, error) {
	resolver := newStyleResolver(slice.Primitives, slice.Tokens)
	resolved := make(map[string]lipgloss.Style, len(slice.Tokens))
	for name := range slice.Tokens {
		s, resolveErr := resolver.resolveToken(name)
		if resolveErr != nil {
			return nil, resolveErr
		}
		resolved[name] = s
	}

	cv := &CompiledVariant{
		Tokens:      resolved,
		Aliases:     slice.Aliases,
		ChromaName:  slice.Chroma,
		GlamourName: slice.Glamour,
	}

	if slice.Huh != "" {
		t, ok := lookupPromptAdapter(slice.Huh)
		if !ok {
			return nil, errUnknownPromptAdapter(themeName, variantName, slice.Huh)
		}
		cv.HuhTheme = t
	}

	return cv, nil
}

// validateManifest runs structural and cross-field checks that are not
// already enforced by [json.Decoder.DisallowUnknownFields] or the JSON
// Schema. It surfaces every problem at once via [errors.Join] so authors
// see the full diagnosis on one parse instead of fixing-and-retrying.
func validateManifest(rt *rawTheme) error {
	var errs []error
	if rt.Name == "" {
		errs = append(errs, errors.New("manifest: name is required"))
	}
	if len(rt.Variants) == 0 {
		errs = append(errs, errors.New("manifest: variants must contain at least one entry"))
	}

	// "default" must reference a declared variant. Single-variant
	// themes can skip it entirely; multi-variant themes need it to
	// disambiguate the fallback case.
	if rt.Default != "" {
		if _, ok := rt.Variants[rt.Default]; !ok {
			errs = append(errs, fmt.Errorf("manifest: default %q does not match any declared variant", rt.Default))
		}
	} else if len(rt.Variants) > 1 {
		errs = append(errs, errors.New("manifest: default is required when more than one variant is declared"))
	}

	for variantKey, slice := range rt.Variants {
		switch variantKey {
		case variantDark, variantLight, variantNoTTY:
		default:
			errs = append(errs, fmt.Errorf("manifest: variant key %q invalid (want dark|light|notty)", variantKey))
			continue
		}
		errs = append(errs, validateVariant(rt.Name, string(variantKey), slice)...)
	}

	return errors.Join(errs...)
}

// validateVariant runs the per-variant structural checks. Returns a
// slice (possibly empty) so the caller can join with the top-level
// errors. It does not return Variant-prefixed messages; the caller is
// expected to wrap the eventual error.
func validateVariant(manifestName, variantName string, slice rawSlice) []error {
	var errs []error
	prefix := func(e error) error {
		return fmt.Errorf("variant %q: %w", variantName, e)
	}

	if len(slice.Primitives) == 0 {
		errs = append(errs, prefix(errors.New("primitives must contain at least one entry")))
	}
	if len(slice.Tokens) == 0 {
		errs = append(errs, prefix(errors.New("tokens must contain at least one entry")))
	}
	if slice.Chroma != "" {
		if _, ok := chromastyles.Registry[slice.Chroma]; !ok {
			errs = append(errs, prefix(fmt.Errorf(
				"chroma %q is not a registered chroma style; available: %v",
				slice.Chroma, chromastyles.Names())))
		}
	}
	if slice.Glamour != "" {
		if _, ok := glamourstyles.DefaultStyles[slice.Glamour]; !ok {
			errs = append(errs, prefix(fmt.Errorf(
				"glamour %q is not a known glamour preset; available: %v",
				slice.Glamour, glamourPresetNames())))
		}
	}
	if slice.Huh != "" {
		if _, ok := lookupPromptAdapter(slice.Huh); !ok {
			errs = append(errs, prefix(errUnknownPromptAdapter(manifestName, variantName, slice.Huh)))
		}
	}
	return errs
}

func parsePromptKnobs(raw *rawPromptKnobs) (*PromptKnobs, error) {
	if raw == nil {
		return nil, errNilPromptKnobs
	}
	out := &PromptKnobs{
		SelectedPrefix:   raw.SelectedPrefix,
		UnselectedPrefix: raw.UnselectedPrefix,
	}
	if raw.Border != "" {
		b, err := resolveBorderRef(raw.Border)
		if err != nil {
			return nil, err
		}
		out.Border = b
	}
	return out, nil
}

// glamourPresetNames returns the sorted set of upstream glamour preset
// names. It is used to build the "did you mean" tail of validation
// errors for the "glamour" field; sorted output keeps the message
// stable across Go map iteration order.
func glamourPresetNames() []string {
	out := make([]string, 0, len(glamourstyles.DefaultStyles))
	for k := range glamourstyles.DefaultStyles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
