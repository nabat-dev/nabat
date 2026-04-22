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
	"fmt"

	"charm.land/lipgloss/v2"

	"nabat.dev/theme"
)

// ExampleTheme demonstrates the canonical declarative shape: a Theme
// is data — one Palette per declared variant, plus cross-variant
// defaults. Resolution picks a variant based on Capabilities and
// applies framework defaults for any nil/empty cascade slot.
func ExampleTheme() {
	t := theme.Theme{
		Name:    "plain",
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusSuccess: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7EC87E")),
				},
			},
		},
	}

	r := t.Resolve(theme.Capabilities{Dark: true, Interactive: true})
	fmt.Println(r.Name())
	fmt.Println(r.Style(theme.StatusSuccess).GetBold())
	// Output:
	// plain
	// true
}

// ExampleResolver shows the escape hatch for themes that need to pick
// a palette based on runtime Capabilities in a way one Palette per
// Variant cannot express. Most themes never need this — the built-in
// catalog and every straight programmatic theme satisfies Resolver via
// the [Theme.Resolve] method that comes with the struct.
func ExampleResolver() {
	r := capabilityAwareResolver{}
	rt := r.Resolve(theme.Capabilities{Dark: false, Interactive: true})
	fmt.Println(rt.Style(theme.StatusError).GetBold())
	// Output: true
}

// capabilityAwareResolver picks a palette by examining capabilities at
// resolve time, branching on the Profile field in a way one Palette
// per Variant cannot express directly.
type capabilityAwareResolver struct{}

func (capabilityAwareResolver) Resolve(c theme.Capabilities) theme.ResolvedTheme {
	fg := lipgloss.Color("#E05454")
	if !c.Dark {
		fg = lipgloss.Color("#A03030")
	}
	t := theme.Theme{
		Name: "capability-aware",
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantUnset: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusError: lipgloss.NewStyle().Foreground(fg).Bold(true),
				},
			},
		},
	}
	return t.Resolve(c)
}
