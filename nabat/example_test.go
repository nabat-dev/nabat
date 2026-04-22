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

package nabat_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"nabat.dev/nabat"
	"nabat.dev/nabattest"
	"nabat.dev/theme"
)

func Example() {
	var out bytes.Buffer
	app, err := nabat.New("myctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	app.MustCommand("hello",
		nabat.WithArg("name", "world"),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Name string `nabat:"name"`
			}
			if bindErr := c.Bind(&args); bindErr != nil {
				return bindErr
			}
			c.Print(args.Name)
			return nil
		}),
	)
	if err = app.RunArgs(context.Background(), "hello"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(out.String())
	// Output:
	// world
}

// Example_themeAndOverride shows the canonical end-to-end shape: pick
// a built-in theme by name, optionally tweak one slot via
// [WithThemeOverride], then read styles back through [App.Theme] in a
// command's run handler. The tweaked StatusError color flows through
// every output path that consumes that token (Context.Error, the
// "error:" prefix on uncaught errors, etc.).
func Example_themeAndOverride() {
	io, _, out, _ := nabattest.NewIO()

	app, err := nabat.New("myctl",
		nabat.WithIO(io),
		nabat.WithTheme(theme.Minimal),
		nabat.WithThemeOverride(theme.StatusError, lipgloss.NewStyle().Bold(true)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	// The resolved theme is a snapshot of every styled slot; consumers
	// (Context.Success, Context.Error, the help renderer, the logging
	// extension) all read through the same accessor.
	rt := app.Theme()
	fmt.Println("name:", rt.Name())
	fmt.Println("variant:", rt.Variant())
	fmt.Println("status.error bold:", rt.Style(theme.StatusError).GetBold())
	_ = out
	// Output:
	// name: minimal
	// variant: notty
	// status.error bold: true
}

func ExampleNew() {
	app, err := nabat.New("myctl")
	if err != nil {
		fmt.Println(err)
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

func ExampleNew_nilOption() {
	_, err := nabat.New("myctl", nil)
	fmt.Println(err != nil)
	// Output:
	// true
}

func ExampleMustNew() {
	app := nabat.MustNew("demo")
	fmt.Println(app != nil)
	// Output:
	// true
}

func ExampleContext_Bind() {
	var out bytes.Buffer
	app := nabat.MustNew("bindctl", nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("show",
		nabat.WithFlag("region", "us-east"),
		nabat.WithRun(func(c *nabat.Context) error {
			var cfg struct {
				Region string `nabat:"region"`
			}
			if err := c.Bind(&cfg); err != nil {
				return err
			}
			c.Print(cfg.Region)
			return nil
		}),
	)
	if err := app.RunArgs(context.Background(), "show"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(out.String())
	// Output:
	// us-east
}

func ExampleApp_Command() {
	var out bytes.Buffer
	app, err := nabat.New("shipctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	app.MustCommand("status",
		nabat.WithDescription("Show deployment status"),
		nabat.WithFlag("env", "staging"),
		nabat.WithRun(func(c *nabat.Context) error {
			env, bindErr := nabat.BindAs[string](c, "env")
			if bindErr != nil {
				return bindErr
			}
			c.Print("env=" + env)
			return nil
		}),
	)
	if err = app.RunArgs(context.Background(), "status", "--env", "production"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// env=production
}

func ExampleApp_Run() {
	var out bytes.Buffer
	app := nabat.MustNew("runctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	app.MustCommand("ping",
		nabat.WithRun(func(c *nabat.Context) error {
			c.Print("pong")
			return nil
		}),
	)
	// In a real program, App.Run reads os.Args. RunArgs is used here so the
	// example is self-contained and runnable by go test.
	if err := app.RunArgs(context.Background(), "ping"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// pong
}

func ExampleApp_RunArgs() {
	var out bytes.Buffer
	app := nabat.MustNew("argctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	app.MustCommand("echo",
		nabat.WithFlag("msg", "default"),
		nabat.WithRun(func(c *nabat.Context) error {
			msg, err := nabat.BindAs[string](c, "msg")
			if err != nil {
				return err
			}
			c.Print(msg)
			return nil
		}),
	)
	if err := app.RunArgs(context.Background(), "echo", "--msg", "hello"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// hello
}

func ExampleBindAs_string() {
	var out bytes.Buffer
	app := nabat.MustNew("getctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	app.MustCommand("show",
		nabat.WithFlag("label", "default"),
		nabat.WithRun(func(c *nabat.Context) error {
			label, err := nabat.BindAs[string](c, "label")
			if err != nil {
				return err
			}
			c.Print(label)
			return nil
		}),
	)
	if err := app.RunArgs(context.Background(), "show", "--label", "from-flag"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// from-flag
}

// ExampleContext_output demonstrates semantic output helpers.
// stdout and stderr share one buffer here because go test's // Output:
// directive captures only stdout; see [nabattest.NewIO] for tests that
// must keep the streams separate.
func ExampleContext_output() {
	var out bytes.Buffer
	app := nabat.MustNew("outctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	app.MustCommand("report",
		nabat.WithRun(func(c *nabat.Context) error {
			c.Success("done", "id", 42)
			c.Warn("late", "ms", 9)
			return nil
		}),
	)
	if err := app.RunArgs(context.Background(), "report"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(out.String())
	// Output:
	// ✓ done id=42
	// ⚠ late ms=9
}

func ExampleContext_Encode() {
	var out bytes.Buffer
	app := nabat.MustNew("encctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	app.MustCommand("dump",
		nabat.WithRun(func(c *nabat.Context) error {
			v := struct {
				Hello string `json:"hello"`
			}{Hello: "world"}
			return c.Encode(&v, nabat.FormatJSON)
		}),
	)
	if err := app.RunArgs(context.Background(), "dump"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// {
	//   "hello": "world"
	// }
}

func ExampleWithEnv() {
	var out bytes.Buffer
	app, err := nabat.New("envex",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	app.MustCommand("region",
		nabat.WithFlag("region", "", nabat.WithEnv("region")),
		nabat.WithRun(func(c *nabat.Context) error {
			region, bindErr := nabat.BindAs[string](c, "region")
			if bindErr != nil {
				return bindErr
			}
			c.Print(region)
			return nil
		}),
	)
	if err = nabattest.Run(nil, app, []string{"region"}, nabattest.WithEnvVars(map[string]string{
		"ENVEX_REGION": "eu-west-1",
	})); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// eu-west-1
}

func ExampleNew_rootCommand() {
	var out bytes.Buffer
	app, err := nabat.New("rootctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
		nabat.WithDescription("demo root"),
		nabat.WithRun(func(c *nabat.Context) error {
			c.Print("root-run")
			return nil
		}),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = app.RunArgs(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// root-run
}

func ExampleWithTheme() {
	_, err := nabat.New("themectl", nabat.WithTheme(theme.Dracula))
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleErrHandled() {
	var out bytes.Buffer
	app := nabat.MustNew("hookctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
	)
	app.MustCommand("status",
		nabat.WithPreRun(func(c *nabat.Context) error {
			c.Println("served by pre-run hook")
			// Skip the command handler and return nil from App.Run.
			return nabat.ErrHandled
		}),
		nabat.WithRun(func(c *nabat.Context) error {
			c.Println("never runs")
			return nil
		}),
	)
	if err := app.RunArgs(context.Background(), "status"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// served by pre-run hook
}

func ExampleConfigErrors() {
	_, err := nabat.New("")
	if err == nil {
		fmt.Println("unexpected")
		return
	}
	var ce *nabat.ConfigErrors
	if !errors.As(err, &ce) || len(ce.Unwrap()) == 0 {
		fmt.Println("not ConfigErrors")
		return
	}
	fmt.Println(ce.Unwrap()[0])
	// Output:
	// nabat: app name cannot be empty
}

func ExampleWithHidden() {
	app := nabat.MustNew("hidectl")
	_, err := app.Command("secret",
		nabat.WithHidden(),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithDeprecated() {
	app := nabat.MustNew("depctl")
	_, err := app.Command("legacy",
		nabat.WithDeprecated("use `new` instead"),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithTypoHints() {
	app := nabat.MustNew("hintctl")
	_, err := app.Command("migrate",
		nabat.WithTypoHints("migr", "migration"),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithAnnotation() {
	app := nabat.MustNew("annctl")
	_, err := app.Command("pods",
		nabat.WithAnnotation("sample", "value"),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithPositionalCompleter() {
	app := nabat.MustNew("compctl")
	_, err := app.Command("env",
		nabat.WithPositionalCompleter(func(args []string, toComplete string) ([]string, nabat.CompletionDirective) {
			return []string{"staging", "prod"}, nabat.CompletionDefault
		}),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithDeprecated_flag() {
	app := nabat.MustNew("flagdep")
	_, err := app.Command("run",
		nabat.WithFlag("legacy", "", nabat.WithDeprecated("use --new instead")),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithCompleter() {
	app := nabat.MustNew("fcctl")
	_, err := app.Command("use",
		nabat.WithFlag("cluster", "", nabat.WithCompleter(
			func(args []string, toComplete string) ([]string, nabat.CompletionDirective) {
				return []string{"eu-1"}, nabat.CompletionDefault
			})),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithCompletion() {
	app, err := nabat.New("ctl",
		nabat.WithCompletion(),
	)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	found := false
	for _, sub := range app.UnsafeRoot().Commands() {
		if sub.Name() == "completion" {
			found = true
			break
		}
	}
	fmt.Println(found)
	// Output:
	// true
}

func ExampleWithParseOptions_allowUnknownFlags() {
	app := nabat.MustNew("popctl")
	_, err := app.Command("proxy",
		nabat.WithParseOptions(nabat.WithAllowUnknownFlags()),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithParseOptions_traverseChildren() {
	app := nabat.MustNew("travctl")
	_, err := app.Command("parent",
		nabat.WithParseOptions(nabat.WithTraverseChildren(true)),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithParseOptions_disableFlagParsing() {
	app := nabat.MustNew("dfpctl")
	_, err := app.Command("raw",
		nabat.WithParseOptions(nabat.WithDisableFlagParsing()),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithArgArity_exactCount() {
	app := nabat.MustNew("arityctl")
	_, err := app.Command("copy",
		nabat.WithArg("src", "", nabat.WithRequired()),
		nabat.WithArg("dst", "", nabat.WithRequired()),
		nabat.WithArgArity(nabat.WithExactArgCount(2)),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithArgArity_minMax() {
	app := nabat.MustNew("mmctl")
	_, err := app.Command("pick",
		nabat.WithArg("one", ""),
		nabat.WithArg("two", ""),
		nabat.WithArgArity(nabat.WithMinArgCount(1), nabat.WithMaxArgCount(2)),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleWithDeprecatedShorthand() {
	app := nabat.MustNew("shctl")
	_, err := app.Command("run",
		nabat.WithFlag("cfg", "", nabat.WithShort('c'), nabat.WithDeprecatedShorthand("use --cfg")),
		nabat.WithRun(func(c *nabat.Context) error { return nil }),
	)
	fmt.Println(err == nil)
	// Output:
	// true
}

func ExampleApp_UnsafeRoot() {
	app := nabat.MustNew("rootacc")
	fmt.Println(app.UnsafeRoot() != nil)
	// Output:
	// true
}

func ExampleCommand_UnsafeCobra() {
	cmd := nabat.MustNew("cobacc").MustCommand("leaf", nabat.WithRun(func(c *nabat.Context) error { return nil }))
	fmt.Println(cmd.UnsafeCobra() != nil)
	// Output:
	// true
}

// Single-command CLI declared with the flat declarative form: root config and
// the run handler all live as direct options on nabat.New, no separate
// subcommand needed.
func ExampleNew_singleCommand() {
	var out bytes.Buffer
	app, err := nabat.New("greet",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
		nabat.WithDescription("Print a friendly greeting"),
		nabat.WithArg("name", "world"),
		nabat.WithRun(func(c *nabat.Context) error {
			name, err := nabat.BindAs[string](c, "name")
			if err != nil {
				return err
			}
			c.Print("hello, " + name)
			return nil
		}),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = app.RunArgs(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// hello, world
}

// Multi-command tree using nabat.WithCommand. Every error from one New() call.
func ExampleWithCommand() {
	var out bytes.Buffer
	app, err := nabat.New("myctl",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
		nabat.WithDescription("My CLI"),
		nabat.WithCommand("up", nabat.WithRun(func(c *nabat.Context) error {
			c.Print("up")
			return nil
		})),
		nabat.WithCommand("down", nabat.WithRun(func(c *nabat.Context) error {
			c.Print("down")
			return nil
		})),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = app.RunArgs(context.Background(), "up"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// up
}

// Nested subcommand tree: nabat.WithCommand inside another nabat.WithCommand.
func ExampleWithCommand_nested() {
	var out bytes.Buffer
	app, err := nabat.New("kubectl-myext",
		nabat.WithIO(nabat.NewIO(strings.NewReader(""), &out, &out)),
		nabat.WithTheme(theme.Minimal),
		nabat.WithCommand("cluster",
			nabat.WithDescription("Cluster management"),
			nabat.WithCommand("scale", nabat.WithRun(func(c *nabat.Context) error {
				c.Print("scaled")
				return nil
			})),
			nabat.WithCommand("status", nabat.WithRun(func(c *nabat.Context) error {
				c.Print("ok")
				return nil
			})),
		),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = app.RunArgs(context.Background(), "cluster", "scale"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// scaled
}

// ExampleArgOptions shows bundling multiple arg options for reuse.
func ExampleArgOptions() {
	io, _, out, _ := nabattest.NewIO()
	app, err := nabat.New("myctl", nabat.WithIO(io),
		nabat.WithCommand("deploy",
			nabat.WithArg("env", "staging",
				nabat.ArgOptions(
					nabat.WithUsage("deployment environment"),
					nabat.WithEnv("env"),
				),
			),
			nabat.WithRun(func(c *nabat.Context) error {
				v, err := nabat.BindAs[string](c, "env")
				if err != nil {
					return err
				}
				c.Print(v)
				return nil
			}),
		),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = app.RunArgs(context.Background(), "deploy"); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// staging
}

// ExampleAsExtension wraps a function as an extension; Init runs during New
// after help, version, and completion are registered.
func ExampleAsExtension() {
	var ran bool
	io, _, out, _ := nabattest.NewIO()
	app, err := nabat.New("myctl", nabat.WithIO(io),
		nabat.AsExtension("probe", func(a nabat.AppSurface) error {
			ran = true
			return a.OnPreRun(func(c *nabat.Context) error { return nil })
		}),
		nabat.WithRun(func(c *nabat.Context) error {
			c.Print("done")
			return nil
		}),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !ran {
		fmt.Println("init did not run")
		return
	}
	if err = app.RunArgs(context.Background()); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Print(strings.TrimSpace(out.String()))
	// Output:
	// done
}
