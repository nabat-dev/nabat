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
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

func TestNewDefaults(t *testing.T) {
	t.Parallel()

	app, err := New("testctl")
	require.NoError(t, err)
	require.NotNil(t, app)
}

func TestNewRejectsNilOption(t *testing.T) {
	t.Parallel()

	_, err := New("testctl", nil)
	require.Error(t, err)
}

func TestMustNewSucceeds(t *testing.T) {
	t.Parallel()

	app := MustNew("testctl")
	require.NotNil(t, app)
}

func TestMustNewPanics(t *testing.T) {
	t.Parallel()

	require.Panics(t, func() {
		MustNew("")
	})
}

func TestRunArgsWithSimpleCommand(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("testctl",
		WithIO(io),
	)

	app.MustCommand("hello",
		WithArg("name", "world"),
		WithRun(func(c *Context) error {
			var args struct {
				Name string `nabat:"name"`
			}
			require.NoError(t, c.Bind(&args))
			c.Info("hello", "name", args.Name)
			return nil
		}),
	)

	err := Run(t, app, []string{"hello"})
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "hello")
}

func TestRunArgsDoesNotLeakAcrossInvocations(t *testing.T) {
	t.Parallel()

	var calls [][]string
	io, _, _, _ := testIO()
	app := MustNew("testctl",
		WithIO(io),
	)
	app.MustCommand("echo",
		WithArg("msg", ""),
		WithRun(func(c *Context) error {
			calls = append(calls, c.Args())
			return nil
		}),
	)

	require.NoError(t, app.RunArgs(context.Background(), "echo", "first"))
	require.NoError(t, app.RunArgs(context.Background(), "echo", "second"))
	require.Len(t, calls, 2)
	require.Equal(t, []string{"first"}, calls[0])
	require.Equal(t, []string{"second"}, calls[1])
}

func TestRootDescription(t *testing.T) {
	t.Parallel()

	app, err := New("testctl",
		WithDescription("short root"),
		WithLongDescription("long root"),
	)
	require.NoError(t, err)
	assert.Equal(t, "short root", app.root.Short)
	assert.Equal(t, "long root", app.root.Long)
}

func TestRootFlag(t *testing.T) {
	t.Parallel()

	var got bool
	io, _, _, _ := testIO()
	app := MustNew("testctl",
		WithIO(io),
		WithFlag("verbose", false, WithShort('v'), WithPersistent()),
	)
	app.MustCommand("deploy",
		WithRun(func(c *Context) error {
			var args struct {
				Verbose bool `nabat:"verbose"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Verbose
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy", "--verbose"})
	require.NoError(t, err)
	assert.True(t, got)
}

func TestInheritedFlagsRenderShorthand(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("testctl",
		WithIO(io),
		WithFlag("verbose", false, WithShort('v'), WithUsage("Enable verbose logs"), WithPersistent()),
	)
	app.MustCommand("deploy",
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "--help"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Global Flags:")
	assert.Contains(t, out.String(), "--verbose, -v")
}

func TestHelpShowsDeprecatedLocalFlag(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("testctl", WithIO(io))
	app.MustCommand("run",
		WithFlag("format", "text",
			WithUsage("output format"),
			WithDeprecated("use --output instead"),
		),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"run", "--help"})
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "--format")
	assert.Contains(t, got, "(deprecated: use --output instead)")
}

func TestHelpShowsInheritedDeprecatedFlag(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("testctl",
		WithIO(io),
		WithFlag("legacy", false,
			WithUsage("legacy toggle"),
			WithDeprecated("remove in v2"),
			WithPersistent(),
		),
	)
	app.MustCommand("deploy",
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "--help"})
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "Global Flags:")
	assert.Contains(t, got, "(deprecated: remove in v2)")
}

func TestHelpShowsShorthandDeprecatedInDescription(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("testctl", WithIO(io))
	app.MustCommand("sync",
		WithFlag("file", "",
			WithShort('f'),
			WithUsage("input file"),
			WithDeprecatedShorthand("use --file instead of -f"),
		),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"sync", "--help"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "(shorthand deprecated: use --file instead of -f)")
}

func TestRootRun(t *testing.T) {
	t.Parallel()

	var called bool
	app := MustNew("testctl",
		WithRun(func(c *Context) error {
			called = true
			return nil
		}),
	)

	err := Run(t, app, nil)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRootFlagInputCollision(t *testing.T) {
	t.Parallel()

	_, err := New("testctl",
		WithArg("target", ""),
		WithFlag("target", ""),
	)
	require.ErrorIs(t, err, ErrArgFlagNameCollision)
}

func TestNewNilOption(t *testing.T) {
	t.Parallel()

	_, err := New("testctl",
		nil,
	)
	require.Error(t, err)
}

func TestRootInputsGenerateUse(t *testing.T) {
	t.Parallel()

	app := MustNew("testctl",
		WithArg("environment", "", WithRequired()),
		WithArg("tag", ""),
	)

	assert.Equal(t, "testctl <environment> [tag]", app.root.Use)
}

func TestRootDescriptionAppliesToRoot(t *testing.T) {
	t.Parallel()

	app, err := New("myctl", WithDescription("foo"))
	require.NoError(t, err)
	assert.Equal(t, "foo", app.root.Short)
}

func TestRootFlagAppliesToRoot(t *testing.T) {
	t.Parallel()

	app, err := New("myctl", WithFlag("v", false))
	require.NoError(t, err)
	require.NotNil(t, app.root.Flags().Lookup("v"), "root flag --v should be registered")
}

func TestRootRunAttachesHandler(t *testing.T) {
	t.Parallel()

	var called bool
	app := MustNew("myctl", WithRun(func(c *Context) error {
		called = true
		return nil
	}))
	require.NoError(t, Run(t, app, nil))
	assert.True(t, called)
}

func TestConfigErrorsUnwrapReturnsAllErrors(t *testing.T) {
	t.Parallel()

	var errs ConfigErrors
	errs.AddErr(errors.New("first issue"))
	errs.AddErr(errors.New("second issue"))
	unwrapped := errs.Unwrap()
	require.Len(t, unwrapped, 2)
	assert.Equal(t, "first issue", unwrapped[0].Error())
	assert.Equal(t, "second issue", unwrapped[1].Error())
}

func TestConfigErrorsUnwrapEmptyReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	var errs ConfigErrors
	assert.Empty(t, errs.Unwrap())
}

func TestConfigErrorsUnwrapReturnsCopy(t *testing.T) {
	t.Parallel()

	var errs ConfigErrors
	errs.AddErr(errors.New("only"))
	unwrapped := errs.Unwrap()
	unwrapped[0] = errors.New("mutated")
	// original must be unchanged
	assert.Equal(t, "only", errs.Unwrap()[0].Error())
}

func TestConfigErrorsAddErrPreservesSentinel(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("sentinel")
	var errs ConfigErrors
	errs.AddErr(sentinel)
	unwrapped := errs.Unwrap()
	require.Len(t, unwrapped, 1)
	assert.ErrorIs(t, unwrapped[0], sentinel)
}

func TestConfigErrorsErrorsIsWorksOnUnwrapped(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("my sentinel")
	var errs ConfigErrors
	errs.AddErr(sentinel)
	// errors.Is should find the sentinel inside ConfigErrors
	assert.ErrorIs(t, &errs, sentinel)
}

func TestAppRunExecutesApp(t *testing.T) {
	t.Parallel()

	var ran bool
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("hello", WithRun(func(c *Context) error {
		ran = true
		return nil
	}))
	app.root.SetArgs([]string{"hello"})
	err := app.Run(context.Background())
	require.NoError(t, err)
	assert.True(t, ran)
}

func TestAppRunReturnsCommandError(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("fail", WithRun(func(c *Context) error {
		return errors.New("boom")
	}))
	app.root.SetArgs([]string{"fail"})
	err := app.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	got := stderr.String()
	assert.Contains(t, got, "error:")
	assert.Contains(t, got, "boom")
	assert.Contains(t, got, "test fail --help")
	assert.Contains(t, got, "for usage.")
}

func TestRunPrintsInvalidSelectArgAndUsageHint(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("deployctl", WithIO(io))
	app.MustCommand("deploy",
		WithSelectArg("environment", "", []string{"staging", "production"}, WithRequired(), WithEnv("environment")),
		WithRun(func(c *Context) error { return nil }),
	)
	err := Run(t, app, []string{"deploy", "ss"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
	got := stderr.String()
	assert.Contains(t, got, "error:")
	assert.Contains(t, got, "deployctl deploy --help")
	assert.Contains(t, got, "for usage.")
}

func TestWithErrorHandlerReplacesDefaultRendering(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	var handled error
	app := MustNew("t",
		WithIO(NewIO(strings.NewReader(""), &buf, &buf)),
		WithErrorHandler(func(err error) { handled = err }),
	)
	app.MustCommand("x", WithRun(func(c *Context) error {
		return errors.New("nope")
	}))
	err := Run(t, app, []string{"x"})
	require.Error(t, err)
	require.Equal(t, err, handled)
	assert.NotContains(t, buf.String(), "for usage.")
	assert.NotContains(t, buf.String(), "error:")
}

func TestNewRejectsNilWithErrorHandler(t *testing.T) {
	t.Parallel()

	_, err := New("x", WithErrorHandler(nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestRunPrintsPlainErrorPrefixWhenStderrNotTTY(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("plain", WithIO(io))
	app.MustCommand("x", WithRun(func(*Context) error { return errors.New("bad") }))
	err := Run(t, app, []string{"x"})
	require.Error(t, err)
	got := stderr.String()
	assert.Contains(t, got, "error: bad")
	assert.NotContains(t, got, "\033[")
}

func TestNewWithNilIOStreamsFails(t *testing.T) {
	t.Parallel()

	_, err := New("test", WithIO(nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestNewIOStreams_nilStreamsAreSubstituted(t *testing.T) {
	t.Parallel()

	// NewIO tolerates nil components by substituting io.Discard /
	// a no-op reader; construction must succeed and writes must not panic.
	app, err := New("test", WithIO(NewIO(nil, nil, nil)))
	require.NoError(t, err)
	require.NotNil(t, app.IO())
	_, err = app.IO().Out.Write([]byte("ignored"))
	require.NoError(t, err)
}

func TestNewWithNilOptionFails(t *testing.T) {
	t.Parallel()

	_, err := New("test", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestHelpShowsInputArguments(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("myctl", WithIO(io))
	app.MustCommand("deploy",
		WithArg("env", "", WithRequired(), WithEnv("env")),
		WithRun(func(c *Context) error { return nil }),
	)
	require.NoError(t, Run(t, app, []string{"deploy", "--help"}))
	got := out.String()
	assert.Contains(t, got, "Arguments:")
	assert.Contains(t, got, "env")
	assert.Contains(t, got, "(env: MYCTL_ENV)")
	assert.Contains(t, got, "required")
}

func TestHelpShowsInheritedFlags(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("myctl", WithIO(io))
	parent := app.MustCommand("cluster",
		WithFlag("config", "", WithPersistent(), WithUsage("config file")),
	)
	parent.MustCommand("create",
		WithRun(func(c *Context) error { return nil }),
	)
	require.NoError(t, Run(t, app, []string{"cluster", "create", "--help"}))
	got := out.String()
	assert.Contains(t, got, "config")
}

func TestWithThemeSuccessOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		theme string
	}{
		{name: "Charm renders success output", theme: theme.Charm},
		{name: "Minimal renders success output", theme: theme.Minimal},
		{name: "Dracula renders success output", theme: theme.Dracula},
		{name: "CatppuccinMocha renders success output", theme: theme.CatppuccinMocha},
		{name: "Nabat renders success output", theme: theme.Nabat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, _, stderr := testIO()
			app := MustNew("test",
				WithTheme(tt.theme),
				WithIO(io),
			)
			app.MustCommand("test", WithRun(func(c *Context) error {
				c.Success("done")
				return nil
			}))
			require.NoError(t, Run(t, app, []string{"test"}))
			assert.Contains(t, stderr.String(), "✓")
		})
	}
}

// TestWithThemeAndWithCustomThemeLastWins locks the post-P6
// behavior: the two installers compose and the last one wins. The
// override slot ([WithThemeOverride]) handles per-token tweaks
// without requiring callers to construct a derived custom theme.
func TestWithThemeAndWithCustomThemeLastWins(t *testing.T) {
	t.Parallel()

	custom := theme.Theme{
		Name:    "acme",
		Default: theme.VariantDark,
		Variants: map[theme.Variant]theme.Palette{
			theme.VariantDark: {
				Tokens: map[theme.Token]lipgloss.Style{
					theme.StatusError: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")),
				},
			},
		},
	}

	t.Run("WithCustomTheme after WithTheme overrides the builtin", func(t *testing.T) {
		t.Parallel()

		app, err := New("myctl",
			WithTheme(theme.Default),
			WithCustomTheme(custom),
		)
		require.NoError(t, err)
		assert.Equal(t, "acme", app.Theme().Name(),
			"WithCustomTheme should override the prior WithTheme")
	})

	t.Run("WithTheme after WithCustomTheme overrides the custom", func(t *testing.T) {
		t.Parallel()

		app, err := New("myctl",
			WithCustomTheme(custom),
			WithTheme(theme.Default),
		)
		require.NoError(t, err)
		assert.Equal(t, theme.Default, app.Theme().Name(),
			"WithTheme should override the prior WithCustomTheme")
	})

	t.Run("either alone succeeds", func(t *testing.T) {
		t.Parallel()

		app1, err := New("myctl", WithTheme(theme.Default))
		require.NoError(t, err)
		require.NotNil(t, app1)

		app2, err := New("myctl", WithCustomTheme(custom))
		require.NoError(t, err)
		require.NotNil(t, app2)
	})
}

func TestConfigErrorsErrorAndHasIssues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(*ConfigErrors)
		wantIssues bool
		errCheck   func(*testing.T, string)
	}{
		{
			name: "multiple issues aggregate in Error string",
			setup: func(errs *ConfigErrors) {
				errs.AddErr(errors.New("error 1"))
				errs.AddErr(errors.New("error 2"))
			},
			wantIssues: true,
			errCheck: func(t *testing.T, msg string) {
				t.Helper()
				assert.Contains(t, msg, "2 configuration errors")
			},
		},
		{
			name: "single issue returns that message as Error",
			setup: func(errs *ConfigErrors) {
				errs.AddErr(errors.New("only error"))
			},
			wantIssues: true,
			errCheck: func(t *testing.T, msg string) {
				t.Helper()
				assert.Equal(t, "only error", msg)
			},
		},
		{
			name:       "empty ConfigErrors yields sentinel Error message",
			setup:      func(errs *ConfigErrors) {},
			wantIssues: false,
			errCheck: func(t *testing.T, msg string) {
				t.Helper()
				assert.Equal(t, "nabat: no configuration errors", msg)
			},
		},
		{
			name: "nil errors are ignored; blank strings are kept verbatim",
			setup: func(errs *ConfigErrors) {
				errs.AddErr(nil)
				errs.AddErr(errors.New("   "))
				errs.AddErr(errors.New(""))
			},
			wantIssues: true,
			errCheck: func(t *testing.T, msg string) {
				t.Helper()
				assert.Contains(t, msg, "2 configuration errors:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var errs ConfigErrors
			tt.setup(&errs)
			assert.Equal(t, tt.wantIssues, errs.HasIssues())
			tt.errCheck(t, errs.Error())
		})
	}
}

func TestNewValidationEmptyName(t *testing.T) {
	t.Parallel()

	_, err := New("")
	require.Error(t, err)
}

func TestAppCommandEmptyNameReturnsError(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	cmd, err := app.Command("")
	require.Nil(t, cmd, "Command returns nil on registration failure")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command name cannot be empty")
}

func TestAppMustCommandPanicsOnEmptyName(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	require.Panics(t, func() {
		app.MustCommand("")
	})
}

func TestAppCommandReturnsCommand(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	cmd, err := app.Command("sub", WithDescription("a sub"))
	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestWithLoggerInjectsLogger(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	io, _, _, _ := testIO()
	app := MustNew("test",
		WithIO(io),
		WithLogger(logger),
	)
	app.MustCommand("run", WithRun(func(c *Context) error {
		c.Logger().Info("hello", "id", 1)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Contains(t, logs.String(), "hello")
	assert.Contains(t, logs.String(), "id=1")
}

func TestWithLoggerNilRejected(t *testing.T) {
	t.Parallel()

	_, err := New("test", WithLogger(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithLogger logger cannot be nil")
}

func TestContextLoggerDiscardsByDefault(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	app.MustCommand("run", WithRun(func(c *Context) error {
		// logger is non-nil and writes go to discard handler
		require.NotNil(t, c.Logger())
		c.Logger().Info("dropped")
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
}

func TestAppSetLoggerOverridesWithLogger(t *testing.T) {
	t.Parallel()

	var first, second bytes.Buffer
	firstLogger := slog.New(slog.NewTextHandler(&first, nil))
	secondLogger := slog.New(slog.NewTextHandler(&second, nil))

	app := MustNew("test", WithLogger(firstLogger))
	app.SetLogger(secondLogger) // extension-style override
	app.MustCommand("run", WithRun(func(c *Context) error {
		c.Logger().Info("hello")
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))

	assert.Empty(t, first.String())
	assert.Contains(t, second.String(), "hello")
}

func TestAppCommandEmptyNameDoesNotPanic(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	require.NotPanics(t, func() {
		_, err := app.Command("")
		assert.Error(t, err)
	})
}

func TestWithGroupRenderedInHelp(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("myctl", WithIO(io))
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
	app.MustCommand("info",
		WithDescription("Show info"),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"--help"})
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "operations:")
	assert.Contains(t, got, "deploy")
	assert.Contains(t, got, "status")
	// ungrouped commands still appear
	assert.Contains(t, got, "Commands:")
}

func TestAsExtensionInitRunsDuringNew(t *testing.T) {
	t.Parallel()

	var ran bool
	app, err := New("x",
		AsExtension("probe", func(a AppSurface) error {
			ran = true
			assert.Equal(t, "x", a.Name())
			return nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, app)
	assert.True(t, ran)
}

func TestWithRootInitNilRejected(t *testing.T) {
	t.Parallel()

	_, err := New("x", WithRootInit(nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}
