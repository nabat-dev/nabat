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
	"image/color"

	"charm.land/lipgloss/v2"
)

// styleResolver turns rawStyle entries into [lipgloss.Style] values.
// It memoizes tokens and tracks an in-flight set so cycles error instead
// of overflowing the stack. One-shot per manifest; do not reuse across
// manifests (primitive maps would crosslink).
type styleResolver struct {
	primitives map[string]string
	rawTokens  map[string]rawStyle
	cache      map[string]lipgloss.Style
	visiting   map[string]bool
}

// newStyleResolver returns a resolver bound to one manifest's primitive
// and token maps.
func newStyleResolver(primitives map[string]string, rawTokens map[string]rawStyle) *styleResolver {
	return &styleResolver{
		primitives: primitives,
		rawTokens:  rawTokens,
		cache:      map[string]lipgloss.Style{},
		visiting:   map[string]bool{},
	}
}

// resolveToken returns the [lipgloss.Style] for token name, building
// referenced tokens on demand. Results are memoized. It fails on an unknown
// token name or a cycle via $token.
func (r *styleResolver) resolveToken(name string) (lipgloss.Style, error) {
	if s, ok := r.cache[name]; ok {
		return s, nil
	}
	if r.visiting[name] {
		return lipgloss.Style{}, fmt.Errorf("token %q: cycle detected via $token reference", name)
	}
	spec, ok := r.rawTokens[name]
	if !ok {
		return lipgloss.Style{}, fmt.Errorf("$token reference to unknown token %q", name)
	}

	r.visiting[name] = true
	style, err := r.resolveSpec(spec)
	delete(r.visiting, name)
	if err != nil {
		return lipgloss.Style{}, fmt.Errorf("token %q: %w", name, err)
	}
	r.cache[name] = style
	return style, nil
}

// resolveSpec turns one rawStyle into a [lipgloss.Style] without caching.
// Order: seed from $primitive or $token; apply fg/bg/border colors;
// apply border preset; apply boolean modifiers (pointers can unset
// inherited ones); apply SetString literal text.
func (r *styleResolver) resolveSpec(spec rawStyle) (lipgloss.Style, error) {
	var s lipgloss.Style
	switch {
	case spec.Token != "":
		base, err := r.resolveToken(spec.Token)
		if err != nil {
			return lipgloss.Style{}, err
		}
		s = base
	case spec.Primitive != "":
		hex, ok := r.primitives[spec.Primitive]
		if !ok {
			return lipgloss.Style{}, fmt.Errorf("$primitive reference to unknown primitive %q", spec.Primitive)
		}
		s = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
	}

	if !spec.Fg.isEmpty() {
		c, err := r.resolveColorRef(spec.Fg, "fg")
		if err != nil {
			return lipgloss.Style{}, err
		}
		s = s.Foreground(c)
	}
	if !spec.Bg.isEmpty() {
		c, err := r.resolveColorRef(spec.Bg, "bg")
		if err != nil {
			return lipgloss.Style{}, err
		}
		s = s.Background(c)
	}
	if !spec.BorderForeground.isEmpty() {
		c, err := r.resolveColorRef(spec.BorderForeground, "borderForeground")
		if err != nil {
			return lipgloss.Style{}, err
		}
		s = s.BorderForeground(c)
	}
	if !spec.BorderBackground.isEmpty() {
		c, err := r.resolveColorRef(spec.BorderBackground, "borderBackground")
		if err != nil {
			return lipgloss.Style{}, err
		}
		s = s.BorderBackground(c)
	}
	if spec.Border != "" {
		border, err := resolveBorderRef(spec.Border)
		if err != nil {
			return lipgloss.Style{}, err
		}
		s = s.BorderStyle(border)
	}
	if spec.Bold != nil {
		s = s.Bold(*spec.Bold)
	}
	if spec.Italic != nil {
		s = s.Italic(*spec.Italic)
	}
	if spec.Underline != nil {
		s = s.Underline(*spec.Underline)
	}
	if spec.Strikethrough != nil {
		s = s.Strikethrough(*spec.Strikethrough)
	}
	if spec.Faint != nil {
		s = s.Faint(*spec.Faint)
	}
	if spec.Blink != nil {
		s = s.Blink(*spec.Blink)
	}
	if spec.Reverse != nil {
		s = s.Reverse(*spec.Reverse)
	}
	if spec.Text != "" {
		s = s.SetString(spec.Text)
	}
	return s, nil
}

// resolveColorRef converts a [rawColorRef] into a [color.Color]
// (lipgloss's color values are this standard-library interface). The
// field argument is the JSON key the ref appeared under (fg/bg/...);
// it is folded into the error message so authors can locate the bad
// reference at a glance.
//
// Multiple-set forms (the schema's oneOf is bypassed somehow) surface
// as an error so the bug is loud rather than silently picking one form
// over another.
func (r *styleResolver) resolveColorRef(ref rawColorRef, field string) (color.Color, error) {
	set := 0
	if ref.Hex != "" {
		set++
	}
	if ref.Primitive != "" {
		set++
	}
	if ref.Token != "" {
		set++
	}
	if set != 1 {
		return nil, fmt.Errorf("%s: color reference must set exactly one of hex / $primitive / $token, got %d", field, set)
	}

	switch {
	case ref.Hex != "":
		return lipgloss.Color(ref.Hex), nil
	case ref.Primitive != "":
		hex, ok := r.primitives[ref.Primitive]
		if !ok {
			return nil, fmt.Errorf("%s: $primitive reference to unknown primitive %q", field, ref.Primitive)
		}
		return lipgloss.Color(hex), nil
	case ref.Token != "":
		s, err := r.resolveToken(ref.Token)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", field, err)
		}
		fg := s.GetForeground()
		if _, isNoColor := fg.(lipgloss.NoColor); isNoColor {
			return nil, fmt.Errorf("%s: $token %q has no foreground color to reference", field, ref.Token)
		}
		return fg, nil
	}
	// Unreachable: the set != 1 guard above ensures exactly one ref form
	// matches the switch. A panic here turns any future invariant break
	// (a new rawColorRef field added without updating the switch) into a
	// loud failure instead of a silent (nil, nil) leak.
	panic(fmt.Sprintf("manifest: resolveColorRef reached unreachable code for field %q", field))
}

// resolveBorderRef maps a [rawBorderRef] onto the corresponding
// lipgloss border preset. Unknown values surface as an error; the schema
// already constrains the enum, but the parser duplicates the check
// defensively so manifests bypassing the schema validator (or future
// schema changes) cannot silently produce an empty border.
func resolveBorderRef(ref rawBorderRef) (lipgloss.Border, error) {
	switch ref {
	case "hidden":
		return lipgloss.HiddenBorder(), nil
	case "normal":
		return lipgloss.NormalBorder(), nil
	case "rounded":
		return lipgloss.RoundedBorder(), nil
	case "thick":
		return lipgloss.ThickBorder(), nil
	case "double":
		return lipgloss.DoubleBorder(), nil
	case "block":
		return lipgloss.BlockBorder(), nil
	case "outerHalfBlock":
		return lipgloss.OuterHalfBlockBorder(), nil
	case "innerHalfBlock":
		return lipgloss.InnerHalfBlockBorder(), nil
	}
	return lipgloss.Border{}, fmt.Errorf("unknown border preset %q", string(ref))
}
