# Testing Standards

How to write tests for Nabat.

Good tests keep the framework working and show how to use it. Nabat is a small Go module with a few packages. Most tests live next to the code they cover and share a small set of helpers.

## Test File Structure

Tests are split by area. The table below lists the files in the root `nabat` package.

| File | Contains |
|------|----------|
| `app_test.go` | `App` construction, `New`, `MustNew`, `Run` flow |
| `app_theme_test.go` | Theme application, overrides, strict-theme validation |
| `arg_test.go` | Argument resolution (positional, env, default, required) |
| `command_test.go` | Command creation, nesting, name collision |
| `command_withcommand_test.go` | Declarative `WithCommand` registration inside `nabat.New` |
| `completion_options_test.go` | Completion option validation (`WithCompletion`, sub-options) |
| `completion_register_test.go` | Completion subcommand registration and shell output |
| `context_test.go` | Bind, value access, generic helpers |
| `context_access_test.go` | Typed `Context` getters and `Has` |
| `context_internal_test.go` | Internal `Context` state and unexported behavior |
| `field_option_test.go` | `FieldOption[T]` family and typed option helpers |
| `flag_test.go` | Flag registration and resolution |
| `flag_deprecation_test.go` | Flag and shorthand deprecation behavior |
| `form_test.go` | Interactive form fields, fallback walk, group layout |
| `help_test.go` | Help output behavior |
| `help_register_test.go` | Help command registration |
| `interactive_test.go` | Interactive prompts and TTY behavior |
| `option_helpers_test.go` | Shared option helpers (`WithRequired`, `WithEnv`, env name cleaning) |
| `output_test.go` | Semantic output (`Success`, `Warn`, `Error`, `Info`, `Print`) |
| `parse_test.go` | Parse option application (`WithParseOptions`) |
| `progress_test.go` | Progress bar validation and concurrency |
| `structured_test.go` | Structured output (JSON, YAML, tables) |
| `table_test.go` | Table rendering styles and cell formatting |
| `valuetype_test.go` | Value types and validation |
| `version_test.go` | Version output |
| `version_register_test.go` | Version command registration |
| `writer_test.go` | Output writer behavior |
| `helpers_test.go` | Shared internal test helpers |
| `example_test.go` | Runnable examples for the public API |

Subpackages keep their own test files: `logging/logging_test.go`, `manpage/manpage_test.go`, `completion/completion_test.go`. `IOStreams` tests live in `nabat/io_test.go` since `IOStreams` is part of the core.

When you add tests for a new feature, put them in the file that matches the area. If a test covers more than one area, pick the area that owns the main behavior under test.

## Package Declaration

Test files use one of two packages.

**Internal tests** use the same package as the code:

```go
package nabat
```

Use this only when the test needs unexported types or fields.

**External tests** use the `_test` suffix:

```go
package nabat_test
```

Prefer external tests when you only touch the public API. They prove that the API works from the outside, and they also work as examples for users. New tests should default to the external package unless you can show you need internal access.

## Test Naming

Use clear names that describe behavior, not mechanics.

| Pattern | Use case | Example |
|---------|----------|---------|
| `TestFunctionName` | Basic test | `TestNewDefaults` |
| `TestFunctionName_Scenario` | Specific scenario | `TestNewRejectsEmptyName` |
| `TestType_MethodName` | Method test | `TestContext_Has` |
| `TestBehaviorDescription` | Behavior test | `TestArgResolvesFromEnv` |

Names should read like a sentence about what happens.

Good:

```go
func TestRequiredArgMissing(t *testing.T) { ... }
func TestFlagResolvedFromEnv(t *testing.T) { ... }
func TestNestedSubcommands(t *testing.T) { ... }
```

Avoid:

```go
func TestArg(t *testing.T) { ... }
func TestCase1(t *testing.T) { ... }
func TestTest(t *testing.T) { ... }
```

### Subtest Naming

In table-driven tests, name each case so the reader knows what it covers.

Good:

```go
{name: "empty name returns error"}
{name: "nil option at index 0"}
```

Avoid:

```go
{name: "test1"}
{name: "case 2"}
```

### Grouping with Subtests

Use `t.Run` to group related cases under a shared setup.

```go
func TestApp_Command(t *testing.T) {
    t.Parallel()

    t.Run("returns command on success", func(t *testing.T) {
        t.Parallel()
        app := MustNew("test")
        cmd, err := app.MustCommand("hello", WithRun(func(c *Context) error { return nil }))
        require.NoError(t, err)
        require.NotNil(t, cmd)
    })

    t.Run("returns error for empty name", func(t *testing.T) {
        t.Parallel()
        app := MustNew("test")
        _, err := app.Command("", WithRun(func(c *Context) error { return nil }))
        require.Error(t, err)
    })
}
```

## Test Helpers

The `nabattest` package provides the canonical test helper for running an app:

```go
package nabattest

func NewIO() (*nabat.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer)
func NewTTYIO() (*nabat.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer)

func Run(t testing.TB, app *nabat.App, args []string, opts ...RunOption) error

func WithContext(ctx context.Context) RunOption
func WithEnvVars(values map[string]string) RunOption
```

`Run` defaults the context to `t.Context()` so it cancels when the test ends. Use `nabattest.WithContext(ctx)` only when you need a custom context (deadline, value, parent context).

Tests in `package nabat_test` import the helper:

```go
import "nabat.dev/nabattest"

func TestDeploy(t *testing.T) {
    t.Parallel()

    var out bytes.Buffer
    app := nabat.MustNew("myctl", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("deploy",
        nabat.WithArg("target", "staging"),
        nabat.WithRun(func(c *nabat.Context) error {
            target, err := nabat.BindAs[string](c, "target")
            if err != nil {
                return err
            }
            c.Print(target)
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"deploy", "production"})
    require.NoError(t, err)
    assert.Contains(t, out.String(), "production")
}
```

Tests in `package nabat` cannot import `nabattest` because that would create an import cycle. They use a small local wrapper in `helpers_test.go` with the same shape.

### Why this shape

- `t` comes first to match `t.Helper()` style.
- `args` is a single `[]string` so the call site is unambiguous when options follow: `Run(t, app, []string{"deploy", "production"}, nabattest.WithEnvVars(...))`.
- `ctx` is hidden by default because almost every test wants `t.Context()`. When you need a custom context, pass `nabattest.WithContext(ctx)`.

### When to use raw constructors

Use `New` and `App.Command` directly when the test checks that construction or registration fails. Both return errors inline.

```go
func TestNewRejectsEmptyName(t *testing.T) {
    t.Parallel()

    _, err := nabat.New("")
    require.Error(t, err)
    var cfgErrs *nabat.ConfigErrors
    require.ErrorAs(t, err, &cfgErrs)
    assert.True(t, cfgErrs.HasIssues())
}
```

## Assertions

Use `testify/assert` or `testify/require`. Do not write `if got != want { t.Errorf(...) }`.

### assert vs require

- `assert` — keeps running after a failure. Use for independent checks.
- `require` — stops the test on failure. Use when later code needs the result.

```go
app, err := nabat.New("myctl")
require.NoError(t, err)
require.NotNil(t, app)

assert.Contains(t, out.String(), "success")
assert.Contains(t, out.String(), "done")
```

### Error checking

| Function | Use when |
|----------|----------|
| `require.NoError(t, err)` | Setup must succeed and the test needs the result |
| `assert.NoError(t, err)` | Independent success check |
| `require.Error(t, err)` | Failure is required to continue |
| `assert.Error(t, err)` | Any error is fine |
| `assert.ErrorIs(t, err, target)` | Error must wrap a known error value |
| `assert.ErrorAs(t, err, &target)` | Error must be a specific type |
| `assert.ErrorContains(t, err, substr)` | Error message must contain text |

A "known error value" is a package-level variable like `ErrNilOption` or `ErrInvalidOption`. When one exists, prefer `ErrorIs`. For aggregated config errors, use `errors.As(err, new(*ConfigErrors))` and then assert on the inner sentinels. Use `ErrorContains` only when no such value fits, because error messages change more often than error values.

### Structured payloads

For JSON or YAML output, compare the parsed shape, not the raw bytes:

```go
assert.JSONEq(t, `{"name":"deploy"}`, out.String())
assert.YAMLEq(t, "name: deploy\n", out.String())
```

This avoids false failures from key order or whitespace.

### cmp.Diff for nested structs

`assert.Equal` works for simple values. For nested structs with many fields, the failure message is hard to read. Use `cmp.Diff` from `github.com/google/go-cmp/cmp` instead:

```go
import "github.com/google/go-cmp/cmp"

if diff := cmp.Diff(want, got); diff != "" {
    t.Fatalf("mismatch (-want +got):\n%s", diff)
}
```

Stay with testify for simple values, errors, and string contains. Use `cmp.Diff` for deep struct comparison.

## Table-Driven Tests

Use the table pattern when one test covers many cases.

```go
func TestValueType_Validate(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        vt      ValueType
        value   any
        wantErr bool
    }{
        {
            name:    "string accepts string value",
            vt:      String(),
            value:   "hello",
            wantErr: false,
        },
        {
            name:    "int rejects string value",
            vt:      Int(),
            value:   "not-an-int",
            wantErr: true,
        },
        {
            name:    "select rejects value not in choices",
            vt:      Select("staging", "production"),
            value:   "dev",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            err := tt.vt.validate(tt.value)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
        })
    }
}
```

## CLI Testing Patterns

Nabat is built for tests. Use these patterns to drive commands without touching `os.Args`.

### Capturing Output

Use `nabattest.NewIO()` to capture stdout and stderr in independent buffers. A
buffer-backed stream is treated as non-interactive, so prompts are skipped.
Keeping stdout and stderr separate is essential: semantic helpers
(`Success`, `Info`, `Warn`, `Error`) write to stderr, while `Print`/`Println`
and structured output (`Table`, `JSON`, `YAML`, `Tree`, ...) write to stdout.
A merged buffer would hide stream-routing bugs.

```go
func TestCommandOutput(t *testing.T) {
    t.Parallel()

    io, _, _, errOut := nabattest.NewIO()
    app := nabat.MustNew("greetctl", nabat.WithIO(io))

    app.MustCommand("greet",
        nabat.WithArg("name", "world"),
        nabat.WithRun(func(c *nabat.Context) error {
            name, err := nabat.BindAs[string](c, "name")
            if err != nil {
                return err
            }
            c.Success("hello", "name", name)
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"greet"})
    require.NoError(t, err)
    assert.Contains(t, errOut.String(), "hello")
}
```

### Testing Argument Resolution

Test each step of the resolution chain in its own test.

**Positional argument:**

```go
func TestArgResolvesFromPositional(t *testing.T) {
    t.Parallel()

    var got string
    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("deploy",
        nabat.WithArg("target", ""),
        nabat.WithRun(func(c *nabat.Context) error {
            v, err := nabat.BindAs[string](c, "target")
            if err != nil {
                return err
            }
            got = v
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"deploy", "production"})
    require.NoError(t, err)
    assert.Equal(t, "production", got)
}
```

**Environment variable:**

```go
func TestArgResolvesFromEnv(t *testing.T) {
    t.Setenv("TEST_TARGET", "production")

    var got string
    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("deploy",
        nabat.WithArg("target", "", nabat.WithEnv("target")),
        nabat.WithRun(func(c *nabat.Context) error {
            v, err := nabat.BindAs[string](c, "target")
            if err != nil {
                return err
            }
            got = v
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"deploy"})
    require.NoError(t, err)
    assert.Equal(t, "production", got)
}
```

`t.Setenv` disables `t.Parallel()` for this test, so do not call `t.Parallel()` here.

**Default value:**

```go
func TestArgFallsBackToDefault(t *testing.T) {
    t.Parallel()

    var got string
    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("deploy",
        nabat.WithArg("target", "staging"),
        nabat.WithRun(func(c *nabat.Context) error {
            v, err := nabat.BindAs[string](c, "target")
            if err != nil {
                return err
            }
            got = v
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"deploy"})
    require.NoError(t, err)
    assert.Equal(t, "staging", got)
}
```

**Required argument missing:**

```go
func TestRequiredArgMissing(t *testing.T) {
    t.Parallel()

    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("deploy",
        nabat.WithArg("target", "", nabat.WithRequired()),
        nabat.WithRun(func(c *nabat.Context) error { return nil }),
    )

    err := nabattest.Run(t, app, []string{"deploy"})
    require.Error(t, err)
}
```

### Testing Flag Resolution

Pass flags as `--name=value` arguments:

```go
func TestFlagResolvesFromArgs(t *testing.T) {
    t.Parallel()

    var got int
    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("scale",
        nabat.WithFlag("replicas", 1),
        nabat.WithRun(func(c *nabat.Context) error {
            n, err := nabat.BindAs[int](c, "replicas")
            if err != nil {
                return err
            }
            got = n
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"scale", "--replicas=3"})
    require.NoError(t, err)
    assert.Equal(t, 3, got)
}
```

### Testing Nested Commands

Use `Command.Command` to build the tree. Registration errors stay on the `App` and are returned from `Run`:

```go
func TestNestedCommandReceivesArg(t *testing.T) {
    t.Parallel()

    var got string
    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))

    cluster := app.MustCommand("cluster", nabat.WithDescription("Cluster management"))
    cluster.MustCommand("create",
        nabat.WithArg("name", ""),
        nabat.WithRun(func(c *nabat.Context) error {
            v, err := nabat.BindAs[string](c, "name")
            if err != nil {
                return err
            }
            got = v
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"cluster", "create", "my-cluster"})
    require.NoError(t, err)
    assert.Equal(t, "my-cluster", got)
}
```

### Testing explicit vs default

Use `Context.Explicit` after resolution when handler logic must distinguish a
user-supplied value (CLI, env, or prompt) from a registered default:

```go
func TestExplicitTrueWhenUserPassesArg(t *testing.T) {
    t.Parallel()

    var explicit bool
    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("deploy",
        nabat.WithArg("target", "staging"),
        nabat.WithRun(func(c *nabat.Context) error {
            explicit = c.Explicit("target")
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"deploy", "production"})
    require.NoError(t, err)
    assert.True(t, explicit)
}
```

For optional reads or type assertions, use `nabat.BindAs` (or `Context.Bind` into a struct).
`BindAs` returns an error when the name has no resolved value.

### Testing Construction Failures

Call `New` and `App.Command` directly when you want to check that registration fails:

```go
func TestNewRejectsEmptyName(t *testing.T) {
    t.Parallel()

    _, err := nabat.New("")
    require.Error(t, err)
    var cfgErrs *nabat.ConfigErrors
    require.ErrorAs(t, err, &cfgErrs)
    assert.True(t, cfgErrs.HasIssues())
}

func TestCommandRejectsNameCollision(t *testing.T) {
    t.Parallel()

    app := nabat.MustNew("test")
    _, err := app.Command("deploy",
        nabat.WithArg("target", ""),
        nabat.WithFlag("target", ""),
        nabat.WithRun(func(c *nabat.Context) error { return nil }),
    )
    require.ErrorIs(t, err, nabat.ErrArgFlagNameCollision)
}
```

### Testing with a Custom Context

Use `nabattest.WithContext(ctx)` when you need to attach a value, set a deadline, or pass a parent context:

```go
func TestCommandReceivesGoContext(t *testing.T) {
    t.Parallel()

    type ctxKey string
    ctx := context.WithValue(t.Context(), ctxKey("env"), "test")

    var out bytes.Buffer
    app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))

    var got string
    app.MustCommand("run",
        nabat.WithRun(func(c *nabat.Context) error {
            got, _ = c.Value(ctxKey("env")).(string)
            return nil
        }),
    )

    err := nabattest.Run(t, app, []string{"run"}, nabattest.WithContext(ctx))
    require.NoError(t, err)
    assert.Equal(t, "test", got)
}
```

## Time and Concurrency

Some code uses `time.Sleep`, `time.After`, or context deadlines. Real time makes tests slow and easy to break. Use `testing/synctest.Test` to run such code with a fake clock. Time only moves when every goroutine in the test is blocked, so a one-hour timeout passes in microseconds.

```go
import "testing/synctest"

func TestRunRespectsDeadline(t *testing.T) {
    t.Parallel()

    synctest.Test(t, func(t *testing.T) {
        ctx, cancel := context.WithTimeout(t.Context(), time.Hour)
        defer cancel()

        var out bytes.Buffer
        app := nabat.MustNew("test", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
        app.MustCommand("wait",
            nabat.WithRun(func(c *nabat.Context) error {
                <-c.Done()
                return c.Err()
            }),
        )

        err := nabattest.Run(t, app, []string{"wait"}, nabattest.WithContext(ctx))
        assert.ErrorIs(t, err, context.DeadlineExceeded)
    })
}
```

A "bubble" is the area of code that runs inside `synctest.Test`. Use `synctest.Wait()` inside the bubble when you need to wait until every other goroutine is blocked before you check state.

## Fuzzing

Nabat parses argv, flag values, environment variables, and value types. These parsers must not panic on any input. Add a fuzz test for every parser.

A fuzz test must:

1. Seed with at least one valid input via `f.Add(...)`.
2. Assert no panic on any input.
3. Assert a stable round-trip when one is defined (parse then format must give the same string).

```go
func FuzzRunArgs(f *testing.F) {
    f.Add("deploy production --env=staging")
    f.Add("--help")
    f.Add("")

    f.Fuzz(func(t *testing.T, raw string) {
        var out bytes.Buffer
        app := nabat.MustNew("fuzzctl", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
        app.MustCommand("deploy",
            nabat.WithArg("target", ""),
            nabat.WithRun(func(c *nabat.Context) error { return nil }),
        )

        _ = nabattest.Run(t, app, strings.Fields(raw))
    })
}
```

Inputs that crash the parser are saved under `testdata/fuzz/<TestName>/`. Commit these files. CI replays them on every run, so any new bug that breaks one of them breaks the build right away.

## Test Metadata and Output

Two small helpers from the standard library make test reports nicer.

**`t.Attr(key, value)`** attaches metadata to a test. The value shows up in the test log and in `go test -json` output, where CI dashboards can read it.

```go
func TestArgResolvesFromEnv(t *testing.T) {
    t.Attr("issue", "GH-123")
    t.Attr("owner", "@cli-team")
    // ...
}
```

**`t.Output()`** returns an `io.Writer` that writes to the test stream without the `file:line` prefix that `t.Log` adds. Use it for diffs, JSON dumps, and structured logs that should stay readable.

```go
fmt.Fprintln(t.Output(), cmp.Diff(want, got))
```

## Test Isolation

Each test must be independent. Do not share state between tests.

- `t.TempDir()` returns a directory that is removed after the test. Use it for any file you create.
- `t.Chdir(dir)` changes the working directory and restores the old one when the test ends. Use it instead of `os.Chdir`.
- `t.Setenv(key, value)` sets an environment variable and restores it after the test. It also disables `t.Parallel()` for that test.
- `t.ArtifactDir()` returns a directory for files you want to keep after a CI failure (golden diffs, generated configs, screenshots). With the `-artifacts` flag, the files are saved; without it, they are removed at the end.

```go
func TestWithTempFile(t *testing.T) {
    t.Parallel()

    dir := t.TempDir()
    path := filepath.Join(dir, "config.toml")
    require.NoError(t, os.WriteFile(path, []byte("name = 'test'"), 0o600))

    // Use path...
}
```

Tests must not depend on:

- The current time. Use a fixed time or `synctest`.
- Random values. Use a fixed seed.
- Network availability. Use a local server or a mock.
- Filesystem state outside `testdata/`, `t.TempDir()`, or `t.ArtifactDir()`.

## Example Tests

Every public API needs at least one runnable example. Place examples in `example_test.go` using the external package.

```go
package nabat_test

import (
    "bytes"
    "context"
    "fmt"
    "strings"

    "nabat.dev/nabat"
    "nabat.dev/nabattest"
)

func ExampleNew() {
    app, err := nabat.New("myctl")
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    if app == nil {
        fmt.Println("nil")
        return
    }
    fmt.Println("ready")
    // Output:
    // ready
}

func Example() {
    var out bytes.Buffer
    app, _ := nabat.New("myctl",
        nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
    )
    app.MustCommand("hello",
        nabat.WithArg("name", "world"),
        nabat.WithRun(func(c *nabat.Context) error {
            name, _ := nabat.BindAs[string](c, "name")
            c.Print(name)
            return nil
        }),
    )
    _ = app.RunArgs(context.Background(), "hello")
    fmt.Print(out.String())
    // Output:
    // world
}
```

### Example guidelines

- Use the external package `nabat_test`.
- The function name starts with `Example`.
- Add an `// Output:` comment so `go test` checks the output.
- Use `fmt.Println` or `fmt.Print` for the comparison.
- Examples cannot use `*testing.T`, so use `_ = ...` for errors that must be ignored. Do not call `nabattest.Run` from an example; call `app.RunArgs` directly.

## Benchmarks

Place benchmarks in `*_bench_test.go` files for code that runs often.

```go
package nabat

import (
    "bytes"
    "strings"
    "testing"
)

func BenchmarkRun(b *testing.B) {
    var out bytes.Buffer
    app := MustNew("bench", WithIO(NewIO(strings.NewReader(""), &out, &out)))
    app.MustCommand("noop", WithRun(func(c *Context) error { return nil }))

    b.ReportAllocs()

    for b.Loop() {
        out.Reset()
        _ = app.RunArgs(b.Context(), "noop")
    }
}

func BenchmarkRun_Parallel(b *testing.B) {
    b.ReportAllocs()

    b.RunParallel(func(pb *testing.PB) {
        var out bytes.Buffer
        app := MustNew("bench", WithIO(NewIO(strings.NewReader(""), &out, &out)))
        app.MustCommand("noop", WithRun(func(c *Context) error { return nil }))

        for pb.Next() {
            out.Reset()
            _ = app.RunArgs(b.Context(), "noop")
        }
    })
}
```

### Benchmark guidelines

- Use `b.Loop()` for sequential benchmarks. It calls `b.ResetTimer()` and `b.StartTimer()` for you and does not block inlining.
- Use `b.ReportAllocs()` to track allocations.
- Use `b.Context()` instead of `context.Background()`.
- Provide both sequential and parallel variants for code that runs often.
- Compare runs with `benchstat` (`golang.org/x/perf/cmd/benchstat`) to confirm a change is real.

## Test Data and Golden Files

### The testdata Directory

Go ignores `testdata/` during `go build`. Use it for fixtures and golden files:

```text
nabat/
├── app.go
├── app_test.go
└── testdata/
    ├── fixtures/
    │   └── valid_config.json
    └── golden/
        └── help_output.txt
```

For fixtures that need many small files, use `golang.org/x/tools/txtar`. A `.txtar` file holds many files in one archive, which keeps reviews short and diffs clear.

### Golden File Testing

Use golden files when the test compares formatted output (help text, version output, structured data).

Wrap the compare-or-update logic in one helper:

```go
import (
    "os"
    "path/filepath"
    "testing"

    "github.com/google/go-cmp/cmp"
    "github.com/stretchr/testify/require"
)

func assertGolden(t *testing.T, path string, got []byte) {
    t.Helper()

    if os.Getenv("UPDATE_GOLDEN") == "1" {
        require.NoError(t, os.WriteFile(path, got, 0o600))
        return
    }

    want, err := os.ReadFile(path)
    require.NoError(t, err)

    if diff := cmp.Diff(string(want), string(got)); diff != "" {
        actual := filepath.Join(t.ArtifactDir(), filepath.Base(path)+".actual")
        _ = os.WriteFile(actual, got, 0o600)
        t.Fatalf("golden mismatch (-want +got):\n%s\nactual saved to %s", diff, actual)
    }
}
```

Use it in a test:

```go
func TestHelpOutput_Golden(t *testing.T) {
    t.Parallel()

    var out bytes.Buffer
    app := nabat.MustNew("myctl",
        nabat.WithDescription("My CLI tool"),
        nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
    )
    app.MustCommand("deploy", nabat.WithDescription("Deploy the application"))

    _ = nabattest.Run(t, app, []string{"--help"})

    assertGolden(t, "testdata/golden/help_output.txt", out.Bytes())
}
```

Update golden files when the output changes on purpose:

```bash
UPDATE_GOLDEN=1 go test ./... -run TestHelpOutput_Golden
```

The env-var pattern (instead of a `flag.Bool`) avoids conflicts between subpackages and works with any `-run` filter.

## Test Coverage

Nabat is a multi-package module. Coverage applies to all packages.

| Target | Minimum | Goal |
|--------|---------|------|
| Per-package coverage | 80% | 90% |

Measure coverage:

```bash
# Coverage summary across all packages
go test -covermode=atomic -coverpkg=./... -cover ./...

# Detailed HTML report
go test -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Coverage by function
go tool cover -func=coverage.out
```

`-covermode=atomic` is required when you also use `-race`. `-coverpkg=./...` makes coverage span every package, not only the one under test, so cross-package usage counts.

## Best Practices

1. **Run tests in parallel.** Call `t.Parallel()` at the top of every test that does not call `t.Setenv` or `t.Chdir`. `testing.AllocsPerRun` panics when called while parallel tests are running, so put any allocation test in its own non-parallel test.

2. **Use testify and cmp.Diff.** Use `testify/assert` and `testify/require` for most checks. Use `cmp.Diff` for nested struct comparison. Never write `if got != want { t.Errorf(...) }`.

3. **Keep tests independent.** Do not share state. Use `t.Setenv` for env, `t.Chdir` for working directory, `t.TempDir` for files.

4. **Clean up with t.Cleanup.** Use `t.Cleanup` instead of `defer` for resources owned by the test:

    ```go
    func TestWithTempFile(t *testing.T) {
        t.Parallel()

        dir := t.TempDir()
        path := filepath.Join(dir, "config.toml")
        require.NoError(t, os.WriteFile(path, nil, 0o600))
        t.Cleanup(func() { /* extra cleanup if needed */ })
    }
    ```

5. **Make tests deterministic.** Do not depend on real time, random values, network, or filesystem state outside `testdata/`, `t.TempDir()`, or `t.ArtifactDir()`.

6. **Write descriptive names.** Test names and subtest names are documentation. Read like a sentence.

7. **Use the race detector.** Run `go test -race -shuffle=on` in CI. The race detector catches data races; `-shuffle=on` catches tests that depend on order.

8. **Use the canonical helpers.** Use `nabat.MustNew` for app construction and `nabattest.Run` for execution. Pass `nabattest.WithContext(ctx)` only when you need a custom context.

9. **Capture I/O with `nabattest.NewIO()`.** When a test inspects output, build the bundle with `nabattest.NewIO()` so stdout and stderr land in independent buffers. Semantic helpers (`Success`, `Info`, `Warn`, `Error`) and progress UI write to stderr; `Print`, `Println`, `Printf`, and structured output (`Table`, `JSON`, `YAML`, `TOML`, `Tree`) write to stdout. Asserting against the wrong buffer is the most common way to silently mask a stream-routing regression.

## Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection and shuffled order
go test -race -shuffle=on ./...

# Run with coverage
go test -covermode=atomic -coverpkg=./... -cover ./...

# Run a specific test
go test -run TestArgResolvesFromEnv ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Run fuzz tests
go test -fuzz=FuzzRunArgs -fuzztime=30s ./...

# Run tests with a timeout
go test -timeout 5m ./...
```

### CI commands

```bash
# Unit tests with race detection, shuffled order, and coverage
go test -race -shuffle=on -covermode=atomic -coverpkg=./... -coverprofile=coverage.out -timeout 10m ./...
```
