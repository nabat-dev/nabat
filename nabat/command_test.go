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

package nabat

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandRequiresName(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("", WithRun(func(c *Context) error { return nil }))
	require.Error(t, err)
}

func TestNilCommandHooksRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "WithRun rejects nil function",
			run: func(t *testing.T) {
				t.Helper()
				app := MustNew("test")
				_, err := app.newCommand(app.root, "x", WithRun(nil))
				require.Error(t, err)
			},
		},
		{
			name: "WithPreRun rejects nil function",
			run: func(t *testing.T) {
				t.Helper()
				app := MustNew("test")
				_, err := app.Command("run", WithPreRun(nil), WithRun(func(c *Context) error { return nil }))
				require.Error(t, err)
			},
		},
		{
			name: "WithPostRun rejects nil function",
			run: func(t *testing.T) {
				t.Helper()
				app := MustNew("test")
				_, err := app.Command("run", WithPostRun(nil), WithRun(func(c *Context) error { return nil }))
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestBuildUseString(t *testing.T) {
	t.Parallel()

	t.Run("required input", func(t *testing.T) {
		t.Parallel()

		use := buildUseString("deploy", []argDef{
			{name: "environment", config: fieldConfig{required: true}},
		}, nil)
		require.Equal(t, "deploy <environment>", use)
	})

	t.Run("optional input", func(t *testing.T) {
		t.Parallel()

		use := buildUseString("deploy", []argDef{
			{name: "tag", config: fieldConfig{required: false}},
		}, nil)
		require.Equal(t, "deploy [tag]", use)
	})

	t.Run("mixed inputs", func(t *testing.T) {
		t.Parallel()

		use := buildUseString("deploy", []argDef{
			{name: "environment", config: fieldConfig{required: true}},
			{name: "tag", config: fieldConfig{required: false}},
		}, nil)
		require.Equal(t, "deploy <environment> [tag]", use)
	})

	t.Run("with passthrough", func(t *testing.T) {
		t.Parallel()

		use := buildUseString("exec", []argDef{
			{name: "service", config: fieldConfig{required: true}},
		}, &passthroughDef{label: "command [args...]"})
		require.Equal(t, "exec <service> [-- command [args...]]", use)
	})

	t.Run("passthrough only", func(t *testing.T) {
		t.Parallel()

		use := buildUseString("run", nil, &passthroughDef{label: "args..."})
		require.Equal(t, "run [-- args...]", use)
	})
}

func TestNewCommandGeneratesUseFromInputs(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	cmd := app.MustCommand("deploy",
		WithArg("environment", "", WithRequired()),
		WithArg("tag", ""),
	)
	require.Equal(t, "deploy <environment> [tag]", cmd.cobra.Use)
}

func TestWithExampleAppearsInHelp(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("myctl", WithIO(io))
	app.MustCommand("deploy",
		WithDescription("Deploy app"),
		WithExample("# Deploy to production:\nmyctl deploy production\n"),
		WithRun(func(c *Context) error { return nil }),
	)

	require.NoError(t, Run(t, app, []string{"deploy", "--help"}))
	got := out.String()
	assert.Contains(t, got, "Examples:")
	assert.Contains(t, got, "Deploy to production")
}

func TestWithExampleNilOptionRejected(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("run",
		CommandOption(nil),
	)
	require.Error(t, err)
}

func TestWithValidationRunsAfterResolution(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithFlag("format", "text"),
		WithFlag("output", ""),
		WithValidation(func(c *Context) error {
			format, err := BindAs[string](c, "format")
			if err != nil {
				return err
			}
			if format == "json" && !c.Explicit("output") {
				return errors.New("--output required when --format=json")
			}
			return nil
		}),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "--format=json"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--output required")
}

func TestWithValidationPassesWhenValid(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithFlag("format", "text"),
		WithFlag("output", ""),
		WithValidation(func(c *Context) error {
			format, err := BindAs[string](c, "format")
			if err != nil {
				return err
			}
			if format == "json" && !c.Explicit("output") {
				return errors.New("--output required when --format=json")
			}
			return nil
		}),
		WithRun(func(c *Context) error { return nil }),
	)

	require.NoError(t, Run(t, app, []string{"deploy", "--format=json", "--output=out.json"}))
}

func TestWithValidationMultipleAllMustPass(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithFlag("a", "ok"),
		WithFlag("b", "ok"),
		WithValidation(func(c *Context) error {
			a, err := BindAs[string](c, "a")
			if err != nil {
				return err
			}
			if a == "bad" {
				return errors.New("a is bad")
			}
			return nil
		}),
		WithValidation(func(c *Context) error {
			b, err := BindAs[string](c, "b")
			if err != nil {
				return err
			}
			if b == "bad" {
				return errors.New("b is bad")
			}
			return nil
		}),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"run", "--b=bad"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "b is bad")
}

func TestWithValidationNilFunctionRejected(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("run",
		WithValidation(nil),
		WithRun(func(c *Context) error { return nil }),
	)
	require.Error(t, err)
}

func TestWithPreRunExecutesBeforeHandler(t *testing.T) {
	t.Parallel()

	var order []string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithPreRun(func(c *Context) error {
			order = append(order, "pre")
			return nil
		}),
		WithRun(func(c *Context) error {
			order = append(order, "run")
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, []string{"pre", "run"}, order)
}

func TestWithPreRunAbortsPipelineOnError(t *testing.T) {
	t.Parallel()

	var ran bool
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithPreRun(func(c *Context) error {
			return errors.New("auth failed")
		}),
		WithRun(func(c *Context) error {
			ran = true
			return nil
		}),
	)
	err := Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "auth failed")
	assert.False(t, ran, "handler must not run when pre-run fails")
}

func TestWithPostRunExecutesAfterHandler(t *testing.T) {
	t.Parallel()

	var order []string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithRun(func(c *Context) error {
			order = append(order, "run")
			return nil
		}),
		WithPostRun(func(c *Context) error {
			order = append(order, "post")
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, []string{"run", "post"}, order)
}

func TestWithPostRunExecutesEvenWhenHandlerFails(t *testing.T) {
	t.Parallel()

	var postRan bool
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithRun(func(c *Context) error {
			return errors.New("handler error")
		}),
		WithPostRun(func(c *Context) error {
			postRan = true
			return nil
		}),
	)
	err := Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
	assert.True(t, postRan, "post-run must execute even when handler fails")
}

func TestWithPostRunErrorReturnedOnlyWhenHandlerSucceeds(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithRun(func(c *Context) error {
			return errors.New("handler error")
		}),
		WithPostRun(func(c *Context) error {
			return errors.New("post error")
		}),
	)
	// Handler error takes priority; post error is suppressed.
	err := Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
	assert.NotContains(t, err.Error(), "post error")
}

func TestWithPostRunErrorSurfacedWhenHandlerSucceeds(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithRun(func(c *Context) error { return nil }),
		WithPostRun(func(c *Context) error {
			return errors.New("cleanup failed")
		}),
	)
	err := Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleanup failed")
}

func TestMultiplePreRunsExecuteInOrder(t *testing.T) {
	t.Parallel()

	var order []string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithPreRun(func(c *Context) error { order = append(order, "pre1"); return nil }),
		WithPreRun(func(c *Context) error { order = append(order, "pre2"); return nil }),
		WithRun(func(c *Context) error { order = append(order, "run"); return nil }),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, []string{"pre1", "pre2", "run"}, order)
}

func TestCommandReturnsError(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("", WithRun(func(c *Context) error { return nil }))
	require.Error(t, err)
}

func TestCommandSuccess(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	cmd := app.MustCommand("hello", WithRun(func(c *Context) error { return nil }))
	require.NotNil(t, cmd)
}

func TestChildCommandReturnsError(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	parent := app.MustCommand("parent", WithDescription("parent"))
	_, err := parent.Command("", WithRun(func(c *Context) error { return nil }))
	require.Error(t, err)
}

func TestInputFlagNameCollision(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("deploy",
		WithArg("target", ""),
		WithFlag("target", ""),
		WithRun(func(c *Context) error { return nil }),
	)
	require.ErrorIs(t, err, ErrArgFlagNameCollision)
}

func TestNestedSubcommands(t *testing.T) {
	t.Parallel()

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))

	cluster := app.MustCommand("cluster", WithDescription("Cluster management"))
	cluster.MustCommand("create",
		WithArg("name", "default"),
		WithRun(func(c *Context) error {
			var args struct {
				Name string `nabat:"name"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Name
			return nil
		}),
	)

	err := Run(t, app, []string{"cluster", "create", "mycluster"})
	require.NoError(t, err)
	assert.Equal(t, "mycluster", got)
}

func TestNestedSubcommandDefault(t *testing.T) {
	t.Parallel()

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))

	cluster := app.MustCommand("cluster")
	cluster.MustCommand("create",
		WithArg("name", "default"),
		WithRun(func(c *Context) error {
			var args struct {
				Name string `nabat:"name"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Name
			return nil
		}),
	)

	err := Run(t, app, []string{"cluster", "create"})
	require.NoError(t, err)
	assert.Equal(t, "default", got)
}

func TestWithLongDescription(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	cmd := app.MustCommand("deploy",
		WithDescription("Deploy app"),
		WithLongDescription("Deploy the application to the specified environment with full control."),
		WithRun(func(c *Context) error { return nil }),
	)
	require.NotNil(t, cmd)
}

func TestWithAliases(t *testing.T) {
	t.Parallel()

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithAliases("d", "dep"),
		WithArg("env", "staging"),
		WithRun(func(c *Context) error {
			var args struct {
				Env string `nabat:"env"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Env
			return nil
		}),
	)

	err := Run(t, app, []string{"d"})
	require.NoError(t, err)
	assert.Equal(t, "staging", got)
}

func TestWithGroup(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithGroup("operations"),
		WithDescription("Deploy app"),
		WithRun(func(c *Context) error { return nil }),
	)
	app.MustCommand("status",
		WithGroup("operations"),
		WithDescription("Show status"),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
}

func TestCommandWithNoRunShowsHelp(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("myctl",
		WithIO(io),
	)
	app.MustCommand("cluster", WithDescription("Cluster management"))

	err := Run(t, app, []string{"cluster"})
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "myctl cluster")
	assert.Contains(t, got, "Cluster management")
}

func TestChildCommandEmptyNameReturnsError(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	parent := app.MustCommand("parent")
	child, err := parent.Command("")
	require.Nil(t, child, "Command returns nil on registration failure")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command name cannot be empty")
}

func TestChildCommandReturnsCommand(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	parent := app.MustCommand("parent")
	child := parent.MustCommand("child", WithDescription("a child"))
	require.NotNil(t, child)
}

func TestRootPreRunPostRunAndRunOrder(t *testing.T) {
	t.Parallel()

	var order []string
	io, _, _, _ := testIO()
	app := MustNew("r", WithIO(io),
		WithPreRun(func(c *Context) error {
			order = append(order, "pre")
			return nil
		}),
		WithPostRun(func(c *Context) error {
			order = append(order, "post")
			return nil
		}),
		WithRun(func(c *Context) error {
			order = append(order, "run")
			return nil
		}),
	)
	require.NoError(t, Run(t, app, nil))
	assert.Equal(t, []string{"pre", "run", "post"}, order)
}

func TestRootHooksWithoutRun(t *testing.T) {
	t.Parallel()

	var order []string
	io, _, _, _ := testIO()
	app := MustNew("r", WithIO(io),
		WithPreRun(func(c *Context) error {
			order = append(order, "pre")
			return nil
		}),
		WithPostRun(func(c *Context) error {
			order = append(order, "post")
			return nil
		}),
	)
	require.NoError(t, Run(t, app, nil))
	assert.Equal(t, []string{"pre", "post"}, order)
}

func TestHiddenCommandNotInParentHelp(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("h", WithIO(io), WithHelpCommand())
	app.MustCommand("public", WithDescription("visible"), WithRun(func(c *Context) error { return nil }))
	app.MustCommand("secret", WithHidden(), WithDescription("hidden"), WithRun(func(c *Context) error { return nil }))
	require.NoError(t, Run(t, app, []string{"help"}))
	got := out.String()
	assert.Contains(t, got, "public")
	assert.NotContains(t, got, "secret")
}

func TestDeprecatedCommandInHelp(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("d", WithIO(io), WithHelpCommand())
	app.MustCommand("old",
		WithDeprecated("use `new` instead"),
		WithRun(func(c *Context) error { return nil }),
	)
	require.NoError(t, Run(t, app, []string{"help", "old"}))
	assert.Contains(t, out.String(), "Deprecated:")
	assert.Contains(t, out.String(), "use `new` instead")
}

func TestWithParseOptionsAllowUnknownFlags(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("p", WithIO(io))
	app.MustCommand("proxy",
		WithParseOptions(WithAllowUnknownFlags()),
		WithRun(func(c *Context) error {
			c.Print("ok")
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"proxy", "--unknown-flag"}))
	assert.Contains(t, out.String(), "ok")
}

func TestWithArgArityExactArgs(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("c", WithIO(io))
	app.MustCommand("copy",
		WithArg("src", "", WithRequired()),
		WithArg("dst", "", WithRequired()),
		WithArgArity(WithExactArgCount(2)),
		WithRun(func(c *Context) error { return nil }),
	)
	err := Run(t, app, []string{"copy", "a"})
	require.Error(t, err)
	require.NoError(t, Run(t, app, []string{"copy", "a", "b"}))
}

func TestWithArgArityRequiresInnerOption(t *testing.T) {
	t.Parallel()

	app := MustNew("emptyarity")
	_, err := app.Command("x",
		WithArgArity(),
		WithRun(func(c *Context) error { return nil }),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least one ArityOption")
}

func TestWithArgArityMayOnlyBeUsedOnce(t *testing.T) {
	t.Parallel()

	app := MustNew("twice")
	_, err := app.Command("x",
		WithArgArity(WithExactArgCount(1)),
		WithArgArity(WithMaxArgCount(2)),
		WithArg("a", ""),
		WithRun(func(c *Context) error { return nil }),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithArgArity may only be used once")
}

func TestWithArgArityExactConflictsWithMin(t *testing.T) {
	t.Parallel()

	app := MustNew("conf")
	_, err := app.Command("x",
		WithArg("a", ""),
		WithArgArity(WithExactArgCount(1), WithMinArgCount(0)),
		WithRun(func(c *Context) error { return nil }),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithExactArgCount cannot be combined")
}

func TestAppRootAndCommandCobra(t *testing.T) {
	t.Parallel()

	app := MustNew("x")
	require.NotNil(t, app.UnsafeRoot())
	require.Equal(t, "x", app.UnsafeRoot().Name())

	cmd := app.MustCommand("leaf", WithRun(func(c *Context) error { return nil }))
	require.NotNil(t, cmd.UnsafeCobra())
	require.Equal(t, "leaf", cmd.UnsafeCobra().Name())
}

func TestWithArgCompletionsOverridesAuto(t *testing.T) {
	t.Parallel()

	app := MustNew("ac")
	cmd := app.MustCommand("env",
		WithSelectArg("target", "a", []string{"a", "b"}),
		WithPositionalCompleter(func(args []string, toComplete string) ([]string, CompletionDirective) {
			return []string{"custom"}, CompletionDefault
		}),
		WithRun(func(c *Context) error { return nil }),
	)
	fn := cmd.UnsafeCobra().ValidArgsFunction
	require.NotNil(t, fn)
	comps, _ := fn(cmd.UnsafeCobra(), nil, "")
	require.Equal(t, []string{"custom"}, comps)
}
