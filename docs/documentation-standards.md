# Documentation Standards

How to write docs for Nabat code.

Good docs help two groups. They help contributors learn how Nabat works. They help users learn how to use Nabat the right way.

## Main Goal

Write clear docs that explain:

- **What** the type, function, or method does.
- **How** to use it.
- **What** inputs it takes and what outputs or errors it returns.

## Quick Template

Most public functions only need a short doc comment. Use this shape as a starting point:

```go
// FunctionName does X.
// One more sentence about limits, side effects, or edge cases.
//
// Errors:
//   - [ErrFoo]: when Y happens.
//
// Example:
//
//	result, err := FunctionName(input)
func FunctionName(input string) (Result, error) { ... }
```

The rest of this file explains each part in more detail.

## What NOT to Write

Do not put these things in docs.

### No Performance Claims

Docs describe behavior, not speed. Do not write:

- Speed words: "fast", "slow", "efficient", "quick", "optimized", "high-performance".
- Big-O numbers: `O(1)`, `O(n)`, time or space limits.
- Algorithm names used to imply speed.
- Benchmark results: "zero allocations", "50% faster than X", any timing or throughput numbers.
- Memory words: "low memory", "minimal allocations", or specific byte counts.

**Why we differ from the official Go guide:** The official Go doc guide allows Big-O notes when they matter (see [`sort.Sort`](https://pkg.go.dev/sort#Sort)). Nabat is a CLI framework, not a perf library. We keep docs focused on what the code does, not how fast it runs.

### Visual Decorations

Do not use:

- Lines of equals signs or dashes (`// ====`).
- ASCII art separators.
- Empty comment lines used only for spacing.
- Comments that add no information.

### TODO Comments About Moving Code

Do not write:

- `// TODO: move this to option.go`
- `// FIXME: this should be in context.go`
- `// NOTE: consider moving to...`

If code needs to move, move it. Git tracks the change.

### File History Comments

Do not write:

- `// merged from...`
- `// moved from arg.go`
- `// originally in...`

Git tracks file history. Comments should explain what code does now, not where it came from.

## What You SHOULD Write

### Purpose

- What the function, type, or method does.
- Why it exists.
- When to use it.

### Functionality

- What it does in plain words.
- How it turns inputs into outputs.
- The order of steps when the order matters.

### Usage

- How to use it, with short examples.
- Common use cases.
- How it fits into the bigger app flow.

### Code Examples in Documentation

Public functions should include examples.

- Use a tab to indent code blocks under an `// Example:` header.
- Show typical use.
- Keep examples short and focused.
- Use valid Go code that compiles.
- Put the example after the main description.

**Note:** GoDoc treats any indented block as a code block. A single tab is the common style. `gofmt` (Go 1.19+) will normalize doc comments for you.

**Inline example format:**

```go
// WithName sets the application name.
// The name appears in help output, the version command, and man pages.
// Nabat also uses it to build the default environment-variable prefix.
//
// Example:
//
//	app, err := nabat.New(
//	    nabat.WithName("myctl"),
//	)
func WithName(name string) Option { ... }
```

**Runnable Example functions (preferred for public APIs):**

Put these in `example_test.go`. The function name tells pkg.go.dev where to show the example:

| Pattern                  | Renders under          | Use when                                     |
|--------------------------|------------------------|----------------------------------------------|
| `Example()`              | Package overview       | A whole-package quick start.                 |
| `ExampleF()`             | Function `F`           | One example for a function.                  |
| `ExampleT()`             | Type `T`               | One example for a type.                      |
| `ExampleT_M()`           | Method `T.M`           | One example for a method.                    |
| `ExampleT_M_withFlags()` | Method `T.M` (variant) | More than one example. Suffix is lower case. |

The `// Output:` comment at the end of the function controls how `go test` runs it:

- **Present (even if empty)** — `go test` runs the example. The test fails if stdout does not match.
- **Missing** — the example is compiled but never run.

```go
func ExampleApp_Command() {
	app, _ := nabat.New("myctl",
		nabat.WithCommand("hello", nabat.WithRun(func(c *nabat.Context) error {
			fmt.Println("hi")
			return nil
		})),
	)
	_ = app.Run(context.Background())
	// Output: hi
}
```

**Whole-file examples:** if an example needs helper types, put it in a `_test.go` file with one `Example*` function plus the helper types in the same file.

### Qualifying Names in Examples

Inline `// Example:` blocks live next to the symbol they document. The reader is already on that symbol's GoDoc page, so qualifying every name with the package prefix is noise. Runnable `Example*` functions and the package overview live in a different namespace, so the prefix is required there.

- **Inline `// Example:` blocks inside `package nabat`:** write names unqualified (`WithRun(...)`, not `nabat.WithRun(...)`).
- **`example_test.go` and `doc.go`:** write names fully qualified (`nabat.WithRun(...)`). These are the copy-pasteable forms users see on pkg.go.dev's landing page and at the top of imports.
- **Subpackages** (e.g. `nabattest`, `manpage`, `completion`, `logging`): apply the same rule with their own package name.

```go
// Inline example inside package nabat — unqualified:
//
// WithRun sets the command handler.
//
// Example:
//
//	app.MustCommand("deploy",
//	    WithFlag("env", "staging"),
//	    WithRun(func(c *Context) error {
//	        var args struct {
//	            Env string `nabat:"env"`
//	        }
//	        if err := c.Bind(&args); err != nil {
//	            return err
//	        }
//	        c.Success("done", "env", args.Env)
//	        return nil
//	    }),
//	)
func WithRun(fn RunFunc) RootOption { ... }
```

```go
// example_test.go — qualified:

func ExampleWithRun() {
	app, _ := nabat.New("myctl",
		nabat.WithCommand("deploy", nabat.WithRun(func(c *nabat.Context) error {
			return nil
		})),
	)
	_ = app.Run(context.Background())
}
```

### Parameters and Return Values

- What each parameter means.
- What values are returned.
- Error conditions and what they mean.

### Behavior and Edge Cases

- Normal behavior.
- Edge cases and how the code handles them.
- Side effects, if any.
- Concurrency safety, when it matters.

### Constraints and Requirements

- What must be true before you call it.
- Limits or known restrictions.
- Other config that must be set first. Example: the version feature is off by default. Turn it on with `nabat.WithVersion("1.2.3")`. This adds the `version` subcommand and the `--version` flag.

### Error Documentation

Document when errors happen and which error values or types are returned:

```go
// New builds an App from the given options.
// It returns an error if any option is nil or if the result is invalid.
// New collects multiple validation failures into a [ConfigErrors] value.
//
// Errors:
//   - [ErrNilOption]: an option in opts is nil.
//   - [*ConfigErrors]: one or more options or app-level invariants failed.
//     Use errors.As(err, new(*ConfigErrors)) to inspect individual issues.
func New(opts ...Option) (*App, error) { ... }
```

```go
// Command creates a named subcommand on the App.
// It returns the new command and an error describing any registration failure.
// On error, no command is registered and the returned [*Command] is nil — callers
// must handle the error before chaining further registrations.
//
// For aggregated registration errors (multiple commands surfaced together with
// option errors), declare commands with [WithCommand] inside [New] instead.
// For panicking chains in main() or test setup, use [App.MustCommand].
//
// Errors:
//   - [ErrRegistrationFrozen]: called after [App.Run] / [App.RunArgs]
//   - [ErrNilOption]: a CommandOption in opts is nil
//   - "nabat: command name cannot be empty"
//   - [ErrArgFlagNameCollision]: an arg and a flag share a name
//   - errors from individual [CommandOption] application (wrapped, joined with
//     [ErrInvalidOption] when triggered by a nil option-list entry)
//   - errors from flag registration and command finalization
func (a *App) Command(name string, opts ...CommandOption) (*Command, error) { ... }
```

### Deprecation

Mark deprecated APIs with the standard prefix:

```go
// Deprecated: Use [New] with [WithIO] instead. This function will be removed in v2.0.
func NewWithWriter(w io.Writer, opts ...Option) (*App, error) { ... }
```

### Interface vs Implementation

**Interfaces** describe the contract:

```go
// RunFunc is the function signature for command handlers.
// It receives a [Context] with resolved args, flags, and the Go context.
// It returns an error when the command fails. A nil return means success.
// A RunFunc must not keep the Context after it returns.
type RunFunc func(c *Context) error
```

**Implementations** point back to the type or interface they satisfy:

```go
// ConfigErrors is an error value that holds many config failures.
// It implements the error interface.
// [New] returns it when validation finds more than one problem.
type ConfigErrors struct { ... }
```

### Reading resolved values on Context

Document [Context.Bind] for command handlers that read many args and flags at once.
Document [Context.Explicit] when logic must treat “user supplied” (CLI, env, or
prompt) differently from “default only”. Mention [BindAs] for one-off typed reads
in tests or narrow cases. Prefer [Context.Bind] in examples over ad hoc one-field
helpers.

```go
// Bind copies resolved positional arg and flag values into target, which must be
// a non-nil pointer to a struct. Each exported field tagged `nabat:"name"` receives
// the resolved value for name.
//
// Example:
//
//	var args struct {
//	    Env       string `nabat:"env"`
//	    Replicas  int    `nabat:"replicas"`
//	}
//	if err := c.Bind(&args); err != nil {
//	    return err
//	}
func (c *Context) Bind(target any) error { ... }
```

```go
// Explicit reports whether the named arg or flag was provided via the command line,
// environment variable, or interactive prompt, as opposed to using only a registered default.
//
// Example:
//
//	if c.Explicit("optional-flag") {
//	    // user supplied the value
//	}
func (c *Context) Explicit(name string) bool { ... }
```

### Concurrency Safety

Document whether a type or method is safe to use from many goroutines. (A goroutine is a lightweight thread managed by the Go runtime.)

```go
// App is safe for concurrent use after construction is done.
// Register all commands and flags before you call [App.Run] or [Run].
type App struct { ... }

// Context is NOT safe for concurrent use.
// Each command call gets its own Context.
// Do not keep or share the Context across goroutines.
type Context struct { ... }

// Theme is safe to share across App instances.
// It is read-only after construction.
type Theme struct { ... }
```

### Panics

If a function or method may panic, say so. Use a "Panics if" sentence near the end of the doc comment.

```go
// MustNew is like [New] but panics on config errors.
// Panics if any option is nil or if validation fails.
func MustNew(opts ...Option) *App { ... }
```

### Goroutines and Lifecycle

If a function starts a background goroutine, document who must stop it and how cancellation works.

```go
// Watch starts a background goroutine that re-renders the progress bar.
// The goroutine runs until ctx is canceled.
// The caller must cancel ctx to release the goroutine.
func (p *Progress) Watch(ctx context.Context) { ... }
```

### Struct Fields

Document each exported struct field above the field. Start with the field name. Follow the same rules as function comments.

```go
type Theme struct {
	// Symbols is the symbol set used for status messages.
	// Symbols defaults to [DefaultSymbols] when nil.
	Symbols *SymbolSet

	// Colors maps roles (success, warning, error) to ANSI styles.
	// A nil map turns colored output off.
	Colors *ColorSet
}
```

### Cross-References

Use the bracket form `[Symbol]` to link to other Go symbols (Go 1.19+):

```go
// WithRun sets the function that runs when the command is called.
// The function receives a [Context] with all resolved args and flags.
// See [WithArg] and [WithFlag] for how to define args and flags.
// See [Context.Ctx] for the Go context.
func WithRun(fn RunFunc) RootOption { ... }
```

**Link targets:**

- `[FunctionName]` — function in the same package.
- `[TypeName]` — type in the same package.
- `[TypeName.MethodName]` — method.
- `[pkg.Symbol]` — symbol in another package (for example, `[context.Context]`, `[io.Writer]`).

**Generics note:** `[T]` inside a doc comment can look like a type parameter. Put a space or punctuation around doc links so the parser can tell them apart from generic syntax.

### Unexported Identifiers

Document unexported types and functions when the name and signature do not make the behavior clear. Skip the extra GoDoc parts. You do not need `[Symbol]` cross-references or `Errors:` blocks. One short line is often enough.

```go
// resolveArg walks the resolution chain (positional, env, prompt, default).
// It returns the first non-empty value and where it came from.
func resolveArg(c *Command, name string) (string, source, bool) { ... }
```

If an unexported function is hard to use without an example, export it.

## Style Rules

### GoDoc Standards

- **Start with the name** — Begin every comment with the name of the thing you document:
  - Good: `// App is the root CLI application.`
  - Good: `// WithName sets the application name.`
  - Bad: `// This is the root CLI application.`
  - Bad: `// Sets the application name.`

- **First sentence is the summary** — pkg.go.dev, `go doc -short`, and IDE tooltips show only the first sentence in listings. Make it self-contained. Do not put key info in the second sentence only.
  - Good: `// New builds an App from the given options and returns an error if any option is invalid.`
  - Bad: `// New builds an App. It returns an error if any option is invalid.` (the error contract is hidden in listings)

- **Use third person** — Write, "App stores..." not "I store..." or "We store...":
  - Good: `// Context stores the resolved values for one command call.`
  - Bad: `// This stores the resolved values...`
  - Bad: `// We store the resolved values...`

### Doc Comment Syntax (Go 1.19+)

Go 1.19 added a small Markdown-like syntax for doc comments. The following parts are supported.

#### Headings

A line that starts with a `#` and a space becomes a heading. The line must be unindented. Put a blank comment line before and after the heading. Use headings in `doc.go` or in long type and function comments.

```go
// # Configuration
//
// The App accepts a list of [Option] values...
```

#### Lists

Bullet markers (`-`, `*`, `+`, `•`) and numbered lists (`1.`, `2.`) work. List items must be indented. **Nested lists are not supported.**

```go
// Run executes the command tree. It performs:
//
//   - argument parsing
//   - environment-variable resolution
//   - interactive prompts for missing required values
//   - the matched command's RunFunc
```

#### Doc Links

Use `[Symbol]` to link to Go identifiers. See the "Cross-References" section above for the link target forms.

#### URL Links

Use the shortcut-reference form. Define link targets at the bottom of the comment block.

```go
// WithTheme applies a color theme. See [Catppuccin] for the default palette.
//
// [Catppuccin]: https://catppuccin.com
```

### Clarity and Conciseness

- Use full sentences.
- Keep comments short but meaningful.
- Avoid extra words.
- Be direct and clear.

### Language Guidelines

- **No marketing words** — Avoid words like "simple", "powerful", "robust", "elegant".
- **No superlatives** — Do not write "the best way" or "most reliable".
- **Stick to facts** — Describe what the code does.

### Code Examples

- Public APIs need examples.
- Use a tab to indent inline code blocks. `gofmt` will normalize them.
- Prefer **runnable Example functions** in `example_test.go`.
- Keep examples **short and focused**.

## Package Documentation (doc.go)

Nabat is a single-package library. Its package-level docs live in `doc.go`.

### Format requirements

- Start with `// Package nabat` followed by a clear description.
- The first sentence is the summary. It appears in package listings.
- Use Markdown-style headers (`#`) for sections.
- Include code examples with a tab indent.
- Cover: purpose, main concepts, quick start, common patterns.

### What to include

- Package overview and purpose.
- Core concepts (App, Command, Context, arg resolution, output).
- Quick start example.
- Links to the `examples/` directory.

### What NOT to include

- Performance details.
- Algorithm complexity.
- File layout or history.
- Per-function docs (those belong in each file).

### Example structure

```go
// Package nabat provides a CLI framework for Go.
//
// Nabat wraps Cobra with a cleaner config API, adaptive arg resolution,
// styled terminal output, and interactive prompts.
//
// # Core Concepts
//
// An [App] is the root of a CLI application. Add commands with [App.Command].
// Each command receives a [Context] with all resolved args and flags.
//
// # Quick Start
//
//	app, err := nabat.New("myctl",
//	    nabat.WithVersion("1.0.0"),
//	    nabat.WithCommand("deploy",
//	        nabat.WithSelectArg("env", "", []string{"staging", "production"},
//	            nabat.WithRequired(),
//	        ),
//	        nabat.WithRun(func(c *nabat.Context) error {
//	            var args struct {
//	                Env string `nabat:"env"`
//	            }
//	            if err := c.Bind(&args); err != nil {
//	                return err
//	            }
//	            c.Success("deployed", "environment", args.Env)
//	            return nil
//	        }),
//	    ),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := app.Run(context.Background()); err != nil {
//	    os.Exit(1)
//	}
//
// # Examples
//
// See the examples/ directory for complete working examples.
package nabat
```

## Examples

### Good Documentation

```go
// App is the root CLI application.
// It holds the command tree, global flags, and config.
// Build it with [New] or [MustNew]. Then add commands with [App.Command]
// and global flags with [App.Flag].
// Call [App.Run] in production or [Run] in tests.
type App struct { ... }

// WithArg defines one positional argument with adaptive resolution.
// The type parameter T selects the value kind (for example, string, int, or [Text]).
// WithArg resolves the value from the first source that has one, in this order:
// positional argument, environment variable, interactive prompt, default value.
// If the arg is required and no source provides a value, the command returns an error.
//
// Example:
//
//	app.MustCommand("deploy",
//	    WithSelectArg("environment", "", []string{"staging", "production"},
//	        WithRequired(),
//	        WithPrompt("Target environment", ""),
//	    ),
//	    WithRun(deployHandler),
//	)
func WithSelectArg(name, defaultVal string, choices []string, opts ...ArgOption) RootOption { ... }

// Success writes a success message to stderr with a styled symbol and optional key-value fields.
// When stderr is not a terminal, Success writes plain text without escape codes.
//
// Example:
//
//	c.Success("deployed", "environment", env, "replicas", 3)
func (c *Context) Success(msg string, args ...any) { ... }
```

### Bad Documentation

```go
// App is a highly optimized, zero-allocation CLI application root.
// Uses the fastest possible command lookup.
// Extremely robust and reliable in all environments.
type App struct { ... }

// WithArg is a powerful option that efficiently handles all arg scenarios.
// Uses O(1) lookup for resolved values.
// Best-in-class adaptive resolution.
func WithArg[T ArgValue](name string, defaultVal T, opts ...ArgOption) RootOption { ... }

// ========================================
// Output Methods
// ========================================

// TODO: move this to output.go
// Success writes a success message.
func (c *Context) Success(msg string, args ...any) { ... }
```

## Tooling

These tools enforce parts of this standard:

- **`gofmt`** — normalizes doc-comment formatting (Go 1.19+).
- **`go vet ./...`** — basic correctness checks.
- **`staticcheck`** — relevant rules:
  - `ST1020` — exported function comments must start with the function name.
  - `ST1021` — exported type comments must start with the type name.
  - `ST1022` — exported variable and constant comments must start with the name.
- **`revive`** — relevant rules: `package-comments`, `var-naming`, `unexported-return`. (The "exported comment must start with the name" check is left to `staticcheck` ST1020/ST1021/ST1022 to avoid duplicate warnings.)
- **`golangci-lint`** — runs the rules above as part of `make lint` and CI.

What tools cannot check (these still need a human review):

- No performance claims.
- No marketing words.
- Examples are runnable and useful.
- Cross-references point to real symbols.

## Review Checklist

When you write or review docs, check:

### Content Rules

- [ ] No performance words (fast, efficient, optimized, zero-allocation).
- [ ] No algorithm complexity (Big-O, O(1)).
- [ ] No benchmark claims.
- [ ] No memory usage details.
- [ ] No marketing words (powerful, robust, simple, amazing).
- [ ] No decorative comment lines (`// ====`, `// ----`).
- [ ] No TODO/FIXME comments about moving code.
- [ ] No file history comments (merged from, moved from, originally in).
- [ ] Every comment adds useful information.

### Style Checks

- [ ] Comments start with the function or type name.
- [ ] The first sentence is a self-contained summary.
- [ ] Third-person, descriptive language.
- [ ] Headings, lists, and links use Go 1.19+ syntax.
- [ ] Clear explanation of what the code does.

### Documentation Completeness

- [ ] Parameters and return values are documented.
- [ ] Error conditions are documented with specific error types or messages.
- [ ] Edge cases and limits are mentioned.
- [ ] Concurrency safety is documented when it matters.
- [ ] Generic type constraints are documented.
- [ ] Panics documented with a "Panics if" sentence.
- [ ] Goroutine lifecycle and cancellation are documented.
- [ ] Exported struct fields are documented.

### Examples and References

- [ ] Public APIs include examples (inline or runnable Example functions).
- [ ] Inline code examples use a tab indent.
- [ ] Cross-references use the `[Symbol]` form.

### Special Cases

- [ ] Deprecated APIs use the `// Deprecated:` prefix and name a replacement.
- [ ] Interfaces document the contract. Implementations point back to the interface.
- [ ] Package documentation lives in `doc.go`.
- [ ] `doc.go` starts with `// Package nabat`.

## Additional Resources

- [Go Doc Comments](https://go.dev/doc/comment) — Official guide (Go 1.19+ syntax).
- [Effective Go — Commentary](https://go.dev/doc/effective_go#commentary) — General guidelines.
- [Example Functions](https://go.dev/blog/examples) — Writing testable examples.
- [Go Wiki: Deprecated](https://go.dev/wiki/Deprecated) — Deprecation marker convention.
- [go/doc/comment](https://pkg.go.dev/go/doc/comment) — Reference for the doc-comment AST.

## Summary

Docs explain **what** code does and **how** to use it. Docs do not explain how fast it runs. Focus on behavior, parameters, errors, and usage patterns.

When in doubt: if it describes the code, include it. If it describes performance, leave it out.
