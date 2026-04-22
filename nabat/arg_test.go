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
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgResolvesFromEnv(t *testing.T) {
	const env = "TEST_TARGET"
	t.Setenv(env, "production")

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithArg("target", "", WithEnv("target")),
		WithRun(func(c *Context) error {
			var args struct {
				Target string `nabat:"target"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Target
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, "production", got)
}

func TestArgDefaultUsedWhenMissing(t *testing.T) {
	t.Parallel()

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithArg("target", "staging"),
		WithRun(func(c *Context) error {
			var args struct {
				Target string `nabat:"target"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Target
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, "staging", got)
}

func TestRequiredArgMissing(t *testing.T) {
	t.Setenv("TEST_TARGET", "")
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithArg("target", "", WithRequired(), WithEnv("target")),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy"})
	require.Error(t, err)
}

// TestArgPromptValidatorsAccepted verifies that typed validators compose
// correctly inside [WithPrompt].
func TestArgPromptValidatorsAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmdName string
		opts    []CommandOption
		wantErr bool
	}{
		{
			name:    "WithValidate accepted on string arg via WithPrompt",
			cmdName: "deploy",
			opts: []CommandOption{
				WithArg("name", "", WithPrompt("Name", "",
					WithValidate(func(s string) error {
						if s == "" {
							return fmt.Errorf("cannot be empty")
						}
						return nil
					}),
				)),
				WithRun(func(c *Context) error { return nil }),
			},
		},
		{
			name:    "WithValidate accepted on bool arg via WithPrompt",
			cmdName: "deploy",
			opts: []CommandOption{
				WithArg("confirm", false, WithPrompt("Confirm?", "",
					WithValidate(func(b bool) error {
						if !b {
							return fmt.Errorf("must confirm")
						}
						return nil
					}),
				)),
				WithRun(func(c *Context) error { return nil }),
			},
		},
		{
			name:    "WithValidate accepted on multi-select arg via WithPrompt",
			cmdName: "deploy",
			opts: []CommandOption{
				WithMultiSelectArg("targets", nil, []string{"a", "b", "c"},
					WithPrompt("Pick targets", "",
						WithValidate(func(vals []string) error {
							if len(vals) == 0 {
								return fmt.Errorf("select at least one")
							}
							return nil
						}),
					),
				),
				WithRun(func(c *Context) error { return nil }),
			},
		},
		{
			name:    "select arg rejects default not in choices",
			cmdName: "deploy",
			opts: []CommandOption{
				WithSelectArg("env", "dev", []string{"staging", "production"}),
				WithRun(func(c *Context) error { return nil }),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := MustNew("test")
			_, err := app.Command(tt.cmdName, tt.opts...)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestArgTypedFromCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    []CommandOption
		runArgs []string
	}{
		{
			name: "int arg parses positional",
			opts: []CommandOption{
				WithArg("count", 1),
				WithRun(func(c *Context) error {
					v, err := BindAs[int](c, "count")
					require.NoError(t, err)
					assert.Equal(t, 42, v)
					return nil
				}),
			},
			runArgs: []string{"run", "42"},
		},
		{
			name: "float arg parses positional",
			opts: []CommandOption{
				WithArg("ratio", 0.0),
				WithRun(func(c *Context) error {
					v, err := BindAs[float64](c, "ratio")
					require.NoError(t, err)
					assert.InDelta(t, 3.14, v, 0.001)
					return nil
				}),
			},
			runArgs: []string{"run", "3.14"},
		},
		{
			name: "int64 arg parses positional",
			opts: []CommandOption{
				WithArg("size", int64(0)),
				WithRun(func(c *Context) error {
					v, err := BindAs[int64](c, "size")
					require.NoError(t, err)
					assert.Equal(t, int64(9876543210), v)
					return nil
				}),
			},
			runArgs: []string{"run", "9876543210"},
		},
		{
			name: "uint arg parses positional",
			opts: []CommandOption{
				WithArg("port", uint(0)),
				WithRun(func(c *Context) error {
					v, err := BindAs[uint](c, "port")
					require.NoError(t, err)
					assert.Equal(t, uint(8080), v)
					return nil
				}),
			},
			runArgs: []string{"run", "8080"},
		},
		{
			name: "duration arg parses positional",
			opts: []CommandOption{
				WithArg("timeout", time.Duration(0)),
				WithRun(func(c *Context) error {
					v, err := BindAs[time.Duration](c, "timeout")
					require.NoError(t, err)
					assert.Equal(t, 30*time.Second, v)
					return nil
				}),
			},
			runArgs: []string{"run", "30s"},
		},
		{
			name: "string arg with multiline mode preserves positional value",
			opts: []CommandOption{
				WithArg("body", ""),
				WithRun(func(c *Context) error {
					v, err := BindAs[string](c, "body")
					require.NoError(t, err)
					assert.Equal(t, "hello world", v)
					return nil
				}),
			},
			runArgs: []string{"run", "hello world"},
		},
		{
			name: "string arg used as file path preserves positional value",
			opts: []CommandOption{
				WithArg("path", ""),
				WithRun(func(c *Context) error {
					v, err := BindAs[string](c, "path")
					require.NoError(t, err)
					assert.Equal(t, "/tmp/test.txt", v)
					return nil
				}),
			},
			runArgs: []string{"run", "/tmp/test.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, _, _ := testIO()
			app := MustNew("test", WithIO(io))
			app.MustCommand("run", tt.opts...)
			require.NoError(t, Run(t, app, tt.runArgs))
		})
	}
}

func TestWithEnvArgPrimaryFirst(t *testing.T) {
	t.Setenv("TEST_TOKEN", "primary-value")
	t.Setenv("FALLBACK_TOKEN", "alias-value")

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(io))
	app.MustCommand("run",
		WithArg("token", "", WithEnv("token"), WithEnvAlias("FALLBACK_TOKEN")),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "token")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "primary-value", got)
}

func TestWithEnvArgFallsBackToAlias(t *testing.T) {
	t.Setenv("FALLBACK_TOKEN", "alias-value")

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithEnvPrefix("NOTSET_"), WithIO(io))
	app.MustCommand("run",
		WithArg("token", "", WithEnv("token"), WithEnvAlias("FALLBACK_TOKEN")),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "token")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "alias-value", got)
}

// TestArgDefaultValIsNonInteractiveFallback locks in the design decision
// from the option-unification refactor: WithFallback was removed and the
// typed `defaultVal` parameter of WithArg is now the only non-interactive
// fallback, so the value type is checked at compile time by Go generics.
func TestArgDefaultValIsNonInteractiveFallback(t *testing.T) {
	t.Parallel()

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithArg("name", "fallback-name"),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "name")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "fallback-name", got)
}

// TestArgPromptKindOptionsAccepted exercises the typed prompt sub-options
// to confirm the [WithPrompt] wiring does not regress.
func TestArgPromptKindOptionsAccepted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []CommandOption
	}{
		{
			name: "string prompt accepts hint+maxChars+suggestions",
			opts: []CommandOption{
				WithArg("name", "",
					WithPrompt("Name", "Your full name",
						WithInlineString(),
						WithHint("alice"),
						WithMaxChars(50),
						WithSuggestions("alice", "bob"),
					),
				),
				WithRun(func(c *Context) error { return nil }),
			},
		},
		{
			name: "bool prompt accepts affirmative/negative labels",
			opts: []CommandOption{
				WithArg("confirm", false,
					WithPrompt("Confirm?", "",
						WithAffirmative("Yes"),
						WithNegative("No"),
					),
				),
				WithRun(func(c *Context) error { return nil }),
			},
		},
		{
			name: "string multiline prompt accepts editor command + extension",
			opts: []CommandOption{
				WithArg("body", "",
					WithPrompt("Body", "",
						WithMultiline(),
						WithEditorCmd("vim"),
						WithEditorExtension(".md"),
					),
				),
				WithRun(func(c *Context) error { return nil }),
			},
		},
		{
			name: "string file-picker prompt accepts allowed types + dir + current dir",
			opts: []CommandOption{
				WithArg("path", "",
					WithPrompt("Pick a file", "",
						WithFilePicker(),
						WithAllowedTypes(".go", ".txt"),
						WithDirAllowed(),
						WithCurrentDir("/tmp"),
					),
				),
				WithRun(func(c *Context) error { return nil }),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := MustNew("test")
			_, err := app.Command("run", tt.opts...)
			require.NoError(t, err)
		})
	}
}

// Kind mismatch between prompt options and field type is now caught at
// compile time by the typed-seal mechanism on [FieldOption]. See
// internal/buildtest/ for the compile-time-failure test suite.

func TestRequiredArgMissingReturnsError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithArg("env", "", WithRequired()),
		WithRun(func(c *Context) error { return nil }),
	)
	err := Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required arg")
}

// Regression: required positional args must be satisfiable from env without any
// CLI positional tokens. Cobra-side MinimumNArgs would reject this valid flow.
func TestRequiredSelectArgFromEnvWithoutCLIPositional(t *testing.T) {
	t.Setenv("DEPLOYCTL_ENVIRONMENT", "staging")
	io, _, out, _ := testIO()
	app := MustNew("deployctl",
		WithEnvPrefix("DEPLOYCTL_"),
		WithIO(io),
	)
	app.MustCommand("deploy",
		WithSelectArg("environment", "staging", []string{"staging", "production"},
			WithRequired(), WithEnv("environment")),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "environment")
			require.NoError(t, err)
			c.Print(v)
			return nil
		}),
	)
	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, "staging", strings.TrimSpace(out.String()))
}

func TestWithArgIntFromEnv(t *testing.T) {
	t.Setenv("TEST_COUNT", "99")
	var got int
	io, _, _, _ := testIO()
	app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(io))
	app.MustCommand("run",
		WithArg("count", 0, WithEnv("count")),
		WithRun(func(c *Context) error {
			v, err := BindAs[int](c, "count")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, 99, got)
}

func TestWithArgUintFromEnv(t *testing.T) {
	t.Setenv("TEST_PORT", "9000")
	var got uint
	io, _, _, _ := testIO()
	app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(io))
	app.MustCommand("run",
		WithArg("port", uint(0), WithEnv("port")),
		WithRun(func(c *Context) error {
			v, err := BindAs[uint](c, "port")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, uint(9000), got)
}

func TestWithArgInt64FromEnv(t *testing.T) {
	t.Setenv("TEST_SIZE", "1099511627776")
	var got int64
	io, _, _, _ := testIO()
	app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(io))
	app.MustCommand("run",
		WithArg("size", int64(0), WithEnv("size")),
		WithRun(func(c *Context) error {
			v, err := BindAs[int64](c, "size")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, int64(1099511627776), got)
}

func TestWithEnvPrefix(t *testing.T) {
	t.Setenv("MYAPP_TARGET", "prod")

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test",
		WithEnvPrefix("MYAPP_"),
		WithIO(io),
	)
	app.MustCommand("deploy",
		WithArg("target", "", WithEnv("target")),
		WithRun(func(c *Context) error {
			var args struct {
				Target string `nabat:"target"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Target
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, "prod", got)
}

func TestSelectArgValidation(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithSelectArg("env", "staging", []string{"staging", "production"}),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestSelectArgFromCLI(t *testing.T) {
	t.Parallel()

	var got string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("deploy",
		WithSelectArg("env", "staging", []string{"staging", "production"}),
		WithRun(func(c *Context) error {
			var args struct {
				Env string `nabat:"env"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Env
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy", "production"})
	require.NoError(t, err)
	assert.Equal(t, "production", got)
}

func TestSelectDefaultValidation(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("test",
		WithSelectArg("env", "c", []string{"a", "b"}),
		WithRun(func(c *Context) error { return nil }),
	)
	require.Error(t, err)
}
