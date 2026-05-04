# Design Principles

The values that guide how we build Nabat.

These principles help two groups of people:

- If you build CLIs with Nabat, they explain why the API looks the way it does.
- If you contribute to Nabat, they tell you which trade-offs to make.

For how the package is structured, see [Architecture](architecture.md). For why we picked
specific approaches, see [Design Decisions](design-decisions.md).

## Audience and Influences

Nabat is a Go library, but it builds CLIs for humans. So the principles below cover both
sides: how we treat the developer who writes Go code, and how the resulting CLI treats the
person at the terminal.

We did not invent these ideas. Nabat stands on top of forty years of CLI design work:

- [The Unix Programming Environment](https://en.wikipedia.org/wiki/The_Unix_Programming_Environment) and the original UNIX philosophy.
- [POSIX utility conventions](https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html) and the [GNU Coding Standards](https://www.gnu.org/prep/standards/html_node/Command_002dLine-Interfaces.html) for command-line behavior.
- [clig.dev](https://clig.dev/), a modern guide that updates the UNIX rules for today's tools.
- [12 Factor CLI Apps](https://medium.com/@jdxcode/12-factor-cli-apps-dd3c227a0e46) by Jeff Dickey.
- The [Charm](https://charm.sh) ecosystem (Lip Gloss, Huh, Glamour) for terminal styling and prompts.
- GitHub CLI's `pkg/iostreams` package, which inspired the `IOStreams` type in the nabat core.

A good developer experience is the goal. Every principle below is one way to reach it.

## 1. Respect the Human at the Terminal

The CLI runs for a person, not for the framework. Output should help them. Errors should
guide them. Prompts should appear when they help and stay quiet when they do not.

**What this means:**

A user sees colors and symbols on a real terminal, plain text in a pipe, and a clean log
in CI. The same code produces all three. The user never sets a flag to fix bad output.

**In practice:**

- `Success`, `Warn`, `Error`, and `Info` carry meaning, not just text. Each one writes to
  the right stream and uses the theme.
- Prompts run only when stdin is a real terminal. In CI the framework falls back to env
  vars, defaults, or a clear error.
- Themes default to safe colors. You opt up for richer palettes when you want them.

**Example:**

```go
nabat.WithRun(func(c *nabat.Context) error {
    c.Success("deployed", "environment", env, "replicas", 3)
    return nil
})
```

On a real terminal you see a green check, the message, and the key-value pairs in the theme
style. Because `Success` writes to stderr, piping the command to `jq` leaves the data stream
clean — the "✓ deployed" line still surfaces on the user's terminal but never reaches the
JSON parser. In CI logs, no escape codes.

## 2. Be a Good UNIX Citizen

A CLI built with Nabat lives among other tools. Pipes, scripts, and CI runners must work
the way users expect.

**What this means:**

We follow the UNIX rules for streams, env vars, exit codes, and color. The framework picks
the right behavior for the context, so the user never has to.

**In practice:**

- `IO.Out` is for data. `IO.ErrOut` is for messages, warnings, and errors. Pipes work
  cleanly because warnings never land in the data stream. See [One IOStreams bundle](design-decisions.md#one-iostreams-bundle-for-input-output-and-errors).
- The framework honors `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, and `TERM=dumb` with no
  setup. See [Terminal-aware output by default](design-decisions.md#terminal-aware-output-by-default).
- Exit codes follow Cobra and POSIX: zero on success, non-zero on failure.
- Machine-readable output is built in. `Context.JSON`, `Context.YAML`, `Context.TOML`, and
  `Context.Encode` give the user a stable format for scripts.

**Example:**

```go
nabat.WithRun(func(c *nabat.Context) error {
    if c.Bool("json") {
        return c.JSON(report)
    }
    c.Warn("partial result", "missing", missing)
    c.Table(headers, rows)
    return nil
})
```

`mycli report --json | jq .` works because the table goes to a different code path and the
warning goes to stderr. Nothing breaks the JSON parser.

## 3. Conventions Over Invention

When a convention exists, we follow it. Users should not have to learn Nabat dialects of
things they already know.

**What this means:**

`--help`, `--version`, env var names, exit codes, and color rules all follow the most common
form. We break a convention only when it would harm clarity, and we say why in
[Design Decisions](design-decisions.md).

**In practice:**

- `--help` and `-h` are on by default. `--version` and `-v` are opt-in but use the same
  shape. See [Help, version, and completion live in the core](design-decisions.md#help-version-and-completion-live-in-the-core).
- Env vars use `PREFIX_NAME` form. The prefix comes from the binary name, and you can change
  it with `WithEnvPrefix`. See [Environment variables](design-decisions.md#environment-variables-withenv-and-withenvalias).
- Sentinel errors and behaviors match the Go standard library: `MustNew` mirrors
  `regexp.MustCompile`, `Context` implements `context.Context`.
- When we break a rule on purpose, we name the break. The help heading reads
  `Global Flags:` to match Cobra and Docker, even though `gh` and `kubectl` say "inherited".
  Read why in [Help heading wording](design-decisions.md#help-heading-wording-global-flags-vs-inherited-flags).

## 4. One Definition, Every Environment

A single command definition runs unchanged on a developer's terminal, in a script, and in
CI. You never write `if isInteractive` in a handler.

**What this means:**

Positional args resolve through a cascade: `arg → env → prompt → default`. The first source
that provides a value wins. The framework picks the right source based on the context.

**In practice:**

- On a real terminal (a TTY), output gets color and prompts ask the user for missing values.
- When stdout is piped or stderr is a CI log, color is stripped. Prompts are skipped. The
  framework reads env vars or defaults instead.
- In tests, buffer-backed IO means non-interactive mode and no escape codes. The same
  command code runs the same way in production and in tests.

The framework detects the environment from the IO bundle you pass in, not from a
`--no-color` or `--non-interactive` flag. The behavior is always right for the context.

**Example:**

```go
nabat.WithSelectArg("environment", "", []string{"staging", "production"},
    nabat.WithRequired(),
    nabat.WithEnv("environment"),
    nabat.WithPrompt("Target environment", "",
        nabat.WithHint("staging"),
    ),
)

// myctl deploy production                -> uses the positional arg
// MYCTL_ENVIRONMENT=staging myctl deploy -> reads the env var (in CI)
// myctl deploy                           -> shows an interactive prompt
```

Flags follow a simpler cascade (`flag → env → default`). They never prompt, because
prompting for `--verbose` would be strange. See [Args and flags are different](design-decisions.md#args-and-flags-are-different).

## 5. Progressive Disclosure

Easy things stay easy. Advanced things are possible. Adding power must not make the basics
harder.

**Three levels:**

1. **Basic** — Works right away with good defaults.
2. **Intermediate** — Common changes are short.
3. **Advanced** — You get full control when you need it.

**Example:**

```go
// Level 1: basic. Just works. A single-command CLI is one flat call.
app, _ := nabat.New("greet",
    nabat.WithRun(func(c *nabat.Context) error {
        c.Success("hello world")
        return nil
    }),
)

// Level 2: common changes. Args, flags, descriptions, one subcommand.
app, _ := nabat.New("myctl",
    nabat.WithCommand("deploy",
        nabat.WithDescription("Deploy application"),
        nabat.WithSelectArg("environment", "", []string{"staging", "production"},
            nabat.WithRequired(),
            nabat.WithPrompt("Target environment", "",
                nabat.WithHint("staging"),
            ),
        ),
        nabat.WithFlag("replicas", 2),
        nabat.WithRun(deployHandler),
    ),
)

// Level 3: full control. Themes, IO, env prefix, nested commands.
app, _ := nabat.New("myctl",
    nabat.WithTheme(theme.Charm),
    nabat.WithIO(nabat.NewIO(customIn, customOut, customErr)),
    nabat.WithEnvPrefix("MYCTL_"),
    nabat.WithCommand("cluster",
        nabat.WithDescription("Cluster management"),
        nabat.WithCommand("scale", nabat.WithRun(scaleHandler)),
        nabat.WithCommand("status", nabat.WithRun(statusHandler)),
    ),
)
```

A new user reads level 1 and ships. A team lead reaches for level 3 once they need it. None
of those steps changes the others. For dynamic registration (loading subcommands
from plugins or runtime config), drop down to the imperative `App.Command` /
`App.MustCommand` API.

## 6. Discoverable APIs

Your IDE should help you find what you need. The way to do this is **scoped autocomplete**:
each room has only the doors that belong in that room.

**What this means:**

Options are typed per context. Inside `nabat.New(...)` you see app options. Inside
`WithFlag(...)` you see flag options. You never scroll past 150 unrelated names.

**In practice:**

```go
// Inside nabat.New(...) you see app config, root-applying RootOption values
// (because every RootOption also satisfies Option), and WithCommand.
nabat.New("myctl",
    nabat.With... // IDE shows: WithTheme, WithIO, WithLogger, WithErrorHandler,
                  //            WithExtension, WithVersion, WithHelp*,
                  //            WithDescription, WithFlag, WithArg, WithRun, ...,
                  //            WithCommand
)

// Inside WithCommand(name, ...) you see the wider CommandOption set.
nabat.WithCommand("deploy",
    nabat.With... // IDE shows: WithDescription, WithRun, WithArg, WithFlag,
                  //            WithAliases, WithGroup, WithHidden, WithCommand, ...
)

// Version sub-options nest inside WithVersion. Focused autocomplete in one place.
nabat.WithVersion("1.2.3",
    nabat.WithVersion... // IDE shows: WithVersionCommit, WithVersionDate, WithVersionFlagName,
                         //            WithVersionShorthand, WithoutVersionShorthand, ...
)

// Extension options live in their own subpackages.
import "nabat.dev/manpage"
nabat.WithExtension(manpage.New(
    manpage.With... // IDE shows: WithCommandName, WithSection, WithHidden
))
```

The lattice keeps each call site coherent. `Option` (the New site) is a
superset of `RootOption` (everything valid on the root command), which is a
strict subset of `CommandOption` (everything valid on any command). All five
families — `Option`, `RootOption`, `CommandOption`, `ArgOption`, and
`FlagOption` — are Go interfaces, so a single value can satisfy more than
one slot when that combination is meaningful. For example,
`WithRequired()`, `WithUsage(text)`, `WithEnv(name)`, and `WithEnvAlias(...)`
each return `interface { ArgOption; FlagOption }` and slot into both
`WithArg(...)` and `WithFlag(...)`; `WithHidden()` returns
`interface { CommandOption; FlagOption }` and slots into both subcommands
and flags; `WithDeprecated(message, ...)` returns the same union for
commands and flags (positional args are deliberately excluded because cobra
has no deprecation hook for them). Misuse at any other slot fails at
compile time, not at registration time and not at runtime.

### Naming Convention

The names follow one rule: `With<Feature>` enables or sets a feature, `Without<Feature>`
turns one off when the default is on. You can read the API as English: `WithVersion`,
`WithoutHelpShorthand`, `WithCommand`. Type `nabat.With` and let the IDE list the verbs.

Plural combinators (`ArgOptions`, `FlagOptions`, `CommandOptions`, `RootOptions`, `AppOptions`)
bundle multiple options of the same family. Value adapters use `As*` (`AsExtension` with
`WithExtension`). Imperative hook registration on a built command uses `On*` (`OnPreRun`,
`OnValidate`, `OnPostRun` on `*Command`, parallel to `App.OnPreRun`).

## 7. Make Wrong Code Unrepresentable

The best error is the one the compiler catches. The Go type system can stop misuse before
you run the program, so we let it do that work.

**What this means:**

Each option type fits one target. You cannot put an arg option on a flag, or a flag option
on an arg, or a string-prompt option on a confirmation dialog. The build fails instead of the
test.

**In practice:**

- You cannot pass an app option where a command option is expected.
- You cannot pass a flag-only option (`WithShort`) on a positional arg.
- You cannot pass a string-prompt option (`WithStringPassword`) on a confirm dialog.
- You cannot pass `WithDeprecated(...)` on a positional arg (cobra has no
  deprecation hook for positionals; the type system rejects the misuse
  instead of letting it silently no-op at runtime).
- You cannot pass `WithHidden()` on a positional arg (the slot still
  consumes the next CLI token regardless of help visibility, so hiding it
  is meaningless).
- You cannot pass a subcommand-only option (`WithGroup`, `WithHidden`,
  `WithAliases`, `WithTypoHints`) directly to `nabat.New`. Cobra ignores
  these on the root command, so the type system rejects them before the
  program runs. They remain valid inside `nabat.WithCommand` for nested
  subcommands.

```go
// Compiles. WithShort is a FlagOption.
nabat.WithFlag("verbose", false, nabat.WithShort('v'))

// Does NOT compile. WithShort is not an ArgOption.
nabat.WithArg("target", "", nabat.WithShort('t'))  // build error

// Compiles. WithDescription is a RootOption (also satisfies Option).
nabat.New("myctl", nabat.WithDescription("root summary"))

// Does NOT compile. WithGroup is a CommandOption, not an Option.
nabat.New("myctl", nabat.WithGroup("admin"))  // build error

// Compiles. WithGroup is valid inside a nested WithCommand.
nabat.New("myctl",
    nabat.WithCommand("admin-task",
        nabat.WithGroup("admin"),
        nabat.WithRun(adminHandler),
    ),
)

// Compiles. WithRequired() satisfies both ArgOption and FlagOption.
nabat.WithArg("name", "", nabat.WithRequired())
nabat.WithFlag("token", "", nabat.WithRequired())

// Compiles. WithDeprecated() satisfies both CommandOption and FlagOption.
nabat.WithCommand("legacy", nabat.WithDeprecated("use new-cmd instead"), ...)
nabat.WithFlag("old", "", nabat.WithDeprecated("use --new instead"))

// Does NOT compile. WithDeprecated is not an ArgOption (cobra has no
// positional-arg deprecation hook).
nabat.WithArg("legacy", "", nabat.WithDeprecated("..."))  // build error
```

For shared concepts like `WithRequired` and `WithEnv`, a single function
returns an anonymous interface that satisfies every relevant option family,
so one helper covers every site where the concept is meaningful. The cost is
one extra interface declaration in the return type. The win is that every
misuse is a build error, never a runtime check, and there is exactly one
exported name to learn per concept.

For the full option type breakdown, see [Architecture — Option Types](architecture.md#key-types)
and [One option type per target](design-decisions.md#one-option-type-per-target).

## 8. Explicit and Opt-In

Nothing happens unless you ask for it. Help is the one carve-out, because every CLI needs it.

**What this means:**

There are no hidden flags, no auto-registered subcommands beyond help, and no surprise state
changes. When you read the call site, you see the full surface of your CLI.

**In practice:**

**Help is built in (flag), opt-in (subcommand).** `nabat.MustNew("myctl")` registers the
persistent `--help`/`-h` flag with Nabat's themed renderer. This matches the GNU/POSIX
convention that every comparable CLI library follows (Cobra, Click, clap, urfave/cli). The
`help <subcmd>` subcommand is opt-in via `nabat.WithHelpCommand`. Use `WithHelpFlagName`,
`WithHelpShorthand`, `WithoutHelpFlag`, or `WithoutHelpShorthand` to tweak the flag. Use
`WithoutHelp` to turn the whole feature off.

**Version is opt-in but core.** Pass `nabat.WithVersion("1.2.3")` to install a `version`
subcommand and a `--version`/`-v` flag. Polish via the nested `VersionOption` family.
Without `WithVersion`, no version surface is installed. Version lives in core because it
must register a root flag, which extensions cannot do. See
[Extensions cannot change existing commands](design-decisions.md#extensions-cannot-change-existing-commands).

**Completion is also opt-in but ships in core.** Pass `nabat.WithCompletion(...)` and
the App grows a `completion` subcommand with bash/zsh/fish/PowerShell generators.
Polish via the nested `CompletionOption` family. Per-flag and per-positional
dynamic candidates use `WithCompleter` / `WithPositionalCompleter` and work
whether or not `WithCompletion` is set. Completion lives in core because Cobra's
completion machinery is already linked into every binary, the per-field
completers attach to flag/arg state the core owns, and shell completion is a
baseline CLI affordance ([clig.dev](https://clig.dev/)). See
[Help, version, and completion live in the core](design-decisions.md#help-version-and-completion-live-in-the-core).

**Other features are opt-in extensions.** `manpage`, `logging`, and any
third-party extension are installed by `nabat.WithExtension(...)`:

```go
app := nabat.MustNew("myctl",
    nabat.WithVersion("1.0.0"),
    nabat.WithCompletion(),
    nabat.WithExtension(manpage.New()),
)
```

The cost is one extra line per feature. The win is that you always know what your CLI
contains.

**Extensions use the same public API.** First-party and third-party extensions are
interchangeable. Both implement the `Extension` interface and install themselves through
`nabat.WithExtension(...)` using the same public accessors. There is no privileged path.

## 9. Errors Are Values, Surfaced Clearly

Errors are not surprises. They are values you get back from a function call. The framework
catches them at the right moment and shows them usefully.

**What this means:**

- **Construction errors** appear when you call `New`. The message tells you what went wrong
  and where.
- **Registration errors** are inline (`App.Command` and `Command.Command` return
  `(*Command, error)`) or explicit (`App.MustCommand` and `Command.MustCommand` panic).
  When you want a single aggregated report — multiple bad commands surfaced together with
  option and validation errors — declare commands with `nabat.WithCommand(...)` inside
  `nabat.New(...)`; everything aggregates into the `*ConfigErrors` returned by `New`.
  There is no deferred error path.
- **Runtime errors** flow through `IO.ErrOut` with the theme. Errors are never silent.

**In practice:**

```go
// Construction error: clear message, returned right away.
app, err := nabat.New("")
// err: "nabat: app name cannot be empty"

// Nil options are caught with their position.
app, err = nabat.New("myctl", nil)
// err: "nabat: option cannot be nil at index 0"

// Aggregated registration errors come back from New when you use WithCommand.
app, err = nabat.New("myctl",
    nabat.WithCommand("deploy",
        nabat.WithArg("env", ""),
        nabat.WithFlag("env", false), // collision with the arg above
    ),
)
// err: `nabat: command "deploy": name "env" is used by both arg and flag: nabat: name used by both arg and flag`
// errors.Is(err, nabat.ErrArgFlagNameCollision) == true

// Inline registration errors come back from Command directly.
app = nabat.MustNew("myctl")
_, err = app.Command("deploy",
    nabat.WithArg("env", ""),
    nabat.WithFlag("env", false),
)
// same error, just at the call site
```

`New` never returns a non-nil App when it returns an error, so you never get a half-built
config. Multiple validation errors collect into a single `*ConfigErrors` value that
implements `Unwrap() []error`, so `errors.Is` and `errors.As` still work.

**`Must*` only where the failure path is fatal.** `MustNew` is the panic-on-error form for
`main()` packages where a setup failure exits the process anyway. `MustCommand` is the
panic-on-error form for command registration when you want fluent chaining
(`app.MustCommand("cluster").MustCommand("scale", ...)`); it is intended for `main()` and
test setup where a registration error is a programmer bug. Both mirror `regexp.MustCompile`
and `template.Must` from the standard library. `MustBind` is the same shape for struct-tag
binding. There are no other `Must*` wrappers.

## 10. Stable, Additive Public Surface

Adding a new option to Nabat must not break a call site you already wrote. This is the core
promise of functional options, and it shapes how we evolve the library.

**What this means:**

We add. We do not change. New features arrive as new options, new methods, new subpackages.
Old code keeps working.

**In practice:**

- **Functional options grow safely.** A new option is a new exported function. No struct
  field, no positional change, no breaking call site. See [Functional options with private config](design-decisions.md#functional-options-with-private-config).
- **`internal/` shields users from refactors.** Implementation code lives under `internal/`,
  so we can move it around without breaking imports. See [Flat public package with `internal/` helpers](design-decisions.md#flat-public-package-with-internal-helpers).
- **Deprecation is visible, not hidden.** `WithDeprecated` and
  `WithDeprecatedShorthand` keep commands and flags in help with a clear
  `(deprecated: …)` note, rather than removing them. See [Cobra is the engine](design-decisions.md#cobra-is-the-engine).
- **Extensions use the same public API as built-ins.** When we add a feature, we add it
  through the surface that third-party authors also use. There is no inside track.

The trade-off is that the option list grows over time. This is fine: each option is one
focused setter. The cost is one autocomplete entry. The win is that you can upgrade Nabat
without rewriting your CLI.

## 11. Escape Hatches, Not Walls

A framework that hides everything traps you. A framework with escape hatches gives you the
common path and a way out when you need it.

**What this means:**

When you need raw access, you can have it. The escape hatches are clearly named so you know
when you are stepping outside the safe path.

**In practice:**

- **`App.UnsafeRoot()`** returns the underlying `*cobra.Command`. Use it when you need a
  Cobra feature Nabat does not wrap directly.
- **`Command.UnsafeCobra()`** is the same escape hatch on a subcommand: returns the
  underlying `*cobra.Command` for cases Nabat does not abstract (C-only Cobra hooks,
  dynamic registration). Mutating the command tree after construction may bypass Nabat
  invariants — prefer the `CommandOption` family for standard use.
- **`WithIO`** lets you bring your own input, output, and error streams. Tests use this.
  Custom hosts use this.
- **`WithLogger`** lets you bring your own `*slog.Logger`. The `logging` extension is one
  preset; bring your own when you want full control.
- **`WithErrorHandler`** lets you replace the default styled error printer.

The `Unsafe` prefix is a Go convention (see `unsafe.Pointer`) that marks the sharp edges.
Use them with intent. Stay on the safe path when you can.

## 12. Testability

If something is hard to test, the design is wrong. Every API choice should
make tests easier, not harder.

**What this means:**

You can test a CLI without `os.Args`, without a real terminal, without a
real network, and without a real clock. The framework gives you the seams.

**In practice:**

- **Errors are values.** `New` returns an error that tests can check, and the same
  error path covers declarative subcommand registration (`nabat.WithCommand` inside
  `New`). Inline registration via `App.Command` returns `(*Command, error)` so tests
  can assert on the specific failure at the call site.
- **IO is injectable.** `nabattest.NewIO()` returns a bundle plus three
  buffers in one call, so tests assert on captured output.
- **Interactivity follows IO.** Buffer-backed IO reports as non-TTY, so
  prompts are skipped and defaults take over. No test-only flag needed.
- **Commands run without `os.Args`.** `nabattest.Run(t, app, args)` lets
  you invoke any command with explicit arguments.
- **Values are inspectable.** `Context.Bind` reads resolved args and flags into a
  struct; `Context.Explicit` reports whether a name came from CLI, env, or
  prompt rather than only a default.
- **`Context` is a `context.Context`.** Cancellation, timeouts, and
  signals work with the standard library.

**Example:**

```go
io, _, out, errOut := nabattest.NewIO()
app := nabat.MustNew("myctl", nabat.WithIO(io))
// ... register commands ...
err := nabattest.Run(t, app, []string{"deploy", "staging"})
require.NoError(t, err)
require.Empty(t, out.String(), "stdout is reserved for command product")
require.Contains(t, errOut.String(), "deployed", "Success messages land on stderr")
```

## Summary

| Principle                              | How it shows up in code                                                                          |
|----------------------------------------|--------------------------------------------------------------------------------------------------|
| 1. Respect the human at the terminal   | TTY-aware output; semantic helpers; safe theme defaults                                          |
| 2. Be a good UNIX citizen              | stdout vs stderr split; honors NO_COLOR family; built-in JSON/YAML/TOML output                   |
| 3. Conventions over invention          | GNU/POSIX `--help` and `--version`; `PREFIX_NAME` env vars; standard-library shapes              |
| 4. One definition, every environment   | Cascade `arg → env → prompt → default`; one definition runs in TTY, CI, and tests                |
| 5. Progressive disclosure              | Three levels: basic just works, intermediate is short, advanced gives full control               |
| 6. Discoverable APIs                   | Scoped option types per call site; `With`/`Without` naming                                       |
| 7. Make wrong code unrepresentable     | Misuse fails the build, not a test                                                               |
| 8. Explicit and opt-in                 | No hidden flags or commands beyond help; extensions install with one line                        |
| 9. Errors are values, surfaced clearly | `*ConfigErrors` aggregates issues; runtime errors go through `IO.ErrOut` with theme              |
| 10. Stable, additive public surface    | New options never break call sites; `internal/` shields refactors; visible deprecation           |
| 11. Escape hatches, not walls          | `UnsafeRoot`, `Command.UnsafeCobra`, `WithIO`, `WithLogger`, `WithErrorHandler` for full control |
| 12. Testability                        | `nabattest.NewIO()`, `nabattest.Run`, buffer IO is non-TTY, `Context` is a Go context             |

These principles guide all development work. When you contribute to Nabat,
make sure your changes follow them. When you build a CLI on Nabat, expect
them to hold across releases.

For the package structure, see [Architecture](architecture.md). For the
reasoning behind specific choices, see [Design Decisions](design-decisions.md).
For the implementation patterns (functional options, private configs, option
type signatures), see [Architecture — Key Types](architecture.md#key-types)
and
[Design Decisions — Functional options with private config](design-decisions.md#functional-options-with-private-config).
