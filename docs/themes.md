# Themes

How Nabat's theme system fits together: the merged leaf+catalog
package, the data-driven `Theme` value, the multi-variant manifest
format, and the per-render-target style blocks (chroma, glamour,
prompt).

## Contents

- [System Overview](#system-overview)
- [Selecting a Theme](#selecting-a-theme)
- [Programmatic Themes](#programmatic-themes)
- [Theme Overrides](#theme-overrides)
- [Manifest Format](#manifest-format)
- [Token Catalog](#token-catalog)
- [Token Aliases](#token-aliases)
- [Optional Fields and Defaults](#optional-fields-and-defaults)
- [Custom Style Authoring Paths](#custom-style-authoring-paths)
- [JSON Schema](#json-schema)
- [Schema Hosting](#schema-hosting)
- [Schema Versioning](#schema-versioning)
- [Adding a Theme](#adding-a-theme)
- [Capabilities](#capabilities)
- [Token Requirements](#token-requirements)

## System Overview

Nabat ships theming through a single package with a strict
no-imports-from-the-root-package invariant:

```text
nabat ──► nabat/theme  (leaf + catalog)
                  └── theme/internal/manifest  (parser, no theme imports)
```

| Package        | Role                                                                                                       |
|----------------|------------------------------------------------------------------------------------------------------------|
| `nabat.dev/theme` | **Leaf primitives + built-in catalog.** `Theme` (data), `Token`, `Capabilities`, `Variant`, `ResolvedTheme`, `Palette`, `Prompt`, `Recipe` interface; embedded DTCG JSON manifests under `data/`, JSON Schema under `schema/`, lazy registry (`Get` / `Names` / `All` / `Schema` / `Manifest`), untyped string constants for every shipped name, and a closed catalog of bundled upstream `huh.Theme` wrappers (`charm`, `base16`, `dracula`, `catppuccin`). No imports from `nabat.dev`. |
| `nabat.dev`    | **Wiring.** `nabat.WithTheme(name)` looks up the registry, `nabat.WithCustomTheme(theme.Recipe)` accepts any `Recipe` value (including a plain `theme.Theme`). `App.finalize` detects `Capabilities`, calls `Theme.Resolve(caps)`, and pins the resulting `theme.ResolvedTheme` for the lifetime of the app. `App.Theme()` returns it. |

`theme.Theme` is **declarative data**, not a function. The struct
carries one or more `Palette` entries (one per declared variant), the
default variant pick when capabilities don't pin one, and a few
cross-variant defaults. Resolution picks a variant, applies framework
defaults to any nil/empty cascade slot, and returns an immutable
`ResolvedTheme`.

The recipe runs **once per `App`**, on construction. After
`nabat.New(...)` returns, the `ResolvedTheme` is immutable and
thread-safe; every output path (`Success`, `Warn`, `Table`, `List`,
`renderHelp`, `Markdown`, the logging extension) reads through
`App.Theme()`.

## Selecting a Theme

Pass the theme name to `nabat.WithTheme`:

```go
import (
    "nabat.dev"
    "nabat.dev/theme"
)

app, _ := nabat.New("myctl", nabat.WithTheme(theme.Dracula))
```

The constants in `nabat.dev/theme` are untyped strings so you can mix
them freely with strings from other sources:

```go
name := os.Getenv("MYCTL_THEME")
if name == "" {
    name = theme.Default
}
app, _ := nabat.New("myctl", nabat.WithTheme(name))
```

Unknown names produce a `*ConfigErrors` from `nabat.New` listing every
registered name, so the user immediately sees what they meant to type.

The catalog shipped with Nabat:

| Constant                  | Manifest                       | Best for                                                                                                                   |
|---------------------------|--------------------------------|----------------------------------------------------------------------------------------------------------------------------|
| `theme.Default`           | `data/default.json`            | Capability-aware default; defers to the terminal's detected color profile and background luminance.                        |
| `theme.Minimal`           | `data/minimal.json`            | Bold-only, no foreground colors. `notty` variant disables chroma and forces glamour into plain-text mode.                  |
| `theme.Charm`             | `data/charm.json`              | Higher-contrast Charm.land palette for dark terminals.                                                                     |
| `theme.Dracula`           | `data/dracula.json`            | Dracula palette for dark backgrounds; pairs with the upstream chroma `dracula` style and glamour `dracula` preset.         |
| `theme.CatppuccinMocha`   | `data/catppuccin-mocha.json`   | Catppuccin Mocha for dark backgrounds; pairs with chroma's `catppuccin-mocha`.                                             |
| `theme.Nabat`             | `data/nabat.json`              | Brand palette: warm Persian rock-candy tones with a framework-shipped chroma style and prompt block.                       |

## Programmatic Themes

When the styling you want cannot be expressed as a JSON manifest —
for example a `huh.Theme` closure or a `chroma.Style` you want to
own in Go — build a `theme.Theme` struct and install it with
`nabat.WithCustomTheme`:

```go
import (
    "charm.land/lipgloss/v2"

    "nabat.dev"
    "nabat.dev/theme"
)

acme := theme.Theme{
    Name:    "acme",
    Default: theme.VariantDark,
    Variants: map[theme.Variant]theme.Palette{
        theme.VariantDark: {
            Tokens: map[theme.Token]lipgloss.Style{
                theme.StatusError: lipgloss.NewStyle().Foreground(lipgloss.Color("#E05454")).Bold(true),
                theme.TextLabel:   lipgloss.NewStyle().Foreground(lipgloss.Color("#C89B3C")).Bold(true),
                theme.TextValue:   lipgloss.NewStyle().Foreground(lipgloss.Color("#EDE4D3")),
            },
            Chroma:  acmeChroma,        // owned *chroma.Style
            Glamour: acmeGlamourCfg,    // owned *ansi.StyleConfig
            Huh:     acmeHuhTheme,      // owned huh.Theme escape hatch
        },
    },
}

app, _ := nabat.New("myctl", nabat.WithCustomTheme(acme))
```

Notes:

- `Theme.Name` is recorded on the resolved theme so error messages
  identify which theme produced an invalid configuration.
- `WithTheme(name)` and `WithCustomTheme(t)` **compose**: the last one
  wins. There is no mutual-exclusion check.
- For palette choices that depend on runtime `Capabilities` in a way a
  per-variant `Palette` cannot express, implement the
  `theme.Recipe` interface directly:

  ```go
  type myRecipe struct{}
  func (myRecipe) Resolve(c theme.Capabilities) theme.ResolvedTheme {
      // pick a Palette / Theme based on c.Profile, c.BackgroundHex, ...
      // and return its Resolve(c) result.
  }
  ```

## Theme Overrides

To tweak a single slot of a built-in theme, reach for
`nabat.WithThemeOverride` instead of constructing a derived theme:

```go
import "charm.land/lipgloss/v2"

app, _ := nabat.New("myctl",
    nabat.WithTheme(theme.Dracula),
    nabat.WithThemeOverride(theme.StatusError,
        lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true),
    ),
)
```

For batch overrides (multiple slots, alias overrides, chroma swaps),
use `WithThemeOverrides` with the helpers from `theme/override.go`:

```go
app, _ := nabat.New("myctl",
    nabat.WithTheme(theme.Dracula),
    nabat.WithThemeOverrides(
        theme.SetToken(theme.StatusError, magenta),
        theme.SetAlias(theme.ListItem, theme.TextBody),
        theme.SetChromaName("monokai"),
    ),
)
```

Overrides apply to **every variant** of the underlying theme, so a
multi-variant manifest stays multi-variant after the override; the
same one-line tweak affects whichever variant the runtime
capabilities pick.

Overrides are silently ignored when the active theme is a bespoke
`theme.Recipe` (anything other than a `theme.Theme` value); the
recipe's `Resolve` method is opaque, so the framework cannot
meaningfully apply per-`Palette` overrides into it.

## Manifest Format

A theme manifest is a JSON document describing one Nabat theme.
Manifests follow the
[Design Tokens Community Group (DTCG)](https://design-tokens.github.io/community-group/format/)
three-tier model, organized per **variant**:

1. **Variants** — top-level map keyed by mode (`dark` / `light` /
   `notty`). Each entry is a self-contained palette.
2. **Primitives** — per-variant named raw colors (hex literals).
3. **Tokens** — per-variant named semantic styles that reference
   primitives or other tokens.
4. **Component-level styles** — per-variant optional fields that
   point at the chroma, glamour, and prompt integrations either by
   upstream name or inline definition.

Minimal example (single-variant):

```json
{
  "$schema": "https://nabat.dev/schemas/theme/v1.json",
  "name": "dracula",
  "variants": {
    "dark": {
      "primitives": {
        "green":   "#50FA7B",
        "red":     "#FF5555",
        "comment": "#6272A4"
      },
      "tokens": {
        "status.success": { "$primitive": "green", "bold": true },
        "status.error":   { "$primitive": "red",   "bold": true },
        "text.muted":     { "$primitive": "comment" }
      }
    }
  }
}
```

Multi-variant example:

```json
{
  "$schema": "https://nabat.dev/schemas/theme/v1.json",
  "name": "myapp",
  "default": "dark",
  "variants": {
    "dark":  { "primitives": { "fg": "#FFFFFF", "bg": "#000000" }, "tokens": { ... } },
    "light": { "primitives": { "fg": "#000000", "bg": "#FFFFFF" }, "tokens": { ... } }
  },
  "prompt": "charm"
}
```

The `$schema` field points at the public JSON Schema URL so editors
that support JSON Schema (VSCode, JetBrains, Zed, Neovim with
`coc-json`) get live validation, autocomplete, and hover documentation.

### Required Fields

| Field        | Type   | Purpose                                                                                                                |
|--------------|--------|------------------------------------------------------------------------------------------------------------------------|
| `name`       | string | Identifier; matches `^[a-z0-9][a-z0-9-]*$`. Must equal the manifest's filename (without `.json`).                      |
| `variants`   | object | Keyed by mode (`dark` / `light` / `notty`). Each entry has its own `primitives` + `tokens`.                            |
| `default`    | enum   | `dark` / `light` / `notty`. Required when more than one variant is declared; optional for single-variant themes.       |

Per-variant required fields:

| Field        | Type   | Purpose                                                                                                                |
|--------------|--------|------------------------------------------------------------------------------------------------------------------------|
| `primitives` | object | Map of name to hex color (`^#[0-9A-Fa-f]{6}$`). Tokens reference these via `$primitive`.                               |
| `tokens`     | object | Map of token name (for example `status.success`) to a style spec.                                                      |

### Style Spec

A style spec is an object with at least one field:

| Field             | Type    | Purpose                                                                                                                            |
|-------------------|---------|------------------------------------------------------------------------------------------------------------------------------------|
| `$primitive`      | string  | Reference to a key in this variant's `primitives` map. Sets the foreground color. Mutually exclusive with `$token`.                |
| `$token`          | string  | Reference to a key in this variant's `tokens` map. Inherits that token's resolved style; other fields layer on top.                |
| `fg` / `bg`       | hex / colorRef | Explicit foreground / background colors.                                                                                    |
| `borderForeground` / `borderBackground` | hex / colorRef | Border colors.                                                                                            |
| `bold`, `italic`, `underline`, `strikethrough`, `faint`, `blink`, `reverse` | bool | Lipgloss attribute toggles.                                                          |
| `text`            | string  | Literal text content. Used by prompt prefix slots (e.g. `"check "`).                                                               |

`$primitive` and `$token` are mutually exclusive within a single spec.
Unknown attributes are rejected at parse time (the JSON Schema sets
`additionalProperties: false` on the style spec).

## Token Catalog

`nabat.dev/theme` defines the well-known tokens used by core consumers.
Token names are an open set — third-party themes and extensions may add
their own and read them back via `ResolvedTheme.Style(token)`. A theme
that omits a token falls through the **alias chain** before hitting
the zero `lipgloss.Style` (terminal default).

| Token              | Consumer                                                                                                      |
|--------------------|---------------------------------------------------------------------------------------------------------------|
| `status.success`   | `Context.Success` (the leading symbol and the success message text).                                          |
| `status.warn`      | `Context.Warn`.                                                                                               |
| `status.error`     | `Context.Error`, the `error:` prefix on uncaught errors.                                                      |
| `status.info`      | `Context.Info`, version-line text, spinner styling.                                                           |
| `text.label`       | Left side of `key=value` output, structured-output headers, help section titles.                              |
| `text.value`       | Right side of `key=value`, table cell values.                                                                 |
| `text.title`       | Help titles and other prominent section titles.                                                               |
| `text.body`        | Help body copy and other multi-line prose.                                                                    |
| `text.muted`       | De-emphasized chrome (overridable by `table.border`, `list.enumerator`, `tree.enumerator` for finer control). |
| `table.border`     | Characters drawn between table cells. *Default-aliased to `text.muted`.*                                      |
| `table.header`     | Cells in a table's header row. *Default-aliased to `text.title`.*                                             |
| `table.cell`       | Cells in a table's data rows. *Default-aliased to `text.value`.*                                              |
| `list.item`        | List item text. *Default-aliased to `text.value`.*                                                            |
| `list.enumerator`  | List enumerator markers. *Default-aliased to `text.muted`.*                                                   |
| `tree.item`        | Tree item text. *Default-aliased to `text.value`.*                                                            |
| `tree.enumerator`  | Tree enumerator markers. *Default-aliased via `list.enumerator` to `text.muted`.*                             |

The full set of constants lives in
[`theme/token.go`](../theme/token.go); add new ones there as the core
grows new output paths.

## Token Aliases

The framework ships a default fall-through chain (`theme.DefaultAliases`)
so manifests can author the primary tokens (`status.*`, `text.*`) and
let chrome tokens (table.border, list.enumerator, tree.enumerator,
list.item, tree.item, table.cell, table.header) inherit through the
chain:

```text
list.enumerator -> text.muted
tree.enumerator -> list.enumerator -> text.muted
table.border    -> text.muted
table.cell      -> text.value
table.header    -> text.title
list.item       -> text.value
tree.item       -> text.value
```

Any manifest can override a step with the per-variant `aliases` field:

```jsonc
"aliases": {
    "tree.item": "text.body",
    "list.item": "text.body"
}
```

An empty value disables the framework default for that key:

```jsonc
"aliases": { "table.border": "" }
```

Cycles (alias chains that loop) surface as a hard error from
`Theme.Resolve` so authoring mistakes never reach the lookup path.

## Optional Fields and Defaults

Manifests may set up to three integration points beyond raw tokens:
chroma (syntax highlighting), glamour (markdown), and prompt
(interactive prompts). Each comes in a **named** form (a string
referencing an upstream library style or one of the four bundled
prompt adapters) and an **inline** form (an object defining the style
directly against the variant's primitives). The two forms are
mutually exclusive on each integration; the JSON Schema and the
manifest validator both enforce that.

When neither form is present, the framework picks a default that
matches the variant key and the resolved [Capabilities](#capabilities):

| Integration | When omitted, the framework substitutes…                                                                                                                                              |
|-------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `chroma`    | `monokai` for `dark`, `github` for `light`, none for `notty`. `Context.Code` then falls back to chroma's own default if the name does not resolve.                                     |
| `glamour`   | `notty` when the variant is `notty` or the primary stream is non-interactive; `dark` when `Capabilities.Dark` is true; `light` otherwise. `Context.Markdown` falls back to raw text on glamour init failure. |
| `prompt`    | `theme.PromptFromTokens` derives a `huh.Theme` from the variant's resolved tokens, so prompts share the same palette as the rest of the output even if no adapter is named.           |

`prompt` and `promptStyle` may also be declared at the **top level** of
the manifest, outside the `variants` map; in that case every variant
inherits the shared adapter / inline block unless it declares its own.

This is what lets `theme.Default` work on every terminal without
hard-coding a separate manifest per environment.

## Custom Style Authoring Paths

There are two ways to ship a custom chroma / glamour / prompt
configuration with a theme: reference an upstream name, or define the
style inline against this variant's primitives.

### 1. Reference an Upstream Name

Use the named field if the styling already exists in the upstream
library catalog. This is the path for "I want the Catppuccin look,
change nothing" cases.

```jsonc
// theme/data/dracula.json (excerpt)
{
  "prompt": "dracula",
  "variants": {
    "dark": {
      "chroma":  "dracula",
      "glamour": "dracula"
    }
  }
}
```

The framework wires upstream chroma / glamour through their own
catalogs. The `prompt` field is a **closed catalog** of four bundled
upstream wrappers — `charm`, `base16`, `dracula`, `catppuccin` — defined
in [`theme/internal/manifest/huh_adapters.go`](../theme/internal/manifest/huh_adapters.go).
Anything not in that list is rejected at registry init with the four
valid names listed in the error.

Unknown chroma / glamour names are tolerated — those libraries fall
back to their own defaults rather than failing — so a theme targeting a
future upstream addition will simply look plain until that release
ships.

### 2. Define the Style Inline

For everything else, embed the definition in the manifest. The three
inline forms — `chromaStyle`, `glamourStyle`, `promptStyle` — speak
the same style-spec dialect as the `tokens` map and may reference
primitives and tokens with `$primitive` / `$token`. They are
mutually exclusive with their named counterparts.

The brand-aligned `nabat` theme uses the inline path for *both* its
chroma palette and its prompt block, so the manifest is the only
place where its styling lives — there is no accompanying Go file:

```jsonc
{
  "chromaStyle": {
    "background":      { "bg": { "$primitive": "darkBg" } },
    "name_function":   { "$primitive": "turquoise" },
    "comment":         { "$primitive": "threadGray", "italic": true },
    "keyword":         { "$primitive": "saffronGold", "bold": true }
  },
  "promptStyle": {
    "title":            { "$token": "text.title" },
    "selectedPrefix":   { "$primitive": "pistachio",  "text": "✓ " },
    "unselectedPrefix": { "$primitive": "threadGray", "text": "• " },
    "buttonFocused":    {
      "fg":   { "$primitive": "warmCream" },
      "bg":   { "$primitive": "crystalGold" },
      "bold": true
    },
    "border":           "rounded"
  }
}
```

#### Inline `chromaStyle`

Keys are snake_case chroma token names — `name_function`,
`literal_string`, `generic_deleted`, `background`, `keyword`, etc.
The parser converts the snake_case key to chroma's PascalCase
identifier and calls `chroma.TokenTypeString`; unknown keys are
rejected at registry init with the offending key in the error.

Values accept either form:

- A raw chroma `StyleEntry` string (`"bold #E05454"`, `"bg:#1F1A16"`,
  `"italic #8A7E72"`) for tight one-line entries.
- A unified `styleSpec` object that may use `$primitive` / `$token`
  references. The parser converts the resolved `lipgloss.Style` into
  the equivalent chroma StyleEntry directives.

Mutually exclusive with `chroma`.

#### Inline `glamourStyle`

A **curated** six-field surface that projects onto a full
[`glamour/v2/ansi.StyleConfig`](https://pkg.go.dev/charm.land/glamour/v2/ansi#StyleConfig).
Six fields cover the regions themes routinely customize:

| Field        | Projects onto                                                    |
|--------------|------------------------------------------------------------------|
| `headings`   | `Heading` plus every per-level `H1`..`H6` (prefixes preserved).  |
| `code`       | `Code` (inline code spans).                                      |
| `link`       | `Link` and `LinkText` (URLs share the link styling).             |
| `blockquote` | `BlockQuote`.                                                    |
| `emphasis`   | `Emph`.                                                          |
| `strong`     | `Strong`.                                                        |

The base config is chosen from glamour's defaults by variant key and
`Capabilities` (`NoTTYStyleConfig` for `notty` or non-interactive,
`DarkStyleConfig` for dark, `LightStyleConfig` otherwise) so any region
the manifest does not touch keeps a sensible look. Themes that need
glamour's full surface (per-language code styles, table cell tints,
list indentation, etc.) should use the named `glamour` field instead.

Each field is a `styleSpec`; color slots accept hex literals or
`$primitive` / `$token` references. Mutually exclusive with `glamour`.

#### Inline `promptStyle`

A **flat** Nabat-native block — one entry per prompt slot. The
framework owns the projection onto huh's struct shape, so manifest
authors never see fields like `focused.textInput.cursor` or have to
deal with `$inherit` between focused and blurred.

| Field              | Slot                                                                                                       |
|--------------------|------------------------------------------------------------------------------------------------------------|
| `title`            | Group / section title above prompts.                                                                       |
| `description`      | Help / explanatory text rendered under each prompt.                                                        |
| `cursor`           | Text-input cursor.                                                                                         |
| `placeholder`      | Text-input placeholder copy.                                                                               |
| `selectedOption`   | Currently-selected list item.                                                                              |
| `unselectedOption` | Items not currently selected.                                                                              |
| `selectedPrefix`   | Marker drawn next to the selected item (e.g. `"✓ "`).                                                      |
| `unselectedPrefix` | Marker drawn next to non-selected items (e.g. `"• "`).                                                     |
| `error`            | Error indicators and messages.                                                                             |
| `help`             | Keybind footer text.                                                                                       |
| `selector`         | Active-row indicator and navigation arrows (next / prev).                                                  |
| `buttonFocused`    | Focused submit / next button.                                                                              |
| `buttonBlurred`    | Inactive button.                                                                                           |
| `border`           | Form / card border preset (`hidden`, `normal`, `rounded`, `thick`, `double`, `block`, `outerHalfBlock`, `innerHalfBlock`). |

Each style field is a `styleSpec`; color slots accept hex literals or
`$primitive` / `$token` references. Mutually exclusive with `prompt`.

When a theme needs huh's full per-state surface (separate focused /
blurred styling, custom textInput layout, etc.), drop into the
programmatic path with a `huh.Theme` of your own and set
`Palette.Huh` directly.

## JSON Schema

The schema lives in two places that must agree:

- **In-repo source of truth:** [`theme/schema/v1.json`](../theme/schema/v1.json).
  Embedded into the binary via `//go:embed`; exposed through
  `theme.Schema() []byte`.
- **Public mirror:** `https://nabat.dev/schemas/theme/v1.json`. The
  same bytes, served from the project's website. Manifests reference
  this URL in the `$schema` field; editors fetch it for validation.

Drift between the two is impossible to commit by accident:

- `TestSchemaIDMatchesPublicURL` pins the `$id` field inside the
  document to the public URL constant. Renaming either the file or
  the URL fails the build.
- `TestManifestsMatchSchema` validates every embedded manifest
  against the embedded schema. A schema change that breaks an
  existing manifest fails the build.

## Schema Hosting

The public URL `https://nabat.dev/schemas/theme/v1.json` is a contract
with every theme author who has ever copied an `$schema` field out of a
Nabat manifest. The hosting requirement is therefore: **the bytes
served at that URL must match the bytes at `theme/schema/v1.json` in
the latest tagged release.**

Two viable mechanisms (the choice is an infra-config decision, not a
code decision):

- **Cloudflare Worker / GitHub Pages redirect** to the GitHub raw URL
  of `theme/schema/v1.json` on the latest tag. No separate publish
  step.
- **Static mirror** that copies the file into the `nabat.dev` site on
  every release. Caches well at the CDN edge.

Either way, the in-repo file is canonical. If the public mirror is
ever unreachable, JSON Schema validators that have already cached the
document keep working (the `$id` field inside the document is the
canonical identifier per the JSON Schema spec).

## Schema Versioning

Pre-1.0 the schema is **rewritten in place** as the framework
matures. The current `v1.json` is the only version; manifest authors
migrate alongside the framework. There is no compat shim.

After 1.0 the URL versioning scheme (`v1.json`, `v2.json`, ...) will
take effect; the in-repo file at the lowest-numbered version stays
canonical for that version, and breaking changes ship as a new
numbered file. Until then, breaking changes happen.

## Adding a Theme

Adding a built-in theme is a JSON-only edit. There is no accompanying
Go file, no registry call, no `init()` hook to wire.

1. Drop a JSON file at `theme/data/<name>.json` matching the schema.
   The filename (without `.json`) becomes the registry key — it must
   equal the manifest's `name` field and match the
   `^[a-z0-9][a-z0-9-]*$` pattern.
2. (Optional) Add a string constant in `theme/names.go` so callers
   get IDE autocomplete and the drift test enforces that the constant
   has a matching manifest. Anonymous string names work too —
   `nabat.WithTheme("my-internal-theme")` is valid as long as a
   manifest exists.
3. Run `go test ./theme/...`. The drift test confirms every constant
   has a matching manifest, `TestManifestsMatchSchema` validates the
   new file against the schema, and `TestEveryThemeResolves` confirms
   the theme produces a usable `theme.ResolvedTheme` across every
   `Capabilities` permutation.

If the styling you want cannot be expressed as a manifest — for example
a `huh.Theme` closure that varies on `Capabilities` — implement the
`theme.Recipe` interface from your application and pass it to
`nabat.WithCustomTheme(...)` instead; that path keeps a programmatic
recipe out of the built-in catalog without requiring a registration
API.

The schema can be exercised against hand-written manifests using
`theme.Schema()` and the
[`github.com/santhosh-tekuri/jsonschema/v6`](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6)
compiler — see `theme/catalog_test.go` for the canonical pattern.

## Capabilities

`theme.Capabilities` is the snapshot of terminal facts the framework
detects once at `App.finalize` time and passes to `Theme.Resolve`:

```go
type Capabilities struct {
    Dark           bool                 // dark terminal background
    BackgroundHex  string               // exact background color when detectable
    Profile        colorprofile.Profile // active color profile of stdout
    Interactive    bool                 // primary stream is a TTY
    Width          int                  // terminal width in cells; 0 when unknown
    Hyperlinks     bool                 // OSC 8 supported
    Unicode        UnicodeLevel         // ASCII / Wide / Emoji
    ReducedMotion  bool                 // suppress animations
}
```

Detection happens in the `nabat` root package using the same
`colorprofile` and `xterm` libraries the IOStreams bundle relies on,
plus environment-variable heuristics (`TERM_PROGRAM`, `LANG`, `NO_MOTION`,
etc.). Defaults are conservative: when in doubt, the framework reports
the safer (less-feature) value.

The leaf `nabat.dev/theme` package has no IO dependency, so tests can
construct a `Capabilities` value directly to exercise theme branches
without standing up an IO bundle.

## Token Requirements

Extensions and core consumers can declare which tokens they read via
the `theme.Requirement` machinery. The framework cross-checks the
declared set against the resolved theme at `App.finalize` time and
surfaces missing tokens as a diagnostic — warn-by-default
(stderr line) or hard error via `nabat.WithStrictThemeRequirements()`.

Extensions opt in by implementing the optional sub-interface:

```go
func (e *Extension) ThemeRequires() theme.Requirement {
    return theme.Require("logging extension",
        theme.StatusInfo, theme.StatusWarn, theme.StatusError,
        theme.TextLabel, theme.TextValue,
    )
}
```

The framework's own consumers are declared in `theme.CoreRequirements()`;
adding a new core consumer (a new `Status*`, `Text*`, `Table*`, etc.)
means adding to the right `Requirement` there so the missing-token
diagnostic stays accurate.

Authors who intentionally ship a sparse theme (the `minimal` theme,
for example) leave strict mode off and ignore the warning; CIs and
tests that want the regression catch flip the option.
