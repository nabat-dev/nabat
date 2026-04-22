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
	"image/color"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveColorRefHexLiteral verifies the bare-string form of the
// rawColorRef union resolves directly to a lipgloss color matching the
// hex literal.
func TestResolveColorRefHexLiteral(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(map[string]string{}, map[string]rawStyle{})
	got, err := r.resolveColorRef(rawColorRef{Hex: "#E05454"}, "fg")
	require.NoError(t, err)
	assert.Truef(t, sameColor(got, lipgloss.Color("#E05454")),
		"resolveColorRef hex: got %v, want #E05454", got)
}

// TestResolveColorRefPrimitive verifies a $primitive object resolves to
// the hex color the manifest's primitives map declares for that name.
func TestResolveColorRefPrimitive(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(
		map[string]string{"warmCream": "#EDE4D3"},
		map[string]rawStyle{},
	)
	got, err := r.resolveColorRef(rawColorRef{Primitive: "warmCream"}, "fg")
	require.NoError(t, err)
	assert.Truef(t, sameColor(got, lipgloss.Color("#EDE4D3")),
		"resolveColorRef primitive: got %v, want #EDE4D3", got)
}

// TestResolveColorRefToken verifies a $token object reads back the
// foreground of the referenced token.
func TestResolveColorRefToken(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(
		map[string]string{"saffronGold": "#D4A853"},
		map[string]rawStyle{
			"text.title": {Primitive: "saffronGold"},
		},
	)
	got, err := r.resolveColorRef(rawColorRef{Token: "text.title"}, "fg")
	require.NoError(t, err)
	assert.Truef(t, sameColor(got, lipgloss.Color("#D4A853")),
		"resolveColorRef token: got %v, want #D4A853", got)
}

// TestResolveColorRefUnknownPrimitive surfaces an actionable error.
func TestResolveColorRefUnknownPrimitive(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(map[string]string{}, map[string]rawStyle{})
	_, err := r.resolveColorRef(rawColorRef{Primitive: "missing"}, "fg")
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing")
	assert.ErrorContains(t, err, "fg")
}

// TestResolveColorRefMultipleSet rejects manifests that bypass the
// schema's oneOf and set more than one form on a single ref.
func TestResolveColorRefMultipleSet(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(map[string]string{"x": "#FFFFFF"}, map[string]rawStyle{})
	_, err := r.resolveColorRef(rawColorRef{Hex: "#000000", Primitive: "x"}, "fg")
	require.Error(t, err)
}

// TestResolveSpecBorderPreset verifies the border enum lands on the
// resulting lipgloss style as the matching preset.
func TestResolveSpecBorderPreset(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(map[string]string{}, map[string]rawStyle{})
	got, err := r.resolveSpec(rawStyle{Border: "rounded"})
	require.NoError(t, err)
	want := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder())
	assert.Equal(t, want.GetBorderStyle(), got.GetBorderStyle(),
		"border style mismatch")
}

// TestResolveSpecBorderUnknown rejects unknown border preset names.
func TestResolveSpecBorderUnknown(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(map[string]string{}, map[string]rawStyle{})
	_, err := r.resolveSpec(rawStyle{Border: "ornate"})
	require.Error(t, err)
}

// TestResolveSpecColorRefInFg exercises the integration: a fg field that
// uses a $primitive object resolves end-to-end through resolveSpec.
func TestResolveSpecColorRefInFg(t *testing.T) {
	t.Parallel()

	r := newStyleResolver(
		map[string]string{"turquoise": "#3EB0CC"},
		map[string]rawStyle{},
	)
	got, err := r.resolveSpec(rawStyle{
		Fg: rawColorRef{Primitive: "turquoise"},
	})
	require.NoError(t, err)
	assert.Truef(t, sameColor(got.GetForeground(), lipgloss.Color("#3EB0CC")),
		"fg primitive ref did not propagate: got %v", got.GetForeground())
}

// sameColor compares two [color.Color] values by their RGBA components.
// [lipgloss.Color] values produced from equal hex strings compare equal
// this way without depending on internal representation. It is shared
// across test files in this package so any color comparison uses the
// same equality definition.
func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}
