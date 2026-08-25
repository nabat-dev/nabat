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
	"image/color"
	"sort"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	glamourstyles "charm.land/glamour/v2/styles"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
)

var errNilPromptKnobs = errors.New("prompt knobs input is nil")

// Compiled is the intermediate [Parse] result. The catalog assembles a
// theme.Theme from it; this type never references the theme package.
// Variants are keyed by mode ("dark" / "light" / "notty").
type Compiled struct {
	// Name is the manifest "name" field and catalog registry key.
	Name string

	// Default is the fallback variant when capabilities do not match.
	// Empty for single-variant themes.
	Default string

	// Variants holds one [CompiledVariant] per declared mode.
	Variants map[string]*CompiledVariant

	// PromptKnobs are theme-wide prompt settings, or nil when omitted.
	PromptKnobs *PromptKnobs
}

// CompiledVariant is per-variant style data from one "variants" entry.
// Token names are plain strings; the catalog wraps them as theme.Token.
type CompiledVariant struct {
	// Tokens is the resolved per-token style map. $primitive / $token
	// / inline refs are already collapsed into lipgloss styles.
	Tokens map[string]lipgloss.Style

	// Aliases are per-variant alias overrides. An empty value disables
	// the framework default for that key (see theme.Palette.Aliases).
	Aliases map[string]string

	// ChromaName is the upstream chroma style from the "chroma" field.
	ChromaName string

	// GlamourName is the upstream glamour preset from the "glamour" field.
	GlamourName string

	// HuhTheme is the [huh.Theme] from the per-variant "huh" adapter.
	HuhTheme huh.Theme
}

// PromptKnobs stores theme-wide prompt controls.
type PromptKnobs struct {
	SelectedPrefix   string
	UnselectedPrefix string
	Border           lipgloss.Border
	BorderColor      color.Color
}

// Parse decodes a manifest into a [*Compiled] intermediate. Errors from
// JSON shape, cross-field validation, or $token / primitive resolution
// surface here. Runtime glamour failures are unrelated and surface later
// through theme.Theme.ResolveErr.
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

// parseVariant compiles one "variants" map entry.
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

// validateManifest runs structural and cross-field checks beyond
// DisallowUnknownFields / JSON Schema. It joins every problem so one
// parse surfaces the full diagnosis.
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

// validateVariant runs per-variant structural checks. Returns a slice
// for the caller to join; messages are not variant-prefixed.
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
				slice.Chroma, chromastyles.Names(),
			)))
		}
	}
	if slice.Glamour != "" {
		if _, ok := glamourstyles.DefaultStyles[slice.Glamour]; !ok {
			errs = append(errs, prefix(fmt.Errorf(
				"glamour %q is not a known glamour preset; available: %v",
				slice.Glamour, glamourPresetNames(),
			)))
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
	if !raw.BorderColor.isEmpty() {
		if raw.BorderColor.Hex == "" {
			return nil, fmt.Errorf("promptKnobs.borderColor must be a hex literal")
		}
		out.BorderColor = lipgloss.Color(raw.BorderColor.Hex)
	}
	return out, nil
}

// glamourPresetNames returns sorted upstream glamour preset names for
// validation error tails.
func glamourPresetNames() []string {
	out := make([]string, 0, len(glamourstyles.DefaultStyles))
	for k := range glamourstyles.DefaultStyles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
