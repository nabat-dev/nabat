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

// Untyped string constants for the well-known theme names shipped with
// Nabat. Use them as the argument to nabat.WithTheme to get IDE
// autocomplete and a compile-time check that the spelling matches an
// embedded manifest:
//
//	app, _ := nabat.New("myctl", nabat.WithTheme(theme.Dracula))
//
// These constants are untyped on purpose so they compose with strings
// from other sources (env vars, flags, user config) without explicit
// conversions:
//
//	name := os.Getenv("MYCTL_THEME")
//	if name == "" {
//	    name = theme.Default
//	}
//	app, _ := nabat.New("myctl", nabat.WithTheme(name))
//
// Every constant here corresponds to an embedded data/<name>.json file;
// TestConstsHaveManifests in catalog_test.go enforces that the two stay
// in lockstep so a misspelled or missing manifest fails the build, not
// nabat.New at runtime.
const (
	// Default is the theme installed when the caller does not pass
	// nabat.WithTheme. Capability-aware: defers to the terminal's
	// detected color profile and background luminance.
	Default = "default"

	// Minimal is a low-color theme that relies on bold instead of
	// foreground colors. It declares variant=notty so the framework
	// disables chroma syntax highlighting and forces glamour into
	// plain-text mode, matching pipe-friendly defaults.
	Minimal = "minimal"

	// Charm is the higher-contrast palette aligned with Charm.land
	// defaults, suitable for dark terminals.
	Charm = "charm"

	// Dracula ships Dracula Classic (dark) and Alucard Classic (light)
	// per draculatheme.com/spec; pickVariant selects by terminal
	// luminance. Dark uses chroma/glamour/huh "dracula"; light uses
	// chroma "github", glamour "light", and token-derived prompts.
	Dracula = "dracula"

	// Gruvbox is the retro groove palette from morhetz/gruvbox; dark and
	// light variants switch via pickVariant. Pairs with chroma's "gruvbox"
	// and "gruvbox-light" styles.
	Gruvbox = "gruvbox"

	// CatppuccinLatte is the Catppuccin Latte palette for light
	// backgrounds. Pairs with chroma's "catppuccin-latte" style.
	CatppuccinLatte = "catppuccin-latte"

	// CatppuccinFrappe is the Catppuccin Frappé palette for dark
	// backgrounds. Pairs with chroma's "catppuccin-frappe" style.
	CatppuccinFrappe = "catppuccin-frappe"

	// CatppuccinMacchiato is the Catppuccin Macchiato palette for dark
	// backgrounds. Pairs with chroma's "catppuccin-macchiato" style.
	CatppuccinMacchiato = "catppuccin-macchiato"

	// CatppuccinMocha is the Catppuccin Mocha palette for dark
	// backgrounds. Pairs with chroma's "catppuccin-mocha" style.
	CatppuccinMocha = "catppuccin-mocha"

	// Nabat is the brand palette: warm Persian rock-candy tones
	// (saffron, pistachio, pomegranate, turquoise). It ships with a
	// matching framework-owned chroma style and huh adapter, both
	// referenced by name from the manifest.
	Nabat = "nabat"

	// Nord is the Nord palette (Polar Night, Snow Storm, Frost,
	// Aurora) for dark backgrounds. Pairs with chroma's "nord" style.
	Nord = "nord"

	// Solarized is the canonical Solarized palette (base tones plus eight
	// accents). Dark/light variants follow https://ethanschoonover.com/solarized/;
	// pickVariant selects by terminal luminance. Pairs with chroma's
	// "solarized-dark" / "solarized-light" styles.
	Solarized = "solarized"
)
