# Design Decisions

This document records **stable** design choices behind Nabat: the trade-offs and
constraints that explain how the API and runtime behave today. It is not a
changelog; API evolution and removals live in the appendix when they still help
readers map old examples to the current model.

[Design Principles](design-principles.md) states the values Nabat optimizes for.
[Architecture](architecture.md) describes structure and execution flow. This file
answers **why** specific forks were taken—so maintainers and advanced users can
tell intentional constraints from accidents.

Nabat is a Go library for building command-line tools on top of
[Cobra](https://github.com/spf13/cobra). It adds typed options, adaptive
positional arguments, themed output, interactive prompts, and an extension
system.

## Contents

### Foundational decisions

- [Cobra is the engine](#cobra-is-the-engine)
- [The core stays small; extensions add features](#the-core-stays-small-extensions-add-features)
- [Extensions cannot change existing commands](#extensions-cannot-change-existing-commands)
- [Help, version, and completion live in the core](#help-version-and-completion-live-in-the-core)
- [Functional options with private config](#functional-options-with-private-config)
- [One option type per target](#one-option-type-per-target)
- [`Option` is an interface, not a function type](#option-is-an-interface-not-a-function-type)
- [Typed fields, defaults, and compile-time kind safety](#typed-fields-defaults-and-compile-time-kind-safety)
- [Args and flags are different](#args-and-flags-are-different)
- [Adaptive arg resolution](#adaptive-arg-resolution)
- [`defaultVal` is the only non-interactive fallback for declarative args](#defaultval-is-the-only-non-interactive-fallback-for-declarative-args)
- [Environment variables: `WithEnv` and `WithEnvAlias`](#environment-variables-withenv-and-withenvalias)
- [Declarative args vs ad-hoc prompts](#declarative-args-vs-ad-hoc-prompts)
- [Two surfaces for registration errors](#two-surfaces-for-registration-errors)
- [Root and subcommands share one definition surface](#root-and-subcommands-share-one-definition-surface)
- [Lifecycle hooks receive `*Context`](#lifecycle-hooks-receive-context)
- [`*Context` implements `context.Context`](#context-implements-contextcontext)
- [One `IOStreams` bundle for input, output, and errors](#one-iostreams-bundle-for-input-output-and-errors)
- [Terminal-aware output by default](#terminal-aware-output-by-default)
- [Themes are built in and customizable](#themes-are-built-in-and-customizable)
- [Theme manifests are validated by a public JSON Schema](#theme-manifests-are-validated-by-a-public-json-schema)
- [Flat public package with `internal/` helpers](#flat-public-package-with-internal-helpers)

### Supporting topics

- [Help heading wording: "Global Flags" vs "Inherited Flags"](#help-heading-wording-global-flags-vs-inherited-flags)
- [Spinner takes `func() error`](#spinner-takes-func-error)

### Reference

- [Appendix: Historical API notes](#appendix-historical-api-notes)

## Cobra is the engine

Nabat wraps Cobra. Cobra handles the command tree, flag parsing, and shell
completion scripts. Nabat adds the layers above: typed options, args, output,
prompts, themes, and extensions.

Nabat uses Cobra because:

- Cobra is the most common CLI library in Go. Many teams and tools already know
  it.
- Flag parsing and completion scripts are hard to get right. Cobra has solved
  them well.
- Wrapping a stable engine lets Nabat focus on what is new instead of rebuilding
  the basics.

Nabat also delegates flag deprecation to
[pflag](https://github.com/spf13/pflag). `WithDeprecated` and
`WithDeprecatedShorthand` call pflag's `MarkDeprecated` and
`MarkShorthandDeprecated`. Nabat's help renderer reads the `Deprecated` text
from pflag and shows it next to the flag, so the flag stays visible with a
clear `(deprecated: …)` note instead of disappearing from help.

## The core stays small; extensions add features

The `nabat` package contains only the parts every CLI needs: the app, commands,
args, flags, output, prompts, themes, help, version, and shell completion.
Optional features live in separate subpackages (`manpage`, `logging`). The core
does not import these subpackages.

You install an extension by passing `WithExtension(ext)` to `New`:

```go
app := nabat.MustNew("myctl",
    nabat.WithCompletion(),
    nabat.WithExtension(manpage.New()),
)
```

An extension is any value that satisfies the `nabat.Extension` interface
(`fmt.Stringer` plus `Init(AppSurface) error`). The `Init` method runs inside `New`
and uses the `AppSurface` accessors (`Command`, `OnPreRun`, `SetLogger`,
`Theme`, `IO`, `Name`, `EnvPrefix`, `UnsafeRoot`) to register subcommands or
hooks. First-party and third-party extensions use the same interface and the
same accessors, so they are interchangeable.

Use `AsExtension` to wrap an inline `func(AppSurface) error` as a named extension without a
separate package type; `Init` runs in `WithExtension` declaration order with other
extensions, after help, version, and completion are registered on the root command.

The split keeps the core sealed and easy to learn:

- The dependency arrow points outward: extensions import `nabat`, never the
  other way around.
- The `nabat.With...` autocomplete list shows a small set of verbs. Each
  extension adds its own options under its own package name.
- A bare `nabat.MustNew("myctl")` produces a working CLI with help. Anything
  else is opt-in, so there is no hidden behavior.

Theme is the one piece of data that stays in the core but is not a subcommand.
Every output path reads it (help rendering, structured output, extension loggers
via `App.Theme()`), so it must be available before any extension runs.

## Extensions cannot change existing commands

Extensions install new subcommands (with `App.Command`) and global hooks (with
`App.OnPreRun`), and may set the logger (with `App.SetLogger`). They must not
change the root command or any command they did not create.

This rule exists because:

- If two extensions can change the same command, the result depends on install
  order. Users cannot tell from the call site which extension wins, so bugs are
  hard to find.
- If an extension adds a flag to a user command, the user's source code does not
  explain where the flag came from. The CLI surface stops matching what the user
  wrote.
- A global hook covers most needs:

  ```go
  app.OnPreRun(func(c *nabat.Context) error {
      return checkAuth(c)
  })
  ```

  Auth checks, audit logs, telemetry, and dry-run gating all fit this shape. The
  hook fires before every command, and the handler decides what to do with each
  invocation.

If a future feature truly needs to add a root-level flag, the right place for it
is the core (see help and version below), not an extension.

Trade-off: some Cobra patterns, such as a third-party library that injects a
flag into another command, do not work in Nabat. They become a hook with a
filter instead.

## Help, version, and completion live in the core

Help, version, and shell completion are the three subsystems that ship inside
the core. Help and version each register a root flag and a subcommand;
completion registers a subcommand and re-uses the dynamic-completer hooks that
already live on every flag and arg. All three use the same install pipeline
inside `New`.

### Help

Help is split along two axes. The persistent `--help` / `-h` flag is on by
default for every `nabat.New` call (GNU/POSIX convention; this is what users
expect from any CLI). The `help <subcmd>` subcommand is opt-in via
`WithHelpCommand`, mirroring how `WithVersion` opts into the version feature.
Nabat renders help with its own themed renderer in both cases.

You can rename the flag with `WithHelpFlagName` and `WithHelpShorthand`, drop a
single piece with `WithoutHelpFlag` or `WithoutHelpShorthand`, and rename the
opt-in subcommand with `WithHelpCommand(WithHelpCommandName("aide"))`.
`WithoutHelp` turns the whole feature off (Cobra's default help takes over).

Help lives in the core because:

- A new CLI should print useful help right away. Users expect `--help` to work
  without setup.
- The renderer reads private state: command metadata, theme styles, and the IO
  bundle. Moving the renderer out of the package would force Nabat to expose all
  that state.
- Other Go CLI libraries (Cobra, Click, urfave/cli, clap) keep help in the core
  for the same reasons.

### Help heading wording: "Global Flags" vs "Inherited Flags"

The renderer prints `Global Flags:` (not `Inherited Flags:`) for the section
that lists flags inherited from parent commands. Two reasons:

- It matches Cobra's default help template and Docker's CLI, the two reference
  points most developers encounter first. `gh` and `kubectl` use "inherited" /
  "parent options inherited", so both terms are well-attested in the wild — we
  pick the one that aligns with the engine we sit on top of.
- "Inherited" is implementation jargon that describes how the flag got there ("a
  parent declared it; we inherited it"). From the user's perspective it is
  simply a flag that works everywhere under the binary, which is exactly what
  "global" conveys.

Code-side identifiers use the upstream API's wording: `cmd.InheritedFlags()`,
`cmd.HasAvailableInheritedFlags()`, `WithPersistent`, and Go test names like
`TestInheritedFlagsRenderShorthand` use the "inherited" terminology because they
describe the mechanism, not the rendered label. Implementation jargon belongs in
code; user-facing wording belongs in help output.

### Version

Version is opt-in. Pass `WithVersion(s, ...VersionOption)` to install a
`version` subcommand and a `--version` / `-v` flag with themed output. Omit it
and nothing is installed.

Version lives in the core because only the core can register a root-level flag.
The rule above forbids extensions from doing it. Help and version are the only
features that need this, so they share the same install pipeline.

### Completion

Completion is opt-in. Pass `WithCompletion(...CompletionOption)` to install a
`completion` subcommand with bash, zsh, fish, and PowerShell generators. Omit
it and nothing is installed. Per-flag and per-positional dynamic completers are
attached with `WithCompleter` and `WithPositionalCompleter`; they always work
because Cobra's hidden `__complete` command is part of every Cobra binary.

Completion lives in the core for four reasons:

- The Cobra completion machinery (`__complete`, `GenBashCompletion`, etc.) is
  already linked into every binary that imports `cobra`. Putting completion in
  an extension subpackage adds package-level ceremony around code that is
  always present.
- The dynamic-completer hooks (`WithCompleter`, `WithPositionalCompleter`)
  attach to flag specs and arg specs that the core already owns. Splitting
  the install surface (subcommand) from the per-field surface (completers)
  across two packages would force one of two awkward choices: a circular
  import, or a parallel `nabat.WithCompleter` / `completion.WithSomething`
  naming collision.
- Shell completion is a baseline CLI affordance per [clig.dev](https://clig.dev/)
  ("Make it easy to install your CLI" / "Help users get unstuck"). Treating it
  as discoverable infrastructure — sitting next to `WithVersion` in autocomplete
  — matches user expectation better than hiding it behind a separate package.
- The same option lattice as `WithVersion` (one app-level switch with nested
  `*Option` polish) keeps the cognitive model consistent. Users who already know
  `WithVersion` learn `WithCompletion` for free.

The naming split between `WithCompletion` (the feature switch) and
`WithCompleter` / `WithPositionalCompleter` (the per-field callbacks) is
deliberate: a "completion" is the user-facing feature; a "completer" is a
single function that returns candidates for one flag or arg. Distinct nouns
prevent the surface from collapsing into a single overloaded `WithCompletion`
identifier with three different signatures.

### Different option shapes

Help mixes the two shapes because its two surfaces have different defaults. Flag
tweaks are app-level options (`WithHelpFlagName`, `WithHelpShorthand`,
`WithoutHelpFlag`, `WithoutHelpShorthand`, `WithoutHelp`). The opt-in subcommand
uses the nested-option pattern (`WithHelpCommand(WithHelpCommandName("aide"))`)
just like `WithVersion`. The shapes differ because:

- The flag is on by default and gets tweaked. Independent options compose well
  for tweaks, and forcing them into a `WithHelp(...)` wrapper would mislead
  developers into thinking the feature is opt-in when it is not.
- The subcommand is off by default. You enable and configure it in one call, so
  nested options fit, and conflicts among its sub-options are easy to validate
  inside the constructor (same reason `WithVersion` does it this way).

## Functional options with private config

Every configuration surface uses `func(*spec) error` options. The `Option`,
`CommandOption`, `ArgOption`, `FlagOption`, and per-prompt option families all
follow this pattern. The options write to a private `config` (or `*spec`) struct
that the constructor validates before it builds the public type.

```go
type Option func(*config) error

func New(name string, opts ...Option) (*App, error) {
    cfg := defaultConfig()
    cfg.name = name
    for _, opt := range opts {
        if err := opt(cfg); err != nil {
            errs.Add(err)
        }
    }
    if err := cfg.validate(); err != nil {
        return nil, err
    }
    // build App from validated config
}
```

This pattern works well because:

- Adding a new option does not change any existing call site. A new struct field
  would force every caller to read the change.
- Callers only set the values they care about. The constructor fills the
  defaults, so there is no "zero value means default" question.
- Each option can reject invalid input as it runs (for example,
  `WithLogger(nil)` returns an error). Errors point straight at the bad option.
- Once `New` returns, the config is sealed. Nothing outside the package can
  change it.
- The public surface stays small: callers see option functions and a few public
  types, not the internal fields. Internals can change without breaking users.

Trade-off: functional options are harder to list in one place than struct
fields. This is fine for a CLI library because the config is set once at
startup, not serialized or diffed at runtime.

## One option type per target

Nabat splits options by target:

| Type            | Used with                                                |
|-----------------|----------------------------------------------------------|
| `Option`        | `New(...)` (the app)                                     |
| `RootOption`    | Subset of `CommandOption` that is also a valid `Option`  |
| `CommandOption` | `Command(...)` and `WithCommand(...)`                    |
| `ArgOption`     | `WithArg`, `WithSelectArg`, ...                          |
| `FlagOption`    | `WithFlag`, `WithStringFlag`, ...                        |

The three command-related types form a lattice. Every `RootOption` value also
satisfies `Option` (so it can be passed directly to `nabat.New(...)` to
configure the root command) and `CommandOption` (so it can be passed to any
subcommand registration). A short list of `CommandOption` constructors
(`WithGroup`, `WithHidden`, `WithAliases`, `WithTypoHints`) are not valid on
the root command and so are not `RootOption`s; passing them to
`nabat.New(...)` directly is a build error. They remain valid inside
`nabat.WithCommand("deploy", ...)` because the inner argument list accepts
the wider `CommandOption` type. See
[Root and subcommands share one definition surface](#root-and-subcommands-share-one-definition-surface)
for the rationale per option.

For values that both args and flags care about, Nabat ships **one** helper
that returns an anonymous interface satisfying both target types:
`WithRequired() interface { ArgOption; FlagOption }`,
`WithUsage(text) interface { ArgOption; FlagOption }`,
`WithEnv(...) interface { ArgOption; FlagOption }`,
`WithEnvAlias(...) interface { ArgOption; FlagOption }`. The same call works
on positional args and flags, with no parallel `*Flag` mirror to remember.
`WithHidden()` and `WithDeprecated(message, ...)` use the same trick for
their command/flag overlap, and the deliberately-empty third arm
(`ArgOption`) makes positional misuse a compile-time error.

The split works this way because:

- The compiler catches mistakes. You cannot pass an `ArgOption` where a
  `FlagOption` is expected, use a flag-only option (`WithShort`) on an arg,
  or pass a subcommand-only option (`WithGroup`) directly to `New`.
- Each call site has a focused autocomplete list. Inside `New(...)` you see
  app options plus root-applying options plus `WithCommand`. Inside
  `WithFlag(...)` you see flag options.
- All five families (`Option`, `RootOption`, `CommandOption`, `ArgOption`,
  `FlagOption`) are interfaces so a single value can satisfy more than one
  slot when that combination is meaningful. Each `apply...` method carries
  an `error` return so options that take callbacks (`WithValidation`) or
  constrained values still reject bad input at apply time.
  See [`Option` is an interface, not a function type](#option-is-an-interface-not-a-function-type).

The cost is the anonymous interface in the return type of every shared
helper. The win is that every misuse becomes a build error rather than a
runtime check. Details on shared helpers appear under [Single exported name per concept](#single-exported-name-per-concept).

### Single exported name per concept

Where an option applies to more than one target (positional args, flags, or
commands), Nabat exposes **one** exported helper. Its return type is an
anonymous interface listing each relevant option family, so one call works at
every valid site without parallel `With*` / `With*Flag` names.

```go
func WithRequired() interface { ArgOption; FlagOption }
func WithUsage(text string) interface { ArgOption; FlagOption }
func WithEnv(names ...string) interface { ArgOption; FlagOption }
func WithEnvAlias(names ...string) interface { ArgOption; FlagOption }
func WithHidden() interface { CommandOption; FlagOption }
func WithDeprecated(message string, sub ...DeprecationOption) interface {
    CommandOption
    FlagOption
}
```

This works because:

- The Go type system supports anonymous interfaces in return types. The same
  value satisfies multiple option families when the combination is meaningful.
- Misuse is a compile-time error. `WithDeprecated(...)` implements
  `CommandOption` and `FlagOption` but not `ArgOption` because cobra/pflag has
  no deprecation hook for positional args; using it inside `WithArg(...)` is
  a build error, not a silent no-op. `WithHidden()` omits `ArgOption` because
  hiding a positional arg does not change that the slot still consumes the next
  CLI token.
- `WithDeprecatedShorthand` is a top-level `FlagOption` rather than a nested
  `DeprecationOption` because it depends on `WithShort` on the same flag;
  keeping it at the top level preserves the "shorthand only" constraint in
  the type system.

Internally, unification rests on a small carrier type, `sharedFieldOpt`, that
targets the embedded `fieldConfig` shared by `argSpec` and `flagSpec`, plus
per-target adapters (`hiddenOpt`, `deprecatedOpt`) for options that straddle
commands and flags.

For the broader principle, see
[Make Wrong Code Unrepresentable](design-principles.md#7-make-wrong-code-unrepresentable).

## `Option` is an interface, not a function type

`Option` is a Go interface with one method:

```go
type Option interface {
    applyToConfig(*config) error
}
```

The internal `optionFn` adapter wraps a `func(*config) error` to satisfy it,
so app-level constructors stay a one-line wrap
(`return optionFn(func(c *config) error { ... })`). This is invisible to users.

An interface—not a bare function type—for app-level options lets one value
satisfy multiple option interfaces at once:

- A `RootOption` value (e.g. `nabat.WithDescription`) also implements `Option`,
  so it works directly in `nabat.New(...)` and applies to the root command.
- A `nabat.WithCommand(name, opts...)` value implements `Option`, `RootOption`,
  and `CommandOption`, so the same call works at every nesting level: top-level
  under `New`, or nested inside another `WithCommand`.

With `Option` as a function type, a value could only ever satisfy one
position. The interface form lets the type system express "this value is valid
in any of these slots" without separate constructors per slot.

`CommandOption` and `RootOption` already used interfaces; `Option` matches them.
The cost is one internal adapter type. The win is the lattice in
[Architecture — Key Types](architecture.md#key-types): every value applies only
in slots its semantics permit.

## Typed fields, defaults, and compile-time kind safety

### `FieldOption[T]` and typed phantom seals

All prompt and form options share the generic type `FieldOption[T]`, where `T`
is the bind type at the call site (string, bool, int, time.Duration, a typed
enum, etc.). Compile-time kind safety uses **typed phantom seals**: each
concrete option struct implements the `fieldOpt(T)` method only for the `T`
values it allows.

| Helper              | Valid kinds                                        | How `T` is fixed          |
|---------------------|----------------------------------------------------|---------------------------|
| `WithHint`          | string, int, int64, uint, float64, Duration        | Inferred from value `v T` |
| `WithMaxChars`      | string only                                        | Concrete struct (`fieldOpt(string)`) |
| `WithPassword`      | string only                                        | Concrete struct |
| `WithSuggestions`   | string only                                        | Concrete struct |
| `WithMultiline`     | string only                                        | Concrete struct |
| `WithFilePicker`    | string only                                        | Concrete struct |
| `WithEditor`        | string only (implies multiline)                    | Concrete struct |
| `WithEditorCmd`     | string only                                        | Concrete struct |
| `WithAllowedTypes`  | string only                                        | Concrete struct |
| `WithShowHidden`    | string only (requires `WithFilePicker`)            | Concrete struct |
| `WithShowSize`      | string only (requires `WithFilePicker`)            | Concrete struct |
| `WithShowPermissions`| string only (requires `WithFilePicker`)           | Concrete struct |
| `WithAffirmative`   | bool only                                          | Concrete struct (`fieldOpt(bool)`) |
| `WithNegative`      | bool only                                          | Concrete struct |
| `WithDefault`       | any (all kinds)                                    | Inferred from value `v T` |
| `WithValidate`      | any (all kinds)                                    | Inferred from `fn func(T) error` |

**How it works:**

The `FieldOption[T]` interface requires a `fieldOpt(T)` phantom method.
Single-kind helpers (e.g. `WithAffirmative`) return a concrete struct with
`fieldOpt(bool)` — passing it where `FieldOption[string]` is expected is a build
error. Multi-kind helpers like `WithDefault[T any](v T)` carry `T` in their
value argument, so Go infers `T` from the literal without an explicit
annotation.

For string fields, widget mode is selected with explicit options such as
`WithMultiline()` and `WithFilePicker()` on ordinary string bindings; sub-options
like `WithEditor()` remain compile-time safe because they only apply to string
fields.

**Select and MultiSelect:**

`Select[E]` and `MultiSelect[E]` are package-level generic functions (not methods)
because Go does not allow type parameters on methods. They take a typed
positional `defaultVal E` (not a `WithDefault` sub-option) because
`SelectOption` cannot be satisfied by `FieldOption[E]`. The positional parameter
keeps compile-time `E` checking: passing an `int` where a typed enum is expected
is a build error.

**`WithSelectField` / `WithMultiSelectField`:**

These constructors take a positional `defaultVal E` / `defaultVal []E` for the
same reason: the default is typed by `E`, and the positional slot preserves the
check at compile time. The name `defaultVal` matches `WithArg(name, defaultVal, ...)`.

### Defaults: `WithDefault` and positional `defaultVal`

Every "value when nothing else is provided" surface uses the word **default**
in a consistent way:

- **Sub-option:** `WithDefault[T any](v T)` for `FieldOption[T]` sites (form
  fields, ad-hoc prompts, arg prompt options). `T` is inferred from the value.
- **Positional:** `defaultVal E` for select constructors (`WithSelectField`,
  `WithMultiSelectField`, `nabat.Select`, `nabat.MultiSelect`) where a
  sub-option cannot carry the same type relationship.

Declarative args use only a positional `defaultVal` on `WithArg` / `WithSelectArg`
/ `WithMultiSelectArg`; see
[`defaultVal` is the only non-interactive fallback for declarative args](#defaultval-is-the-only-non-interactive-fallback-for-declarative-args).

### Form fields implement both `FormOption` and `GroupOption`

Form fields satisfy both `FormOption` and `GroupOption` — the same dual-interface
idea as `WithRequired() interface{ ArgOption; FlagOption }`. One
`WithFormField` call can attach to a top-level `Context.Form` or sit inside
`WithFormGroup` for multi-page flows:

```go
nabat.WithFormField(&name, "Name", "", nabat.WithDefault(""))   // FormOption
nabat.WithFormGroup(nabat.WithFormField(&name, "Name", ""))       // GroupOption
```

Group chrome (`WithGroupTitle`, `WithGroupDescription`) differs from form chrome
(`WithFormTitle`, `WithFormDescription`) because in Huh, chrome lives on
`Group`s. **Chrome precedence:** form-level title/description fall through to the
first group only when that group has no explicit `WithGroupTitle` /
`WithGroupDescription`.

## Args and flags are different

Positional arguments ("args") and named options ("flags") are separate types
with separate rules.

- Args are positional and represent the main data a command works on. They go
  through the adaptive cascade (arg, env, prompt, default).
- Flags are named and represent options or modifiers. They use Cobra's flag
  system with an env-variable fallback.
- Args support declarative prompts via `WithPrompt`, because prompting
  for "what do you want to deploy?" makes sense. Flags do not, because
  prompting for `--verbose` would be strange.
- Flags support `WithShort` and `WithPersistent()`. Args do not, because
  positional values cannot have shorthands or be inherited by child commands.

The compiler enforces the split through the
[typed option system](#one-option-type-per-target). `WithPrompt` is only an
`ArgOption`, so passing it to `WithFlag` is a build error. Inside one
command, arg names and flag names share one namespace, so a name collision
between the two is reported at construction time.

## Adaptive arg resolution

Nabat resolves a positional arg through this cascade:

```text
arg -> env -> prompt -> default
```

The same command works in three modes without extra logic in the handler:

```go
nabat.WithSelectArg("environment", "", []string{"staging", "production"},
    nabat.WithRequired(),
    nabat.WithEnv("environment"),
    nabat.WithPrompt("Target environment", "",
        nabat.WithHint("staging"),
    ),
)

// myctl deploy production           -> arg
// MYCTL_ENVIRONMENT=staging myctl deploy -> env
// myctl deploy                      -> interactive prompt
```

The cascade works this way because:

- If the user passes a value, that value wins. There is no ambiguity.
- CI systems set environment variables. The same command runs in CI without
  arguments.
- In an interactive terminal, a missing value triggers a prompt instead of an
  error.
- If nothing else provides a value and the arg is not required, the default
  applies. If it is required and nothing matches, the error is clear and
  immediate.

For the exact rules, see
[Architecture — Arg Resolution Flow](architecture.md#arg-resolution-flow).

## `defaultVal` is the only non-interactive fallback for declarative args

Nabat declarative args have exactly one non-interactive fallback: the typed
`defaultVal` parameter passed to `WithArg`, `WithSelectArg`, and
`WithMultiSelectArg`. There is no separate `WithDefault` option on the arg level.

```go
nabat.WithArg("name", "world",                       // defaultVal = "world"
    nabat.WithPrompt("Your name", "",
        nabat.WithHint("alice"),
    ),
)

// myctl run alice                  -> "alice" (positional arg)
// MYCTL_NAME=bob myctl run         -> "bob"   (env)
// myctl run                        -> interactive prompt (TTY)
// myctl run </dev/null             -> "world" (defaultVal, non-interactive)
```

This works because:

- `WithArg[T]` is generic over the value type, so `defaultVal` is type-checked
  by the Go compiler. Passing `WithArg("count", "42", ...)` is a build error;
  the literal `42` would be required for an `int` arg. A separate
  untyped option could not enforce that, and a generic
  `WithDefault[T any](T)` could not enforce that the `T` matches the one
  inferred for the surrounding `WithArg[T]` (Go infers them independently),
  so a wrong value type only failed at registration time.
- One value to think about per arg, not two with subtle precedence rules.
- Interactive and non-interactive runs see the same final fallback when no
  prompt fires, so behavior is easier to reason about under `</dev/null`,
  pipes, and CI.

`WithRequired()` still fails when nothing comes from the arg, env, or prompt,
even if `defaultVal` is set. Defaults do not satisfy required positional args.

The `WithDefault[T any](v T)` helper is used on the **ad-hoc**
`Context.Input`, `Context.Confirm`, etc. surfaces (and on form fields via
`WithFormField`): those have no positional `defaultVal` parameter and need a
way to produce a value without prompting in non-interactive mode. Declarative
args have `defaultVal`, so they do not need a parallel mechanism.

This matches public CLI guidance such as
[clig.dev](https://clig.dev/#interactivity), which asks tools not to prompt
when stdin is not a TTY and to make the non-interactive path explicit.

## Environment variables: `WithEnv` and `WithEnvAlias`

Args and flags use two options to set their environment variable names:

- `WithEnv(name...)` adds names that combine with the app's env prefix.
  `nabat.WithEnv("token")` with prefix `MYAPP_` reads `MYAPP_TOKEN`.
- `WithEnvAlias(name...)` adds names that are read as-is, with no prefix.
  `nabat.WithEnvAlias("GITHUB_TOKEN")` reads `GITHUB_TOKEN`.

Both helpers return `interface { ArgOption; FlagOption }`, so the same call
works on positional args (`WithArg(...)`) and named flags (`WithFlag(...)`)
without parallel `*Flag` mirrors.

Two explicit options work well because:

- The intent is in the name. Reading the call tells you which variables the
  command checks.
- Order does not matter. You can write
  `WithEnv("token"), WithEnvAlias("GITHUB_TOKEN")` or the other way around with
  the same result.
- Empty names are dropped silently, so misuse is harmless.

For how env fits into the full resolution chain, see
[Adaptive arg resolution](#adaptive-arg-resolution).

## Declarative args vs ad-hoc prompts

Nabat keeps two separate interactive surfaces:

- **Declarative arg prompting** through `WithArg`, `WithSelectArg`, and the
  other `With*Arg` helpers, with the cascade `arg → env → prompt → default`.
- **Ad-hoc runtime prompts** through `Context` methods (`Confirm`, `Form`,
  `Spinner`, `Input`, `Select`, ...) inside the run function.

The split works this way because:

- Arg options describe how one named value is resolved. Runtime helpers support
  choices that depend on the current state of the command.
- Args prompt only when the value is missing. Flags never prompt and stay safe
  to use in scripts (`flag → env → default`).
- Some runtime interactions do not map to a single arg:
  - `Confirm` is a safety check before an action.
  - `Form` collects several values at once, often based on values resolved
    earlier.
  - `Spinner` shows progress around a long-running task.

This split keeps command definitions declarative and predictable while still
allowing rich interactive flows when needed.

## Two surfaces for registration errors

Nabat offers two equally honest surfaces for command registration:

- **Inline** (`(*Command, error)` from `App.Command` and `Command.Command`) for
  callers that want to handle each registration explicitly. This is the direct,
  idiomatic Go shape.
- **Aggregated** (`nabat.WithCommand(...)` passed to `nabat.New(...)`) for
  callers that want every option, validation, and registration error in one
  `*ConfigErrors` return value.

`MustCommand` is the panicking variant of the inline surface, intended for
`main()` and test setup where chaining
(`app.MustCommand("cluster").MustCommand("scale", ...)`) is more readable than
scoped `_, err := ...` blocks. It mirrors `MustNew` from the same package and
`regexp.MustCompile` from the standard library.

Nabat does **not** support a *deferred* error path: registration errors are
not stored on the `*App` for a later `Run`. They surface at `New` or at the
inline `Command` / `MustCommand` call. That matches principle 9 ("errors are
never silent") and means a successfully constructed app has a consistent
command tree.

```go
// Inline: each call is its own boundary.
_, err := app.Command("deploy",
    nabat.WithArg("env", ""),
    nabat.WithFlag("env", false),
)
require.ErrorIs(t, err, nabat.ErrArgFlagNameCollision)

// Must: chain freely; failure is a programmer bug.
cluster := app.MustCommand("cluster", nabat.WithDescription("Cluster management"))
cluster.MustCommand("scale", nabat.WithRun(scaleHandler))
cluster.MustCommand("status", nabat.WithRun(statusHandler))

// Aggregated: every problem in one *ConfigErrors return.
app, err := nabat.New("myctl",
    nabat.WithCommand("deploy",
        nabat.WithArg("env", ""),
        nabat.WithFlag("env", false),
    ),
    nabat.WithCommand("status", nabat.WithRun(statusHandler)),
)
```

`*ConfigErrors` implements `Unwrap() []error`, so callers can match individual
sentinel values (for example `errors.Is(err, nabat.ErrInvalidOption)`).
`Error()` formats the list as numbered items so the output stays readable.

The Cobra `AddCommand` precedent (also used by Gin, Echo, Chi, net/http) does
not justify a deferred path here, because Cobra does not validate at
registration time. Nabat does validate (option compatibility, name collisions,
flag-registration errors), so it must report at registration time.

## Root and subcommands share one definition surface

There is no separate scope for configuring the root command. Root options
(`WithDescription`, `WithFlag`, `WithRun`, etc.) pass directly to
`nabat.New(...)`. Subcommands declare via the same `nabat.WithCommand(...)`
value used at every nesting level.

```go
app, err := nabat.New("myctl",
    nabat.WithTheme(theme.Dracula),             // app-level Option
    nabat.WithVersion("1.0.0"),                  // app-level Option

    nabat.WithDescription("My CLI"),             // RootOption applied to root
    nabat.WithFlag("verbose", false,             // RootOption applied to root
        nabat.WithShort('v'), nabat.WithPersistent()),

    nabat.WithCommand("deploy",                  // subcommand under root
        nabat.WithRun(deployHandler),
    ),
    nabat.WithCommand("cluster",                 // subcommand under root
        nabat.WithDescription("Cluster management"),
        nabat.WithCommand("scale",               // nested subcommand
            nabat.WithRun(scaleHandler),
        ),
    ),
)
```

This unification matches what every modern CLI library has converged on:

- **Cobra** (Go): root and sub are the same `*cobra.Command` struct.
- **Kong** (Go, alecthomas): the root struct's fields are the root's flags and
  args; subcommands are nested struct fields. Single declarative expression.
- **urfave/cli v3** (Go): `cli.App` was deprecated in favor of `cli.Command`
  being both root and any subcommand. The root's `Commands` slice holds
  subcommands of the same type.
- **clap** (Rust): `Command::new("myapp").subcommand(Command::new("deploy"))`.
  Same `Command` type at every level.
- **Kingpin** (Go) is the cautionary counter-example: `*Application` for the
  root and `*CmdClause` for subcommands. The two-type split is widely cited as
  a reason teams moved off it.

There is no `WithRoot(...)` wrapper. Root options pass straight to `New`, and
the type system applies `RootOption` values to the root and `WithCommand` to
nested commands. The single-command case stays flat:

```go
app, err := nabat.New("encrypt",
    nabat.WithDescription("Encrypt a file"),
    nabat.WithArg("file", "", nabat.WithRequired()),
    nabat.WithFlag("output", "", nabat.WithShort('o')),
    nabat.WithRun(encryptFile),
)
```

The type system still enforces correctness. A short list of `CommandOption`
values are not valid on the root because Cobra ignores them there
(`WithGroup`, `WithHidden`, `WithAliases`, `WithTypoHints`).
They return `CommandOption`, not `RootOption`, and so they fail to compile if
passed directly to `New(...)` or to a top-level position where only `Option`
is accepted. They are valid inside `nabat.WithCommand("deploy", ...)` because
the inner argument list accepts the wider `CommandOption` type.

## Lifecycle hooks receive `*Context`

Nabat runs its own hook pipeline inside Cobra's `RunE`:

```text
preRun -> validations -> run -> postRun
```

Hooks are `func(*Context) error`, so they have access to resolved args and
flags, the theme, the IO bundle, the logger, and the Go context.

The pipeline works this way because:

- Cobra hooks receive `(*cobra.Command, []string)`, which has none of the above.
  Nabat hooks carry the full per-invocation state.
- Cobra allows only one `PersistentPreRunE` per command. Nabat's pipeline is a
  slice, so any number of hooks coexist. The built-in help and version features
  both register hooks here.
- Hook order is fixed. `preRun` runs before `validations`, which run before
  `run`, which runs before `postRun`.
- `postRun` runs even when `run` returns nil. Its errors are captured. The first
  error from `run` takes precedence, but cleanup hooks still execute.

A hook can short-circuit the rest of the pipeline by returning
`nabat.ErrHandled`. The framework treats `ErrHandled` as a successful exit:
remaining hooks and the run function are skipped, and `App.Run` returns nil. The
built-in `--version` and `--help` flag handlers use this sentinel.

Trade-off: Nabat owns Cobra's `RunE` slot for its pipeline. Direct Cobra access
stays available for extensions through `App.UnsafeRoot()`.

## `*Context` implements `context.Context`

`*nabat.Context` satisfies Go's `context.Context` interface. Handlers pass `c`
directly to any function that expects a `context.Context`:

```go
nabat.WithRun(func(c *nabat.Context) error {
    req, err := http.NewRequestWithContext(c, "GET", url, nil)
    if err != nil {
        return err
    }
    rows, err := db.QueryContext(c, query)
    return err
})
```

This works well because:

- CLI commands make network calls and database queries that need cancellation.
  Without `context.Context`, handlers have no standard way to respond to signals
  or timeouts.
- Handlers do not need to extract a context field. They pass `c` directly.
- `context.WithTimeout`, `context.WithCancel`, and signal handling all work as
  expected because `*Context` is a `context.Context`.

Trade-off: `*Context` carries both Nabat state (resolved values, output methods)
and Go context state (cancellation, deadlines). Callers must not store a
`*Context` past the handler's lifetime, the same rule that applies to
`http.Request.Context()`.

## Spinner takes `func() error`

The `Spinner` callback signature is `func() error` with no `*Context` argument:

```go
c.Spinner("Deploying", func() error {
    return deploy(c)
})
```

This shape works well because:

- Work under a spinner can fail (network calls, file I/O). An `error` return
  makes failure easy to handle.
- Callers already have `c` in scope. A `*Context` parameter would be redundant
  and could trick callers into thinking it differs from the outer one.
- In non-interactive mode the callback runs without the spinner. Its error
  propagates the same way.

## One `IOStreams` bundle for input, output, and errors

`App.IO` is a single `*nabat.IOStreams` value with three streams (`In`,
`Out`, `ErrOut`) and a set of capability methods. Set it with `WithIO(io)`. Each
`*Context` exposes the same bundle through the public `c.IO` field, so handlers
reach it without going back to the App.

The bundle holds:

- `IO.In`, `IO.Out`, `IO.ErrOut` for input, normal output, and diagnostics.
- Capability methods: `IO.IsStdoutTTY()`, `IO.IsStderrTTY()`, `IO.IsStdinTTY()`,
  `IO.ColorEnabled()`, `IO.CanPrompt()`, `IO.TerminalWidth()`.
- Raw accessors `IO.RawIn()`, `IO.RawOut()`, `IO.RawErrOut()` for libraries that
  need the underlying `*os.File`.

`Context.Warn`, `Context.Error`, `Context.Success`, and `Context.Info` all
write to `IO.ErrOut`. Only "the product" of a command — tables, lists, trees,
encoded formats (`JSON`, `YAML`, `TOML`), markdown, and the raw `Print` /
`Println` family — goes to `IO.Out`. Progress bars and spinners also render on
`IO.ErrOut`. The split matches POSIX convention: `mycli deploy | jq` never
feeds a "✓ deployed" line into the JSON parser, and a script consuming the
data stream is not surprised by status chrome.

This is the same routing GitHub CLI (`gh`) uses: positive narrative messages
are diagnostic UI for the human running the command, not data the next
process in the pipeline should consume.

One bundle works well because:

- One accessor per concept. There is no need to choose between a "raw" writer
  and a "color-aware" writer at every call site.
- The shape matches GitHub CLI's `pkg/iostreams.IOStreams` and Kubernetes'
  `genericiooptions.IOStreams`. Code reads the same as in those projects.
- Handing `IO.Out` or `IO.ErrOut` to libraries (huh, bubbletea, glamour) keeps
  their own terminal detection working, because the wrapper preserves the file
  descriptor.
- Tests stay simple. `nabattest.NewIO()` returns a bundle plus three buffers in
  one call:

  ```go
  io, _, out, errOut := nabattest.NewIO()
  app := nabat.MustNew("myctl", nabat.WithIO(io))
  // ... run app ...
  require.Contains(t, out.String(), "deployed")
  require.Contains(t, errOut.String(), "warning:")
  ```

## Terminal-aware output by default

`IOStreams` detects whether stdin and stdout are terminals and adjusts the
output for you.

- In a terminal, output is colored and prompts run.
- In a pipe or CI, ANSI escape codes are stripped and prompts are skipped.
- `NO_COLOR`, `CLICOLOR`, and `CLICOLOR_FORCE` are honored without extra config.

The user does not set a `--no-color` or `--non-interactive` flag, and CI logs
stay clean without changes to the source.

This is implemented in the `nabat.dev` core package via the `IOStreams` type.
`IO.Out` and `IO.ErrOut` wrap the underlying writers with a
`colorprofile.Writer` that strips ANSI escapes off-TTY and preserves the file
descriptor on-TTY.

## Themes are built in and customizable

Nabat ships several built-in themes as embedded JSON manifests under
`themes/data/`, plus matching string constants in
[`nabat.dev/theme`](../theme):

- `theme.Default` — capability-aware default. Defers to the terminal's
  detected color profile and background luminance.
- `theme.Charm` — the higher-contrast Charm.land palette.
- `theme.Minimal` — bold-only, no foreground colors. Forces glamour into
  plain-text mode and disables chroma syntax highlighting; suited for CI
  logs, piped output, and accessibility.
- `theme.Dracula` — Dracula palette for dark backgrounds.
- `theme.CatppuccinMocha` — Catppuccin Mocha for dark backgrounds.
- `theme.Nabat` — the brand palette (warm Persian rock-candy tones).

You set the theme once with `WithTheme(name)`. The name resolves through
the embedded `themes` registry and flows through every output path:
semantic helpers (`Success`, `Warn`, `Error`, `Info`), structured output
(`Table`, `List`, `Tree`, encoded formats), help rendering, and Huh
prompts. The visual style stays the same across the CLI.

**The manifest is the single source of truth for built-in themes.**
There is no `init()`-based adapter registry, no public registration API
for chroma styles or prompt themes. A built-in theme is defined entirely
by its `theme/data/<name>.json` file: tokens, alias overrides,
syntax-highlighting palette (`chromaStyle`), markdown styling
(`glamourStyle`), and interactive prompt styling (`promptStyle`) all
share one style-spec dialect with `$primitive` and `$token` references
for palette consistency.

Three extension points cover customization beyond the catalog:

- For a string-shaped value (env var, flag, user config), pass the string
  to `WithTheme` directly. Unknown names produce a build-time
  `*ConfigErrors` from `nabat.New`, listing every available name.
- For a one-line tweak of a built-in theme's slot, use
  `nabat.WithThemeOverride` (or `nabat.WithThemeOverrides` for batches):

  ```go
  app, _ := nabat.New("myctl",
      nabat.WithTheme(theme.Dracula),
      nabat.WithThemeOverride(theme.StatusError, magenta),
  )
  ```

- For a fully programmatic theme — including one that needs a closure-
  based `huh.Theme` or a `chroma.Style` you want to own in Go —
  construct a `theme.Theme` value (or implement the `theme.Recipe`
  interface for capability-aware palette switching) and install it via
  `nabat.WithCustomTheme`:

  ```go
  acme := theme.Theme{
      Name:    "acme",
      Default: theme.VariantDark,
      Variants: map[theme.Variant]theme.Palette{
          theme.VariantDark: {
              Tokens: map[theme.Token]lipgloss.Style{
                  theme.StatusError: lipgloss.NewStyle().Foreground(lipgloss.Color("#E05454")),
              },
              Huh: acmeHuh, // huh.Theme escape hatch
          },
      },
  }
  app, _ := nabat.New("myctl", nabat.WithCustomTheme(acme))
  ```

`WithCustomTheme` is the only programmatic path; no built-in uses it,
and no public adapter registry exists to register a new built-in by side
effect. Adding a new built-in is a JSON-only edit.

Nabat uses the [Charm](https://charm.sh) ecosystem (Lip Gloss for
styling, Huh for prompts, Glamour for markdown) plus
[Chroma](https://github.com/alecthomas/chroma) for syntax highlighting
because:

- These libraries are actively maintained and handle the hard parts of
  terminal rendering (color profiles, non-TTY detection, wide
  characters, resize handling).
- Lip Gloss styles compose naturally, so one `theme.ResolvedTheme` can
  drive every output path.
- Huh provides selects, multi-selects, confirms, text inputs, file
  pickers, and spinners that share the same theme.

Trade-off: extra dependencies. Terminal UI is hard to get right, and
these libraries cover edge cases that a custom build would take a long
time to match.

## Theme manifests are validated by a public JSON Schema

Themes are described by JSON manifests that follow the
[Design Tokens Community Group](https://design-tokens.github.io/community-group/format/)
three-tier model. Every manifest references a JSON Schema in its `$schema`
field so editors give authors live validation, autocomplete, and hover
documentation while they edit.

The schema lives in two places that must agree:

- The in-repo file at [`themes/schema/v1.json`](../themes/schema/v1.json)
  is the source of truth. It is embedded into the binary via `//go:embed`
  and exposed through `theme.Schema() []byte`.
- The public URL `https://nabat.dev/schemas/theme/v1.json` mirrors the
  same bytes. Manifests reference this URL because editors fetch by URL,
  not by filesystem path.

Two tests guard the contract:

- `TestSchemaIDMatchesPublicURL` pins the `$id` field inside the schema
  document to the public URL constant. Renaming either side fails the
  build, not editor validation in the wild.
- `TestManifestsMatchSchema` validates every embedded manifest against
  the embedded schema. Drift between the schema and the manifests cannot
  be committed.

The URL is forever-binding — once an author has copied an `$schema` line
out of a Nabat manifest, the URL must keep serving valid bytes for as
long as their fork exists. The versioning policy reflects that:
backward-compatible additions ship inside `v1.json`; backward-incompatible
changes ship as `v2.json` at a new URL while `v1.json` stays unchanged.

For the manifest format and the hosting requirement, see
[Themes](themes.md).

## Flat public package with `internal/` helpers

Nabat exposes one user-facing package, `nabat`. Implementation-only code lives
under `internal/`.

This layout works well because:

- A flat public API keeps `With...` options and core types in one autocomplete
  surface.
- App authors do not have to learn or import a package map (`nabat/prompt`,
  `nabat/table`, and so on).
- Go's `internal/` rules block external packages from depending on internals, so
  Nabat can refactor them without breaking users.
- The opt-in subpackages (`completion`, `manpage`, `logging`) sit
  beside the core, not inside it. They are part of the public surface only when
  you import them.

## Appendix: Historical API notes

Short pointers for readers migrating old examples or forks:

- **Arg/flag doublets** — Older snapshots exposed parallel names (`WithRequired` /
  `WithRequiredFlag`, `WithEnv` / `WithEnvFlag`, etc.). The current API uses one
  helper per concept returning an anonymous interface over the relevant option
  families; see [Single exported name per concept](#single-exported-name-per-concept).
- **`WithHiddenIf`** — Removed in favor of `if cond { opts = append(opts,
  nabat.WithHidden()) }`; the language already expresses conditional option lists.
- **`nabat.Text` / `nabat.File`** — Former wrapper types that carried widget kind.
  Replaced by explicit mode options (`WithMultiline`, `WithFilePicker`, …) on
  string fields so the bind type stays `string` and intent stays visible at the
  call site.
- **`WithRoot(...)`** — Older wrapper around root-only options. Dropped because
  root configuration already flows through `New` via `RootOption`; no separate
  scope needed.
- **Deferred registration errors** — An older design queued registration
  failures on the app for `Run` to report. That violated silent-error
  expectations for code that built an app without running it. Inline and
  aggregated registration errors now report at construction time.
