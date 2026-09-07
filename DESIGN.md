# Nabat Design System & Brand Specification

> **Status:** Living design contract  
> **Project:** [nabat-dev/nabat](https://github.com/nabat-dev/nabat)  
> **Palette:** [nabat-dev/palette](https://github.com/nabat-dev/palette)

This document defines the brand identity, visual language, terminal UX conventions, and cross-surface design rules for Nabat.

It is intentionally not the source of truth for raw color values, API architecture, or theme implementation details. Those responsibilities are owned by the repositories and documents described in [Sources of Truth](#2-sources-of-truth).

---

## 1. Scope

Nabat has several related but distinct design surfaces:

- the Nabat project and brand;
- command-line experiences built with Nabat;
- Nabat's built-in theme system;
- the Nabat color palette;
- documentation, examples, screenshots, and marketing material;
- the future `nabat.dev` website.

This document connects those surfaces through a common design language.

It answers questions such as:

- What should Nabat feel like?
- What visual ideas make Nabat recognizable?
- How should semantic CLI output behave?
- How should colors be used without coupling components to raw color values?
- How should terminal behavior degrade in pipes, CI, and `NO_COLOR` environments?
- How should the future website visually relate to the CLI framework?
- Where does each design decision belong?

It does **not** duplicate implementation documentation or machine-readable palette data.

---

## 2. Sources of Truth

Nabat deliberately separates brand intent, color data, semantic styling, and engineering architecture.

| Concern | Canonical source |
| --- | --- |
| Brand identity and visual language | This `DESIGN.md` |
| Terminal UX design rules | This `DESIGN.md` |
| Raw colors, names, RGB/hex values, variants, ANSI mappings | [`nabat-dev/palette`](https://github.com/nabat-dev/palette) |
| Nabat semantic theme tokens | [`theme/token.go`](theme/token.go) |
| Theme aliases and fallbacks | [`theme/aliases.go`](theme/aliases.go) |
| Theme manifests and runtime theme behavior | [`theme/`](theme/) |
| Theme authoring and manifest documentation | [`docs/themes.md`](docs/themes.md) |
| API and engineering design principles | [`docs/design-principles.md`](docs/design-principles.md) |
| Package and implementation architecture | [`docs/architecture.md`](docs/architecture.md) |

### 2.1 No Duplicate Color Authority

`nabat-dev/palette` is the authoritative source for Nabat's raw colors.

Do not maintain a second authoritative table of hex or RGB values in this document.

Consumers should use the machine-readable palette files from the Palette repository.

This includes:

- websites;
- editor themes;
- terminal themes;
- documentation tooling;
- generated CSS variables;
- marketing assets;
- future ports.

The Nabat framework may contain the color values required by its built-in `nabat` theme, but those values represent an implementation of the Nabat Palette rather than a separate palette definition.

When the palette changes, implementations should be updated or generated from the canonical palette rather than independently redesigned.

---

## 3. Brand Identity

### 3.1 Name

The product name is **Nabat**.

Pronunciation:

`/næˈbɑːt/` — approximately **nah-BAHT**

Origin:

Persian **نبات**, traditional crystallized sugar or rock candy.

Nabat is traditionally formed by allowing sugar crystals to grow around a thread. Saffron is commonly used to give the candy a warm golden character.

This physical structure provides the central metaphor for the Nabat project.

### 3.2 Product Description

> **Adaptive CLI framework for Go.**

Nabat builds on Cobra and adds typed, human-oriented and automation-friendly CLI capabilities such as:

- adaptive positional arguments;
- typed options;
- environment resolution;
- interactive prompts;
- structured output;
- semantic status output;
- live status displays;
- themes;
- terminal-aware rendering.

The short product description may evolve independently from this document. The current project README remains authoritative for product positioning and feature inventory.

---

## 4. The Core Metaphor

### 4.1 The Thread

The thread represents **Cobra**.

Cobra provides the structural spine:

- command routing;
- flag parsing;
- command hierarchy;
- completion machinery;
- established CLI behavior.

Like the thread inside traditional Nabat, this structure is essential but should not dominate the visual identity.

The thread is represented visually by:

- thin lines;
- vertical connections;
- dashed paths;
- restrained turquoise accents;
- CLI chevrons and directional marks.

### 4.2 The Crystals

The crystals represent the layers Nabat grows around Cobra.

Examples include:

- adaptive argument resolution;
- prompts;
- semantic output;
- tables, lists, and trees;
- themes;
- spinners;
- multi-row status displays;
- typed APIs;
- structured machine output.

The visual language should suggest progressive crystallization rather than decoration added arbitrarily around a CLI.

### 4.3 Design Interpretation

The metaphor can be summarized as:

```text
               Nabat
                 │
          crystals / layers
                 │
       ┌─────────┼─────────┐
       │         │         │
     Prompt    Output    Themes
       │         │         │
       └─────────┼─────────┘
                 │
              Cobra
                 │
               thread
```

This metaphor should remain recognizable across the logo, website, diagrams, documentation, and other brand surfaces.

---

## 5. Core Design Principles

### 5.1 Warm Over Cool

Nabat should feel warm without becoming ornamental.

Its identity is built around roasted neutrals, saffron gold, warm cream, and restrained botanical or tile-inspired accents.

Many developer tools default to cold blue, purple, cyan, or neon palettes. Nabat deliberately creates a warmer visual environment.

Warmth is part of the identity, not a requirement to make every element gold.

Neutral space remains important.

### 5.2 Two Flavors, Shared Semantics

Nabat Dark and Nabat Light preserve the same semantic language while adapting luminance and contrast to their respective surfaces.

A semantic role should remain stable across variants:

```text
success  → success
warning  → warning
error    → error
info     → info
primary  → primary
muted    → muted
accent   → accent
```

The underlying color value may differ between dark and light variants.

The design system guarantees semantic continuity, not identical hex values.

### 5.3 Standard Meanings Stay Standard

Nabat does not redefine common status conventions merely to appear distinctive.

In general:

- green represents success;
- amber/yellow represents warning;
- red represents error;
- blue/turquoise represents informational or active state.

Brand personality should operate within familiar CLI semantics.

### 5.4 Human UX Must Not Corrupt Machine UX

Human-facing decoration must not break scripts.

Nabat maintains a deliberate separation between:

```text
stdout → command product / machine-consumable output
stderr → human status / diagnostics / interactive progress
```

A user should be able to pipe command output into another program without status messages corrupting the stream.

### 5.5 Progressive Terminal Enhancement

Every important result must remain understandable without color or animation.

The rendering hierarchy is:

```text
plain / piped / CI / NO_COLOR
             ↓
          styled TTY
             ↓
       interactive TTY
```

Color and interactivity enhance the experience. They must not be required to understand it.

### 5.6 Meaning Never Depends on Color Alone

Color is supplementary.

Success, warning, error, selection, progress, and failure should also be expressed through one or more of:

- symbols;
- labels;
- text;
- position;
- structure;
- borders;
- state transitions.

A plain-text rendering must still communicate the same essential meaning.

### 5.7 Terminal First

Nabat is a CLI framework.

Terminal behavior takes precedence over reproducing web-interface conventions inside a terminal.

Design decisions should account for:

- monospace layout;
- ANSI width;
- narrow terminals;
- TTY detection;
- pipes;
- CI logs;
- Unicode availability;
- `NO_COLOR`;
- `TERM=dumb`;
- non-interactive environments.

The website may extend the visual language, but it should not redefine the CLI.

---

## 6. Color Language

The complete palette, exact color names, RGB/hex values, and ANSI mappings belong to:

> [nabat-dev/palette](https://github.com/nabat-dev/palette)

This document defines only how the palette families relate to Nabat's broader design language.

### 6.1 Crystal

**Purpose:** structure and neutral hierarchy.

Crystal contains the warm neutral foundation of the system.

Typical uses:

- backgrounds;
- surfaces;
- body text;
- secondary text;
- muted text;
- dividers;
- borders;
- code surfaces.

The Crystal family should normally carry the majority of a screen or document.

It keeps Nabat restrained while allowing the accent families to remain meaningful.

### 6.2 Saffron

**Purpose:** identity and emphasis.

Saffron is the principal brand family.

Typical uses:

- prominent titles;
- section emphasis;
- important labels;
- brand marks;
- focused controls;
- warm highlights;
- warning roles where semantically appropriate.

Saffron should feel deliberate rather than ubiquitous.

If everything is saffron, saffron stops functioning as emphasis.

### 6.3 Garden

**Purpose:** semantic and secondary accent.

Garden takes inspiration from Persian botanical colors and tilework.

It supplies roles such as:

- success;
- error;
- info;
- selection;
- links;
- secondary visual accents.

The Garden family provides enough contrast to prevent Nabat from becoming visually monochromatic while preserving the warm overall character.

### 6.4 Family Hierarchy

Conceptually:

```text
Crystal
└── structure / neutral foundation

Saffron
└── identity / emphasis

Garden
└── semantic state / interaction / secondary accent
```

Components should not normally choose a palette family directly.

They should consume semantic theme tokens.

---

## 7. Theme Architecture

Nabat separates physical colors from semantic roles.

The conceptual pipeline is:

```text
Nabat Palette
(raw named colors and variants)
          │
          ▼
Theme primitives
(per-variant implementation values)
          │
          ▼
Semantic tokens
(status.*, text.*, accent.*, code.*)
          │
          ▼
Component tokens / aliases
(table.*, list.*, tree.*, spinner.*, ...)
          │
          ▼
CLI components
```

This separation is fundamental.

A component should ask for:

```text
status.success
```

rather than:

```text
pistachio
```

A table should ask for:

```text
table.header
```

rather than:

```text
saffronGold
```

This allows themes to replace appearance without changing component meaning.

---

## 8. Semantic Tokens

Nabat's well-known semantic tokens are defined by `theme.Token`.

Token names describe **roles**, not colors.

### 8.1 Status

```text
status.success
status.warning
status.error
status.info
status.active
```

Use these for states and status-oriented information.

### 8.2 Text

```text
text.primary
text.secondary
text.title
text.link
text.muted
```

General intent:

- `text.primary` — primary content;
- `text.secondary` — supporting or descriptive content;
- `text.title` — prominent titles and section headings;
- `text.link` — links;
- `text.muted` — de-emphasized chrome and metadata.

### 8.3 Accent

```text
accent.primary
```

The primary brand/accent role.

It is commonly used where Nabat needs emphasis without implying success, warning, error, or info.

### 8.4 Code

```text
code.surface
```

Used where a theme needs to define a code-oriented surface.

### 8.5 Structured Output

```text
table.border
table.header
table.cell

list.item
list.enumerator

tree.item
tree.enumerator
```

These component-specific tokens allow a theme to override structured output independently when required.

### 8.6 Interactive Progress

```text
spinner.active
status.active
```

These identify actively running state rather than completed semantic state.

---

## 9. Token Aliases

Nabat allows component tokens to fall back to broader semantic roles.

The framework default relationships are conceptually:

```text
list.enumerator  → text.muted

tree.enumerator  → list.enumerator
                 → text.muted

table.border     → text.muted

list.item        → text.primary
tree.item        → text.primary
table.cell       → text.primary

table.header     → text.title

spinner.active   → status.info
status.active    → status.info
```

This is intentional.

A minimal theme can define a relatively small semantic vocabulary while richer themes may customize individual components.

Themes may override these relationships.

The fallback architecture should remain predictable and acyclic.

---

## 10. Terminal Output Model

Nabat distinguishes between two kinds of output.

### 10.1 Command Product

Command product belongs on `stdout`.

Examples include:

- plain command output;
- tables;
- lists;
- trees;
- JSON;
- YAML;
- TOML;
- encoded machine-readable results.

This allows compositions such as:

```sh
myctl report --json | jq .
```

### 10.2 Human Status and Diagnostics

Human-oriented status belongs on `stderr`.

Examples include:

- success messages;
- warnings;
- errors;
- informational progress;
- spinners;
- live status displays.

A command may therefore produce structured stdout while simultaneously telling the human what it is doing through stderr.

This separation is part of Nabat's design language, not merely an implementation detail.

---

## 11. Semantic Status Messages

Nabat provides four primary semantic message patterns:

```text
✓ deployed        environment=production replicas=3
⚠ slow migration table=audit_logs latency=840ms
✗ build failed    target=darwin/arm64
• retrying        attempt=2 delay=500ms
```

Default symbols:

| Meaning | Symbol | Semantic token |
| --- | :---: | --- |
| Success | `✓` | `status.success` |
| Warning | `⚠` | `status.warning` |
| Error | `✗` | `status.error` |
| Info | `•` | `status.info` |

These messages write to `stderr`.

### 11.1 Message Anatomy

The current semantic message structure is:

```text
<semantic symbol> <message> <key=value> <key=value> ...
```

Styling responsibilities:

- the leading symbol uses its `status.*` token;
- the human-readable message remains ordinary readable text;
- metadata keys use `accent.primary`;
- metadata values use `text.primary`.

The message must remain understandable when all styling is removed.

---

## 12. Aligned Fields

`Context.Fields` produces aligned key/value blocks.

Example:

```text
Name      api
Status    ✓ running
Replicas  3
Region    eu-west
```

### 12.1 Alignment

The default key width is calculated from the longest key in the block.

It is not a globally fixed width.

A caller may explicitly override the width when multiple independently rendered blocks need to align.

### 12.2 ANSI Safety

Padding and visible-width calculations must occur before styling introduces ANSI escape sequences.

ANSI sequences must never corrupt visible column alignment.

### 12.3 Styling

By default:

- keys use `text.muted`;
- values are caller-provided rendered content.

Values may contain pre-rendered content such as badges.

`Fields` should therefore avoid imposing styling that would destroy nested semantic presentation.

---

## 13. Badges

`Context.Badge` is intentionally lightweight.

A badge consists of:

```text
<semantic glyph> <label>
```

Examples:

```text
✓ running
✗ failed
! degraded
• pending
? unknown
```

Built-in icon meanings include:

```text
IconSuccess  → status.success
IconError    → status.error
IconWarning  → status.warning
IconInfo     → status.info
IconUnknown  → text.muted
```

The current CLI badge is **not** a web-style pill and does not require:

- a filled background;
- a rounded rectangle;
- a border;
- padding around the label.

The glyph carries semantic styling while the label remains readable terminal text.

Website badge components may use richer surfaces, but they should preserve the same semantic relationship.

---

## 14. Tables

`Context.Table` is a structured stdout component.

### 14.1 Default Structure

The default table uses a normal single-line Unicode border.

Other supported border shapes may be selected explicitly, including:

- rounded;
- ASCII;
- thick;
- double;
- block;
- Markdown;
- hidden.

Rounded borders are a supported style, not the universal default.

### 14.2 Semantic Styling

By default:

```text
border → table.border
header → table.header
cell   → table.cell
```

These may inherit through the default alias system.

### 14.3 Layout

Table presentation should prioritize:

1. readable data;
2. stable alignment;
3. terminal width;
4. optional decoration.

Zebra striping is not a default Nabat CLI behavior.

If a future component introduces alternating rows, it should be treated as an explicit component feature rather than an assumed design-system rule.

---

## 15. Lists

Lists are lightweight structured output.

Semantic roles:

```text
item       → list.item
enumerator → list.enumerator
```

Enumeration shape and styling are separate concerns.

Changing the glyph or enumeration strategy should not require changing the semantic text role.

A list must remain understandable when styling is stripped.

---

## 16. Trees

Trees are used for hierarchical output.

The default tree uses familiar box-drawing connections such as:

```text
root
├── child-a
│   └── nested
└── child-b
```

Nabat also supports a rounded enumerator style.

Semantic roles:

```text
tree item       → tree.item
tree enumerator → tree.enumerator
```

Connector geometry and color are independent.

Tree structure must remain legible in plain text.

---

## 17. Spinners

`Context.Spinner` represents a single active operation.

### 17.1 Behavior

On an interactive terminal:

- the operation may animate;
- the active frame uses `spinner.active`;
- the title uses the informational semantic style;
- completion resolves to an appropriate static semantic icon.

On non-TTY output:

- no animation is emitted;
- the title is printed plainly;
- the operation runs normally.

### 17.2 Delayed Animation

The default spinner intentionally waits briefly before opening an animated interface.

Fast operations therefore produce a static completion line rather than flashing a spinner for a fraction of a second.

This is a UX principle worth preserving:

> Do not animate work that completes too quickly for animation to help.

### 17.3 Default Spinner

The default animation is a compact dot/Braille-style spinner.

Alternative spinner shapes are implementation options rather than brand identity.

---

## 18. Multi-Row Status

`Context.Status` handles concurrent or multi-step work.

Conceptually:

```text
⠋ Deploying

    OBJECT      REASON       AGE
 ✓  api          ready        4s
 !  worker       retrying     2s
 ⠋  database     migrating    1s
```

### 18.1 Interactive TTY

On a TTY, Status may:

- animate active rows;
- update rows in place;
- display optional columns;
- prioritize rows;
- show a completion state;
- adapt to terminal height.

### 18.2 Non-TTY

In non-interactive environments:

- animation disappears;
- the title is plain text;
- the final state is rendered as a stable text table.

Logs should remain readable after the process ends.

### 18.3 State Communication

Completed rows use semantic state styling.

Active state uses:

```text
status.active
```

State must never depend on color alone.

Static glyphs remain part of the information architecture.

---

## 19. Interactive Prompts

Nabat's prompt styling is built around a small semantic surface rather than exposing every low-level field of the underlying prompt library.

Important prompt roles include:

```text
title
description
cursor
placeholder

selected option
unselected option

selected prefix
unselected prefix

error
help
selector

focused button
blurred button

border
border color
```

### 19.1 Default Semantic Mapping

When a prompt style is derived from tokens, the intended relationships are:

```text
Title            → text.title
Description      → text.secondary

Cursor           → status.info
Placeholder      → text.muted

SelectedOption   → status.success
UnselectedOption → text.primary

SelectedPrefix   → status.success
UnselectedPrefix → text.muted

Error            → status.error
Help             → accent.primary
Selector         → status.info

ButtonFocused    → accent.primary
ButtonBlurred    → text.muted

BorderColor      → accent.primary
```

### 19.2 Selection

Selection should be communicated through more than color.

Nabat's brand theme uses explicit selected/unselected markers so a user can identify state in plain text.

### 19.3 Focus

Focus may be represented through:

- accent color;
- border visibility;
- selector glyphs;
- cursor position.

A color change alone is insufficient.

### 19.4 Borders

Rounded prompt borders fit the Nabat visual language and may be used by the brand theme.

They are a prompt presentation choice rather than a universal requirement for every Nabat output component.

---

## 20. Help Output

Nabat's help renderer should feel structured, calm, and scannable.

Its semantic hierarchy is based on roles rather than raw palette colors.

Typical mapping:

```text
command title       → text.title
section heading     → accent.primary
body / descriptions → text.secondary
muted metadata      → text.muted
warnings            → status.warning
```

For usage expressions:

```text
command path  → text.title
[flags]       → text.muted
[command]     → text.muted
<required>    → accent.primary
[optional]    → text.secondary
```

Help should prioritize comprehension over decoration.

Alignment, spacing, and grouping should do most of the visual work.

---

## 21. Typography

### 21.1 Actual CLI

Nabat does not control the user's terminal font.

The CLI must inherit whatever monospace font the terminal environment provides.

Therefore:

> No Nabat CLI behavior may depend on JetBrains Mono, Fira Code, Nerd Fonts, ligatures, or any other specific installed font.

Standard Unicode symbols may be used when they have reliable terminal support, with plain-text degradation where necessary.

### 21.2 Documentation and Screenshots

For controlled Nabat-owned surfaces, the preferred monospace stack is:

```css
"JetBrains Mono",
"Fira Code",
"Cascadia Code",
"SF Mono",
Menlo,
Consolas,
monospace
```

JetBrains Mono is the preferred first choice for:

- documentation examples;
- terminal mockups;
- code screenshots;
- diagrams containing code or CLI output.

### 21.3 Website Interface Typography

For documentation and web interface text:

```css
Inter,
-apple-system,
BlinkMacSystemFont,
"Segoe UI",
Roboto,
sans-serif
```

The website may combine:

- sans-serif for long-form reading;
- monospace for commands, code, tokens, labels, and technical highlights.

Typography recommendations apply only to surfaces Nabat controls.

---

## 22. Logo Direction — "The Crystal Thread"

This section defines the visual direction for the Nabat identity.

Until canonical logo assets are committed, it should be interpreted as a brand specification rather than a pixel-perfect asset contract.

### 22.1 Core Elements

The logo language should combine:

1. a thread;
2. a CLI prompt or directional mark;
3. a growing geometric crystal.

### 22.2 The Thread

The thread represents Cobra.

Recommended characteristics:

- vertical or structurally directional;
- thin;
- restrained;
- optionally dashed;
- turquoise-oriented within the Nabat Palette.

It should feel structural rather than decorative.

### 22.3 CLI Chevron

A `>` form may be integrated into the thread or crystal.

It represents the command-line interface without relying on a generic terminal-window icon.

The chevron should remain recognizable at small sizes.

### 22.4 Crystal

The crystal is the primary Nabat element.

Recommended characteristics:

- geometric;
- faceted;
- sharp rather than bubbly;
- visually stable at small sizes;
- saffron/gold dominant;
- capable of monochrome reproduction.

The form should suggest growth around the thread.

### 22.5 Highlights and Satellite Crystals

Larger illustrations may include:

- warm cream facet highlights;
- small diamond glints;
- restrained mint or pistachio satellite fragments;
- secondary crystals along the thread.

These elements should disappear in small logo variants.

### 22.6 Required Logo Variants

The identity should eventually provide at least:

#### Full Brand Illustration

Crystal, thread, secondary fragments, and environmental detail.

Use for:

- website hero areas;
- posters;
- conference slides;
- social graphics.

#### Primary Mark

Simplified crystal plus thread or CLI chevron.

Use for:

- README;
- project page;
- documentation navigation;
- social avatar.

#### Small Mark

Minimal crystal/chevron geometry.

Use for:

- favicons;
- tiny avatars;
- compact navigation.

#### Monochrome Mark

Single-color version with no dependency on palette gradients or multiple fills.

Use where color is unavailable.

#### Wordmark

Primary mark plus the name:

```text
Nabat
```

The wordmark should preserve enough whitespace to keep the crystal geometry distinct.

### 22.7 Logo Restraints

Avoid:

- generic terminal-window logos;
- a Go gopher derivative as the primary mark;
- excessive Persian ornamental patterns;
- photorealistic rock candy;
- gradients required for recognition;
- detail that disappears below icon scale.

The cultural reference should come primarily from the concept, palette, and material language rather than decorative motifs.

---

## 23. Broader Visual Language

The crystal metaphor should extend beyond the logo without turning every component into a crystal.

Recommended motifs include:

- faceted corner cuts;
- angular separators;
- small diamond indicators;
- dashed thread-like guides;
- vertical connective lines;
- asymmetric crystal clusters;
- restrained refraction/glint details.

These are especially suitable for:

- documentation diagrams;
- section dividers;
- hero illustrations;
- feature callouts;
- architecture visualizations;
- social media assets.

Avoid defaulting to a generic SaaS identity made entirely from:

- rounded cards;
- soft gradients;
- floating glass panels;
- oversized pills.

Rounded UI may still be used where functionally appropriate, particularly for prompts and web controls, but it should not become the main visual metaphor.

---

## 24. Website Design Direction

The future `nabat.dev` website should express the same system as the CLI without pretending the browser is a terminal.

### 24.1 Relationship to the CLI

The website may be richer than the terminal, but it should preserve:

- the Nabat Palette;
- Crystal / Saffron / Garden hierarchy;
- semantic status meanings;
- crystal geometry;
- thread motifs;
- terminal examples;
- technical restraint.

### 24.2 Dark and Light

The website should support both Nabat Dark and Nabat Light.

Theme values should come from the canonical Palette rather than copied hex literals maintained independently in the website source.

### 24.3 Semantic Web Tokens

The website should introduce a semantic layer such as:

```text
background.canvas
background.surface

text.primary
text.secondary
text.muted
text.title
text.link

border.default

brand.primary
brand.secondary

status.success
status.warning
status.error
status.info

interaction.focus
interaction.selected
interaction.disabled
```

Those semantic roles should resolve to Nabat Palette values.

Components should consume semantic variables rather than raw palette names.

For example:

```css
.button {
  color: var(--nabat-text-primary);
}

.status-success {
  color: var(--nabat-status-success);
}
```

Prefer:

```text
--nabat-status-success
```

over coupling components directly to:

```text
--nabat-pistachio
```

### 24.4 Generated Tokens

Where practical, web artifacts should be generated from the machine-readable Palette.

Desired direction:

```text
nabat-dev/palette
       │
       ├── terminal ports
       ├── editor ports
       ├── CSS variables
       └── website theme data
```

This prevents the website from becoming another palette authority.

---

## 25. Accessibility

Accessibility requirements apply across terminal, documentation, and web surfaces.

### 25.1 Never Use Color Alone

A success state must still look like success without green.

An error must still look like an error without red.

Use symbols and text alongside color.

### 25.2 Plain-Text Equivalence

A user viewing:

- redirected output;
- CI logs;
- `NO_COLOR`;
- `TERM=dumb`;
- a limited terminal;

must still receive meaningful output.

### 25.3 Contrast

Palette ports and website mappings should validate appropriate foreground/background contrast for their target environment.

Dark and light variants may use different color values specifically to preserve semantic identity while maintaining readability.

### 25.4 Animation

Animation should communicate active work, not decorate idle states.

Short operations should avoid unnecessary animation.

Non-interactive environments should receive stable output rather than animation escape sequences.

### 25.5 Unicode

Unicode symbols are welcome where they provide useful semantic structure.

However, the system should not depend on private glyph sets or Nerd Font icons for essential information.

---

## 26. Design Rules for New CLI Components

When adding a new user-facing component to Nabat, use the following sequence.

### Step 1 — Define Meaning

Determine the component's semantic roles before choosing colors.

Ask:

```text
What information does this component communicate?
```

not:

```text
Which Nabat color looks good here?
```

### Step 2 — Reuse Existing Tokens

Prefer existing tokens where their meaning fits.

For example:

```text
status.success
text.muted
accent.primary
```

### Step 3 — Add a Component Token Only When Needed

If the component needs independent theming, introduce a component role such as:

```text
component.part
```

and provide a sensible alias to an existing semantic token.

Avoid introducing a new raw color for every component.

### Step 4 — Define Plain Behavior

Specify what the component looks like:

- without ANSI;
- outside a TTY;
- under `NO_COLOR`;
- in logs.

### Step 5 — Define Stream Ownership

Explicitly determine whether the component is:

```text
command product → stdout
```

or:

```text
human status / diagnostics → stderr
```

Do not choose a stream based merely on convenience.

### Step 6 — Avoid Color-Only State

Add text, structure, glyphs, or other non-color state cues.

### Step 7 — Test Narrow and Non-Interactive Environments

Terminal UX is incomplete until the component remains useful outside an ideal interactive terminal.

---

## 27. Design Governance

Changes should be made in the repository that owns the concept.

### Change a Raw Color

Change:

```text
nabat-dev/palette
```

Then update affected implementations or generated artifacts.

Do not start by editing this file.

### Change a Semantic Theme Role

Change:

```text
nabat-dev/nabat/theme
```

Update theme documentation and this design contract if the conceptual model changes.

### Change CLI Component Behavior

Change:

```text
nabat-dev/nabat
```

Update this document when the change affects the user-facing design contract.

### Change API Philosophy

Change:

```text
docs/design-principles.md
```

Do not duplicate the full engineering rationale here.

### Change Brand or Visual Direction

Change:

```text
DESIGN.md
```

and update official assets accordingly.

### Change Website Presentation

Change the website implementation while keeping it consistent with:

```text
DESIGN.md
+
nabat-dev/palette
```

---

## 28. Current vs Proposed Design

This document distinguishes between two categories.

### Current Contract

A rule is current when it describes behavior already represented by Nabat or its theme architecture.

Examples include:

- stdout/stderr separation;
- semantic status tokens;
- token aliases;
- status symbols;
- table/list/tree tokenization;
- TTY-aware behavior;
- prompt semantic roles;
- spinner and status degradation.

### Brand Direction

A rule may describe an intended Nabat-owned visual surface that is not yet represented by a committed implementation.

Examples may include:

- final logo geometry;
- wordmark construction;
- website layouts;
- documentation visual motifs;
- generated CSS token naming.

Such guidance must not be presented as an existing API or implementation feature.

When a proposed design becomes implemented and stable, this document should be updated to remove the distinction.

---

## 29. Design System Summary

The complete Nabat design architecture can be viewed as:

```text
                         NABAT

                     Brand Philosophy
                           │
          ┌────────────────┴────────────────┐
          │                                 │
     Visual Language                  Terminal UX
          │                                 │
   Crystal / Thread                   UNIX-friendly
   Persian Nabat                      stdout / stderr
   Warm identity                     TTY awareness
          │                           plain fallback
          │                                 │
          └───────────────┬─────────────────┘
                          │
                          ▼
                     Nabat Palette
                Crystal / Saffron / Garden
                          │
                          ▼
                    Theme Primitives
                          │
                          ▼
                    Semantic Tokens
                ┌─────────┼──────────┐
                │         │          │
              status.*   text.*   accent.*
                │
                ▼
                  Component Tokens
        ┌─────────┼────────┬───────────┐
        │         │        │           │
      table.*   list.*   tree.*   spinner/status
        │         │        │           │
        └─────────┴────────┴───────────┘
                          │
                          ▼
                       Renderers
                          │
             ┌────────────┴────────────┐
             │                         │
           stdout                    stderr
       command product          human interaction
```

The essential rule is:

> **Palette defines color. Tokens define meaning. Components consume meaning. The terminal environment determines presentation.**

---

## 30. Final Principles

When making a design decision for Nabat, prefer the choice that preserves these properties:

1. **Warm, recognizable identity without unnecessary decoration.**
2. **Shared semantics across dark and light variants.**
3. **Standard CLI meanings over invented conventions.**
4. **Clean machine-readable stdout.**
5. **Human status and interaction on stderr.**
6. **Useful plain-text behavior before styled behavior.**
7. **Semantic tokens instead of direct color coupling.**
8. **Meaning that survives without color.**
9. **Terminal-native behavior rather than web UI imitation.**
10. **One canonical source for every kind of design data.**

Nabat should feel like its namesake: a small, solid structure that grows useful layers around a dependable thread.
