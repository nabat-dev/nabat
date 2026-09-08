# Themes

The theme package, JSON manifests, tokens, and how chroma / glamour / prompt
styles attach.

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

Theming lives in one package. That package does not import `nabat.dev`:

```text
nabat ──► nabat/theme  (leaf + catalog)
                  └── theme/internal/manifest  (parser, no theme imports)
```

| Package        | Role                                                                                                       |
|----------------|------------------------------------------------------------------------------------------------------------|
| `nabat.dev/theme` | **Leaf primitives + built-in catalog.** `Theme` (data), `Token`, `Capabilities`, `Variant`, `ResolvedTheme`, `Palette`, `Prompt`, `Resolver` interface; embedded JSON manifests under `data/`, JSON Schema under `schema/`, lazy registry (`Get` / `Names` / `All` / `Schema` / `Manifest`), untyped string constants for every shipped name, and a closed catalog of bundled upstream `huh.Theme` wrappers (`charm`, `base16`, `dracula`, `catppuccin`). No imports from `nabat.dev`. |
| `nabat.dev`    | **Wiring.** `nabat.WithTheme(name)` looks up the registry, `nabat.WithCustomTheme(theme.Resolver)` accepts any `Resolver` value (including a plain `theme.Theme`). `App.finalize` detects `Capabilities` and pins a `theme.ResolvedTheme` for the lifetime of the app. For a concrete `theme.Theme`, finalize uses `Theme.ResolveErr`; other resolvers have only `Resolve`. `App.Theme()` returns the pinned result. |

`theme.Theme` is data, not a function. It holds one `Palette` per variant, a
default variant when capabilities do not pin one, and a few cross-variant
defaults. Resolve picks a variant, fills empty cascade slots, and returns an
immutable `ResolvedTheme`.

That resolve runs once at `nabat.New(...)`. After that, `App.Theme()` is
fixed; `Success`, `Warn`, `Table`, help, Markdown, and the logging extension
all read it.

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

Unknown names return a `*ConfigErrors` from `nabat.New` that lists every
registered name.

The catalog shipped with Nabat:

| Constant                    | Manifest                         | Best for                                                                                          |
|-----------------------------|----------------------------------|---------------------------------------------------------------------------------------------------|
| `theme.Default`             | `data/default.json`              | Capability-aware default with dark, light, and notty variants. Variant selection follows primary-output TTY state and detected background luminance; output color depth is adapted separately. |
| `theme.Minimal`             | `data/minimal.json`              | Low-color, single `notty` variant. Status and accent roles primarily rely on bold styling, while text, link, and code roles use neutral foreground and background primitives. |
| `theme.Charm`               | `data/charm.json`                | Higher-contrast Charm.land palette for dark terminals.                                            |
| `theme.Dracula`             | `data/dracula.json`              | Dracula Classic (dark) and Alucard Classic (light).                                               |
| `theme.Gruvbox`             | `data/gruvbox.json`              | morhetz/gruvbox dark and light.                                                                   |
| `theme.CatppuccinLatte`     | `data/catppuccin-latte.json`     | Catppuccin Latte for light backgrounds.                                                           |
| `theme.CatppuccinFrappe`    | `data/catppuccin-frappe.json`    | Catppuccin Frappé for dark backgrounds.                                                           |
| `theme.CatppuccinMacchiato` | `data/catppuccin-macchiato.json` | Catppuccin Macchiato for dark backgrounds.                                                        |
| `theme.CatppuccinMocha`     | `data/catppuccin-mocha.json`     | Catppuccin Mocha for dark backgrounds.                                                            |
| `theme.Nabat`               | `data/nabat.json`                | Brand palette. Token-derived chroma, glamour, and prompt color; `promptKnobs` for prefixes/border. |
| `theme.Nord`                | `data/nord.json`                 | Nord for dark backgrounds.                                                                        |
| `theme.Solarized`           | `data/solarized.json`            | Solarized dark and light.                                                                         |

## Programmatic Themes

When the styling you want cannot be expressed as a JSON manifest (for
example a `huh.Theme` closure or a `chroma.Style` you want to own in Go),
build a `theme.Theme` struct and install it with `nabat.WithCustomTheme`:

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
                theme.StatusError:   lipgloss.NewStyle().Foreground(lipgloss.Color("#E05454")).Bold(true),
                theme.TextPrimary:   lipgloss.NewStyle().Foreground(lipgloss.Color("#EDE4D3")),
                theme.TextSecondary: lipgloss.NewStyle().Foreground(lipgloss.Color("#D5CDC2")),
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
  `theme.Resolver` interface directly:

  ```go
  type myResolver struct{}
  func (myResolver) Resolve(c theme.Capabilities) theme.ResolvedTheme {
      // pick a Palette / Theme based on c.Dark, c.Interactive, ...
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
        theme.SetAlias(theme.ListItem, theme.TextSecondary),
        theme.SetChromaName("monokai"),
    ),
)
```

Overrides apply to **every variant** of the underlying theme, so a
multi-variant manifest stays multi-variant after the override; the
same one-line tweak affects whichever variant the runtime
capabilities pick.

> [!WARNING]
> Overrides are silently ignored when the active theme is a bespoke
> `theme.Resolver` (anything other than a `theme.Theme` value). The resolver's
> `Resolve` method is opaque, so the framework cannot apply per-`Palette`
> overrides into it.

## Manifest Format

A theme manifest is a JSON document describing one Nabat theme.
The format is defined by [`theme/schema/v1.json`](../theme/schema/v1.json).

Nabat's theme architecture is inspired by design-token concepts such as
primitives, semantic roles, references, and aliases. The JSON manifest
format itself is Nabat-specific. It is not the
[DTCG Design Tokens Format](https://www.w3.org/community/reports/design-tokens/CG-FINAL-format-20251028/).
Nabat uses `$primitive` and `$token`, not DTCG `$value` / `$type`.

Each manifest is organized per **variant**:

1. **Variants**: top-level map keyed by mode (`dark` / `light` /
   `notty`). Each entry is a self-contained palette.
2. **Primitives**: per-variant named raw colors (hex literals).
3. **Tokens**: per-variant named semantic styles that reference
   primitives or other tokens.
4. **Named integrations** (optional): per-variant `chroma`, `glamour`,
   and `huh` strings.
5. **`promptKnobs`** (optional, top-level): prefixes and border for the
   Nabat-native prompt path.

The schema accepts these top-level fields: `$schema`, `name`,
`description`, `default`, `promptKnobs`, `variants`.

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
    "dark": {
      "primitives": { "fg": "#FFFFFF", "bg": "#000000" },
      "tokens": { "text.primary": { "$primitive": "fg" } },
      "huh": "charm"
    },
    "light": {
      "primitives": { "fg": "#000000", "bg": "#FFFFFF" },
      "tokens": { "text.primary": { "$primitive": "fg" } }
    }
  }
}
```

The `$schema` field points at the public JSON Schema URL so editors
that support JSON Schema (VSCode, JetBrains, Zed, Neovim with
`coc-json`) get live validation, autocomplete, and hover documentation.

### Top-level fields

| Field          | Type   | Purpose                                                                                                                |
|----------------|--------|------------------------------------------------------------------------------------------------------------------------|
| `name`         | string | Identifier; matches `^[a-z0-9][a-z0-9-]*$`. Must equal the manifest's filename (without `.json`).                      |
| `variants`     | object | Keyed by mode (`dark` / `light` / `notty`). Each entry has its own `primitives` + `tokens`.                            |
| `default`      | enum   | `dark` / `light` / `notty`. Required when more than one variant is declared; optional for single-variant themes.       |
| `description`  | string | Optional human-readable summary.                                                                                       |
| `promptKnobs`  | object | Optional theme-wide prompt prefixes and border. See [Custom Style Authoring Paths](#custom-style-authoring-paths).     |

Per-variant required fields:

| Field        | Type   | Purpose                                                                                                                |
|--------------|--------|------------------------------------------------------------------------------------------------------------------------|
| `primitives` | object | Map of name to hex color (`^#[0-9A-Fa-f]{6}$`). Tokens reference these via `$primitive`.                               |
| `tokens`     | object | Map of token name (for example `status.success`) to a style spec.                                                      |

Per-variant optional fields:

| Field     | Type   | Purpose                                                                                         |
|-----------|--------|-------------------------------------------------------------------------------------------------|
| `aliases` | object | Override or disable [default alias](#token-aliases) targets.                                    |
| `chroma`  | string | Upstream chroma style name. Unknown names fail at parse time.                                   |
| `glamour` | string | Upstream glamour preset name. Unknown names fail at parse time.                                 |
| `huh`     | enum   | Upstream prompt adapter: `charm`, `base16`, `dracula`, or `catppuccin`.                         |

### Style Spec

A style spec is an object with at least one field:

| Field             | Type    | Purpose                                                                                                                            |
|-------------------|---------|------------------------------------------------------------------------------------------------------------------------------------|
| `$primitive`      | string  | Reference to a key in this variant's `primitives` map. Sets the foreground color. Mutually exclusive with `$token`.                |
| `$token`          | string  | Reference to a key in this variant's `tokens` map. Inherits that token's resolved style; other fields layer on top.                |
| `fg` / `bg`       | hex / colorRef | Explicit foreground / background colors.                                                                                    |
| `borderForeground` / `borderBackground` | hex / colorRef | Border colors.                                                                                            |
| `border`          | enum    | Lipgloss border preset (`hidden`, `normal`, `rounded`, `thick`, `double`, `block`, `outerHalfBlock`, `innerHalfBlock`). |
| `bold`, `italic`, `underline`, `strikethrough`, `faint`, `blink`, `reverse` | bool | Lipgloss attribute toggles.                                                          |
| `text`            | string  | Literal text stored on the lipgloss style.                                                                         |

`$primitive` and `$token` are mutually exclusive within a single spec.
Unknown attributes are rejected at parse time (the JSON Schema sets
`additionalProperties: false` on the style spec).

## Token Catalog

`nabat.dev/theme` defines the well-known tokens used by core consumers.
Token names are an open set: third-party themes and extensions may add
their own and read them back via `ResolvedTheme.Style(token)`. A theme
that omits a token falls through the **alias chain** before hitting
the zero `lipgloss.Style` (terminal default).

| Token              | Consumer                                                                                                      |
|--------------------|---------------------------------------------------------------------------------------------------------------|
| `status.success`   | Leading symbol on `Context.Success`.                                                                          |
| `status.warning`   | Leading symbol on `Context.Warn`.                                                                             |
| `status.error`     | Leading symbol on `Context.Error`, and the `error:` prefix on uncaught errors.                                |
| `status.info`      | Leading symbol on `Context.Info`; spinner title; version line.                                                |
| `status.active`    | Spinner icon on active `Context.Status` rows. *Default-aliased to `status.info`.*                             |
| `text.primary`     | Primary body text; default for table/list/tree item text; status metadata values.                             |
| `text.secondary`   | Descriptive text, help body, and other prose.                                                                 |
| `text.title`       | Help titles, table headers, and other prominent section titles.                                               |
| `text.link`        | Hyperlinks.                                                                                                   |
| `text.muted`       | De-emphasized chrome (overridable by `table.border`, `list.enumerator`, `tree.enumerator` for finer control). |
| `accent.primary`   | Labels, key chrome, help section headings, status metadata keys.                                              |
| `code.surface`     | Code-oriented surfaces.                                                                                       |
| `table.border`     | Characters drawn between table cells. *Default-aliased to `text.muted`.*                                      |
| `table.header`     | Cells in a table's header row. *Default-aliased to `text.title`.*                                             |
| `table.cell`       | Cells in a table's data rows. *Default-aliased to `text.primary`.*                                            |
| `list.item`        | List item text. *Default-aliased to `text.primary`.*                                                          |
| `list.enumerator`  | List enumerator markers. *Default-aliased to `text.muted`.*                                                   |
| `tree.item`        | Tree item text. *Default-aliased to `text.primary`.*                                                          |
| `tree.enumerator`  | Tree enumerator markers. *Default-aliased via `list.enumerator` to `text.muted`.*                             |
| `spinner.active`   | Live spinner icon. *Default-aliased to `status.info`.*                                                        |

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
table.cell      -> text.primary
table.header    -> text.title
list.item       -> text.primary
tree.item       -> text.primary
spinner.active  -> status.info
status.active   -> status.info
```

Any manifest can override a step with the per-variant `aliases` field:

```jsonc
"aliases": {
    "tree.item": "text.secondary",
    "list.item": "text.secondary"
}
```

An empty value disables the framework default for that key:

```jsonc
"aliases": { "table.border": "" }
```

Cycles (alias chains that loop) are reported by `Theme.ResolveErr`.
`Theme.Resolve` intentionally discards resolution errors; callers that
need diagnostics should use `Theme.ResolveErr`.

When a concrete `theme.Theme` is installed on a Nabat App, `App.finalize`
uses `ResolveErr`, so the error fails app construction rather than
reaching the first `Style` lookup.

## Optional Fields and Defaults

A variant may name upstream chroma, glamour, and huh styles. Theme-wide
prompt knobs live at the top level as `promptKnobs`.

These JSON fields are names or small knobs. They are not inline style
trees. Owned chroma, glamour, prompt, and huh values are a programmatic
`Palette` concern (`nabat.WithCustomTheme`), not extra manifest objects.

When a named field is omitted, `Theme.Resolve` fills
the slot:

| Field | Where | When omitted, Resolve uses |
| --- | --- | --- |
| `chroma` | per variant | `ChromaFromTokens` from the variant's tokens |
| `glamour` | per variant | `GlamourFromTokens` on the preset from `GlamourPreset`: `notty` when the variant is `notty` or `Capabilities.Interactive` is false; otherwise `dark` or `light` from `Capabilities.Dark` |
| `huh` | per variant | `PromptFromTokens`, then top-level `promptKnobs` if set |
| `promptKnobs` | top-level | no extra prefixes or border; token-derived prompt colors still apply |

`huh` is a closed catalog: `charm`, `base16`, `dracula`, `catppuccin`.
Unknown `huh`, `chroma`, and `glamour` names fail at parse time.

If a variant sets `huh`, that upstream wrapper wins. `promptKnobs` are
not applied to it. `promptKnobs` apply only to the Nabat-native prompt
path (`Palette.Prompt` or `PromptFromTokens`).

`Context.Markdown` falls back to raw text if glamour init fails.

That is how `theme.Default` covers dark, light, and notty without a
separate file per environment.

## Custom Style Authoring Paths

### 1. Name an upstream style in the manifest

Use the named field when the look already exists upstream.

```jsonc
// theme/data/dracula.json (excerpt)
{
  "variants": {
    "dark": {
      "chroma":  "dracula",
      "glamour": "dracula",
      "huh":     "dracula"
    }
  }
}
```

The `huh` adapters live in
[`theme/internal/manifest/huh_adapters.go`](../theme/internal/manifest/huh_adapters.go).
Anything else is rejected at parse time with the four valid names in the
error.

### 2. Set `promptKnobs` on the token-derived prompt path

The Nabat brand theme does this. It does not set `chroma`, `glamour`, or
`huh`; those integrations are derived from tokens. Prefixes and border
come from `promptKnobs`:

```jsonc
{
  "promptKnobs": {
    "selectedPrefix": "✓ ",
    "unselectedPrefix": "• ",
    "border": "rounded"
  }
}
```

| Field | Purpose |
| --- | --- |
| `selectedPrefix` | Literal marker before selected options. |
| `unselectedPrefix` | Literal marker before unselected options. |
| `border` | Form/card border preset (`hidden`, `normal`, `rounded`, `thick`, `double`, `block`, `outerHalfBlock`, `innerHalfBlock`). |
| `borderColor` | Focused border foreground. Hex literal only (`#RRGGBB`). |

### 3. Own the style in Go

When a named upstream style is not enough, build a `theme.Theme` and set
`Palette.Chroma`, `Palette.Glamour` / `Palette.GlamourFor`,
`Palette.Prompt`, or `Palette.Huh`.

Resolve order:

- Chroma: `Palette.Chroma` → `Palette.ChromaName` → `ChromaFromTokens`
- Glamour: `Palette.Glamour` → `Palette.GlamourFor` → `Palette.GlamourName` → `GlamourFromTokens`
- Prompt: `Palette.Huh` → `Palette.Prompt` (plus `Theme.PromptKnobs`) → `PromptFromTokens` (plus `Theme.PromptKnobs`)

`Palette.Huh` wins over `Palette.Prompt` and `PromptFromTokens`. Use it
when the closed `Prompt` surface is not enough.

## JSON Schema

The schema lives in two places that should agree:

- **In-repo source of truth:** [`theme/schema/v1.json`](../theme/schema/v1.json).
  Embedded into the binary via `//go:embed`; exposed through
  `theme.Schema() []byte`.
- **Public mirror:** `https://nabat.dev/schemas/theme/v1.json`. The
  same bytes, served from the project's website. Manifests reference
  this URL in the `$schema` field; editors fetch it for validation.

Repository tests keep the embedded schema, its public `$id`, and bundled
manifests internally consistent.

- `TestSchemaIDMatchesPublicURL` pins the `$id` field inside the
  document to the public URL constant. Renaming either the file or
  the URL fails the build.
- `TestManifestsMatchSchema` validates every embedded manifest
  against the embedded schema. A schema change that breaks an
  existing manifest fails the build.

Publishing the same schema bytes at the public URL is a release/hosting
responsibility and is not verified by these unit tests.

## Schema Hosting

> [!IMPORTANT]
> The public URL `https://nabat.dev/schemas/theme/v1.json` is a contract
> with every theme author who has copied an `$schema` field out of a
> Nabat manifest. The bytes served at that URL must match
> `theme/schema/v1.json` in the latest tagged release.

Hosting is infra config, not library code. Two options that work:

- Redirect (Cloudflare Worker or GitHub Pages) to the GitHub raw URL of
  `theme/schema/v1.json` on the latest tag.
- Copy the file into the `nabat.dev` site on each release.

The in-repo file is canonical. If the public URL is down, validators that
already cached the document still work (`$id` inside the document is the
identifier per the JSON Schema spec).

## Schema Versioning

> [!WARNING]
> Pre-1.0 the schema is **rewritten in place**. `v1.json` is the only
> version; update manifests with the framework. No compat shim.

After 1.0 we will version URLs (`v1.json`, `v2.json`, ...). Breaking changes
then ship as a new file. Until then, breaking changes land in place.

## Adding a Theme

Adding a built-in theme is a JSON-only edit. There is no accompanying
Go file, no registry call, no `init()` hook to wire.

1. Drop a JSON file at `theme/data/<name>.json` matching the schema.
   The filename (without `.json`) becomes the registry key. It must
   equal the manifest's `name` field and match the
   `^[a-z0-9][a-z0-9-]*$` pattern.
2. (Optional) Add a string constant in `theme/names.go` so callers
   get IDE autocomplete and the drift test enforces that the constant
   has a matching manifest. Anonymous string names work too:
   `nabat.WithTheme("my-internal-theme")` is valid as long as a
   manifest exists.
3. Run `go test ./theme/...`. The drift test confirms every constant
   has a matching manifest, `TestManifestsMatchSchema` validates the
   new file against the schema, and `TestEveryThemeResolves` confirms
   the theme produces a usable `theme.ResolvedTheme` across every
   `Capabilities` permutation.

If the styling you want cannot be expressed as a manifest (for example
a `huh.Theme` closure that varies on `Capabilities`), implement the
`theme.Resolver` interface from your application and pass it to
`nabat.WithCustomTheme(...)` instead. That path keeps a programmatic
resolver out of the built-in catalog without requiring a registration
API.

The schema can be exercised against hand-written manifests using
`theme.Schema()` and the
[`github.com/santhosh-tekuri/jsonschema/v6`](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v6)
compiler. See `theme/catalog_test.go` for the canonical pattern.

## Capabilities

`theme.Capabilities` is the snapshot of terminal facts the framework
detects once at `App.finalize` time and passes to theme resolution:

```go
type Capabilities struct {
    Dark           bool                 // dark terminal background
    BackgroundHex  string               // exact background color when detectable
    Profile        colorprofile.Profile // active color profile of stdout
    Interactive    bool                 // primary output stream is a TTY
    Width          int                  // terminal width in cells; 0 when unknown
    Hyperlinks     bool                 // OSC 8 supported
    Unicode        UnicodeLevel         // ASCII / Wide / Emoji
    ReducedMotion  bool                 // reduced-motion preference was detected
}
```

`Interactive` reports whether the primary output stream is a TTY for theme
resolution (`io.IsStdoutTTY()`). Despite the field name, this capability is
not the same as prompt interactivity. Prompt availability is determined
separately by the command/context I/O state.

`Theme.pickVariant` uses `Interactive` to prefer a `notty` variant when
primary output is not a TTY. `Profile` is not part of that variant-selection
algorithm; color depth is adapted separately.

`ReducedMotion` is detected from environment signals and exposed to theme
resolution. Spinner and Status do not currently consume it to disable
animation.

Detection happens in the `nabat` root package using the same
`colorprofile` and `xterm` libraries the IOStreams bundle relies on,
plus environment-variable heuristics (`TERM_PROGRAM`, `LANG`,
`NABAT_REDUCED_MOTION`, `REDUCE_MOTION`, `NO_MOTION`, and similar).
Defaults are conservative: when in doubt, the framework reports
the safer (less-feature) value.

The leaf `nabat.dev/theme` package has no IO dependency, so tests can
construct a `Capabilities` value directly to exercise theme branches
without standing up an IO bundle.

## Token Requirements

Extensions and core consumers can declare which tokens they read via
the `theme.Requirement` machinery. The framework cross-checks the
declared set against the resolved theme at `App.finalize` time and
surfaces missing tokens as a diagnostic (warn-by-default on stderr, or
a hard error via `nabat.WithStrictThemeRequirements()`).

Extensions opt in by implementing the optional sub-interface:

```go
func (e *Extension) ThemeRequires() theme.Requirement {
    return theme.Require("logging extension",
        theme.StatusInfo, theme.StatusWarning, theme.StatusError,
        theme.AccentPrimary, theme.TextPrimary,
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
