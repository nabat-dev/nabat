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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validVariant returns a minimal but well-formed variant slice the
// tests below use as a base — primitives + tokens both populated so
// the per-variant required-field checks pass without ceremony.
func validVariant() rawSlice {
	return rawSlice{
		Primitives: map[string]string{"a": "#000000"},
		Tokens:     map[string]rawStyle{"text.primary": {Primitive: "a"}},
	}
}

// TestValidateManifestRejectsUnknownChromaName guards the parse-time
// check that unknown upstream chroma style names fail loudly. Without
// it an author who typos "monokaii" would silently get unstyled output.
func TestValidateManifestRejectsUnknownChromaName(t *testing.T) {
	t.Parallel()

	v := validVariant()
	v.Chroma = "definitely-not-a-chroma-style"
	rt := &rawTheme{
		Name:     "x",
		Variants: map[rawVariant]rawSlice{variantDark: v},
	}
	err := validateManifest(rt)
	require.Error(t, err)
	assert.ErrorContains(t, err, "definitely-not-a-chroma-style")
	assert.ErrorContains(t, err, "available:")
}

// TestValidateManifestRejectsUnknownGlamourName mirrors the chroma
// check for the "glamour" field. Both upstream packages ship a finite
// preset list, so unknown names should fail at registry init.
func TestValidateManifestRejectsUnknownGlamourName(t *testing.T) {
	t.Parallel()

	v := validVariant()
	v.Glamour = "definitely-not-a-glamour-preset"
	rt := &rawTheme{
		Name:     "x",
		Variants: map[rawVariant]rawSlice{variantDark: v},
	}
	err := validateManifest(rt)
	require.Error(t, err)
	assert.ErrorContains(t, err, "definitely-not-a-glamour-preset")
	assert.ErrorContains(t, err, "available:")
}

// TestValidateManifestAcceptsKnownChromaAndGlamourNames documents the
// happy path for the catalog-membership checks: known names parse
// without error so the catalog defaults stay reachable.
func TestValidateManifestAcceptsKnownChromaAndGlamourNames(t *testing.T) {
	t.Parallel()

	v := validVariant()
	v.Chroma = "monokai"
	v.Glamour = "dark"
	rt := &rawTheme{
		Name:     "x",
		Variants: map[rawVariant]rawSlice{variantDark: v},
	}
	assert.NoError(t, validateManifest(rt),
		"known chroma/glamour names must validate cleanly")
}

func TestValidateManifestRejectsUnknownHuhAdapter(t *testing.T) {
	t.Parallel()

	v := validVariant()
	v.Huh = "definitely-not-an-adapter"
	rt := &rawTheme{
		Name:     "x",
		Variants: map[rawVariant]rawSlice{variantDark: v},
	}
	err := validateManifest(rt)
	require.Error(t, err)
	assert.ErrorContains(t, err, "manifest \"x\"")
	assert.ErrorContains(t, err, "variant \"dark\"")
	assert.ErrorContains(t, err, "\"huh\"")
	assert.ErrorContains(t, err, "supported adapters:")
}

// TestValidateManifestRejectsBadVariantKey covers the variant-enum
// check: a key outside dark/light/notty surfaces a typed error so a
// typo in the variants map fails at registry load instead of running
// time.
func TestValidateManifestRejectsBadVariantKey(t *testing.T) {
	t.Parallel()

	rt := &rawTheme{
		Name:     "x",
		Variants: map[rawVariant]rawSlice{"daark": validVariant()},
	}
	err := validateManifest(rt)
	require.Error(t, err)
	assert.ErrorContains(t, err, "daark")
	assert.ErrorContains(t, err, "dark|light|notty")
}

// TestValidateManifestRequiresDefaultWhenMultiVariant locks the rule
// that a multi-variant manifest must declare which variant
// [theme.Theme.Resolve] should fall back to when capabilities don't
// pin a clear pick.
func TestValidateManifestRequiresDefaultWhenMultiVariant(t *testing.T) {
	t.Parallel()

	rt := &rawTheme{
		Name: "x",
		Variants: map[rawVariant]rawSlice{
			variantDark:  validVariant(),
			variantLight: validVariant(),
		},
	}
	err := validateManifest(rt)
	require.Error(t, err)
	assert.ErrorContains(t, err, "default is required")
}

// TestValidateManifestRejectsBadDefault catches mistyped or otherwise
// non-existent default targets.
func TestValidateManifestRejectsBadDefault(t *testing.T) {
	t.Parallel()

	rt := &rawTheme{
		Name:     "x",
		Default:  variantLight,
		Variants: map[rawVariant]rawSlice{variantDark: validVariant()},
	}
	err := validateManifest(rt)
	require.Error(t, err)
	assert.ErrorContains(t, err, "default \"light\"")
	assert.ErrorContains(t, err, "does not match any declared variant")
}

func TestParsePromptKnobsRejectsNilInput(t *testing.T) {
	t.Parallel()

	knobs, err := parsePromptKnobs(nil)
	require.ErrorIs(t, err, errNilPromptKnobs)
	assert.Nil(t, knobs)
}

func TestParseRejectsTrailingJSONData(t *testing.T) {
	t.Parallel()

	const data = `{
		"name":"x",
		"variants":{
			"dark":{
				"primitives":{"a":"#000000"},
				"tokens":{"text.primary":{"$primitive":"a"}}
			}
		}
	}{"extra":true}`

	_, err := Parse([]byte(data))
	require.Error(t, err)
	assert.ErrorContains(t, err, "trailing data after top-level object")
}
