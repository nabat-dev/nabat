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

// Package nabat provides a CLI framework for Go built on Cobra.
//
// # Core concepts
//
// An [App] is the root of a CLI. Add subcommands with [App.Command], install
// extensions with [WithExtension], enable the built-in version feature with
// [WithVersion], configure the root by passing [RootOption] values directly to
// [New], and run the tree with [App.Run].
//
// A [Command] wraps a single [github.com/spf13/cobra.Command] and Nabat metadata
// (positional args, flags, hooks). Nested commands use [Command.Command] (returns
// `(*Command, error)`) or [Command.MustCommand] (panics on failure; suitable for
// `main()` chains). For aggregated registration errors across multiple commands,
// declare the entire tree with [WithCommand] inside [New]; every problem
// aggregates into the [*ConfigErrors] returned by [New]. There is no deferred
// error path.
//
// Each invoked handler receives a [Context] with resolved args and flags.
// Handlers are [RunFunc] values set with [WithRun]. [Context] implements
// [context.Context] for deadlines and cancellation.
//
// Positional args resolve in order: CLI arg, environment variable (only if
// [WithEnv] is set), interactive prompt (TTY only, only if [WithPrompt] is
// attached), then the typed default passed to the constructor. Flags resolve:
// CLI flag, environment variable (only if [WithEnv] is set), then default.
// All names passed to [WithEnv] are combined with the app env prefix.
// Use [WithEnvAlias] for verbatim env names that must not receive the prefix.
// Define args with [WithArg] and related helpers; define flags with [WithFlag]
// and related helpers.
//
// # Quick start
//
// Construct an app and its command tree in one call, then run it:
//
//	app, err := nabat.New("myctl",
//	    nabat.WithVersion("0.1.0"),
//	    nabat.WithCompletion(),
//	    nabat.WithExtension(manpage.New()),
//
//	    nabat.WithCommand("deploy",
//	        nabat.WithDescription("Deploy application"),
//	        nabat.WithSelectArg("env", "", []string{"staging", "production"},
//	            nabat.WithRequired(),
//	            nabat.WithPrompt("Target environment", ""),
//	        ),
//	        nabat.WithFlag("replicas", 3,
//	            nabat.WithEnv("replicas"),
//	            nabat.WithUsage("Number of replicas"),
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
//
//	if err := app.Run(context.Background()); err != nil {
//	    os.Exit(1)
//	}
//
// On failure, [App.Run] prints a styled error and a "run --help" hint to stderr
// before returning the error, unless you override rendering with
// [WithErrorHandler].
//
// Help has a two-axis design. The persistent `--help` (`-h`) flag is on by
// default for every app (GNU/POSIX convention) and is configured via
// [WithHelpFlagName], [WithHelpShorthand], [WithoutHelpFlag], and
// [WithoutHelpShorthand]. The `help <subcmd>` subcommand is opt-in via
// [WithHelpCommand] and [WithHelpCommandName] (mirroring [WithVersion]).
// Use [WithoutHelp] to opt out of the entire feature and let Cobra's
// defaults take over.
//
// Version is opt-in: pass [WithVersion] to install a `version` subcommand and
// `--version`/`-v` flag with themed output. Customize via the nested
// [VersionOption] family ([WithVersionCommandName], [WithVersionFlagName],
// [WithVersionShorthand], etc.); omit [WithVersion] entirely to install nothing.
//
// Shell completion is opt-in: pass [WithCompletion] to install a `completion`
// subcommand that emits bash, zsh, fish, and PowerShell scripts. Customize
// via the nested [CompletionOption] family ([WithCompletionName],
// [WithCompletionHidden], [WithCompletionShells]); omit [WithCompletion]
// entirely to install nothing. Per-flag and per-positional dynamic completion
// candidates are wired with [WithCompleter] and [WithPositionalCompleter] and
// work whether or not [WithCompletion] is enabled.
//
// # Interactive forms
//
// [Context.Form] collects multiple values in one typed form. Field constructors
// ([WithFormField], [WithSelectField], [WithMultiSelectField]) satisfy both
// [FormOption] and [GroupOption], so they slot into either [Context.Form]
// directly or inside [WithFormGroup] (multi-page forms):
//
//	// Single-page form
//	c.Form(
//	    nabat.WithFormField(&name, "Name", "", nabat.WithDefault("alice")),
//	    nabat.WithSelectField(&env, "Environment", "",
//	        []string{"staging", "production"}, "staging"),
//	)
//
//	// Multi-page wizard
//	c.Form(
//	    nabat.WithFormGroup(
//	        nabat.WithGroupTitle("Identity"),
//	        nabat.WithFormField(&name, "Name", "", nabat.WithDefault("")),
//	    ),
//	    nabat.WithFormGroup(
//	        nabat.WithGroupTitle("Deployment"),
//	        nabat.WithSelectField(&env, "Environment", "",
//	            []string{"staging", "production"}, "staging"),
//	    ),
//	)
//
// [WithDefault] sets the non-interactive fallback value; T is inferred from the
// value. Select fields use the positional defaultVal parameter instead, which
// keeps compile-time E-type checking. [WithFormNote] adds display-only
// instructional text. [WithOptionsFunc] provides dynamic select choices.
// [WithFormAccessible] enables screen-reader-friendly mode.
//
// # Examples
//
// See the examples/ directory for complete programs. Runnable godoc examples are
// in example_test.go.
package nabat
