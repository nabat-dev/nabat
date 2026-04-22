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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagResolvedFromCLI(t *testing.T) {
	t.Parallel()

	var got int
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("replicas", 1),
		WithRun(func(c *Context) error {
			var args struct {
				Replicas int `nabat:"replicas"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Replicas
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy", "--replicas", "5"})
	require.NoError(t, err)
	assert.Equal(t, 5, got)
}

func TestFlagResolvedFromEnv(t *testing.T) {
	t.Setenv("TEST_REPLICAS", "7")

	var got int
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("replicas", 0, WithEnv("replicas")),
		WithRun(func(c *Context) error {
			var args struct {
				Replicas int `nabat:"replicas"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Replicas
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, 7, got)
}

func TestFlagResolvedFromCustomEnvVar(t *testing.T) {
	t.Setenv("CUSTOM_COUNT", "42")

	var got int
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("replicas", 0, WithEnv("replicas"), WithEnvAlias("CUSTOM_COUNT")),
		WithRun(func(c *Context) error {
			var args struct {
				Replicas int `nabat:"replicas"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Replicas
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy"})
	require.NoError(t, err)
	assert.Equal(t, 42, got)
}

func TestFlagCLITakesPrecedenceOverEnv(t *testing.T) {
	t.Setenv("TEST_REPLICAS", "99")

	var got int
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("replicas", 0, WithEnv("replicas")),
		WithRun(func(c *Context) error {
			var args struct {
				Replicas int `nabat:"replicas"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Replicas
			return nil
		}),
	)

	err := Run(t, app, []string{"deploy", "--replicas", "3"})
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestRequiredFlagMissing(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("target", "", WithRequired()),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag")
}

func TestBoolFlag(t *testing.T) {
	t.Parallel()

	var got bool
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("verbose", false),
		WithRun(func(c *Context) error {
			var args struct {
				Verbose bool `nabat:"verbose"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Verbose
			return nil
		}),
	)

	err := Run(t, app, []string{"run", "--verbose"})
	require.NoError(t, err)
	assert.True(t, got)
}

func TestFloatFlag(t *testing.T) {
	t.Parallel()

	var got float64
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("rate", 1.5),
		WithRun(func(c *Context) error {
			var args struct {
				Rate float64 `nabat:"rate"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Rate
			return nil
		}),
	)

	err := Run(t, app, []string{"run", "--rate", "3.14"})
	require.NoError(t, err)
	assert.Equal(t, 3.14, got)
}

func TestStringSliceFlag(t *testing.T) {
	t.Parallel()

	var got []string
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("tags", []string{}),
		WithRun(func(c *Context) error {
			var args struct {
				Tags []string `nabat:"tags"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Tags
			return nil
		}),
	)

	err := Run(t, app, []string{"run", "--tags", "a,b,c"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestStringSliceFlagFromEnv(t *testing.T) {
	t.Setenv("TEST_TAGS", "x,y")

	var got []string
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("tags", []string{}, WithEnv("tags")),
		WithRun(func(c *Context) error {
			var args struct {
				Tags []string `nabat:"tags"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Tags
			return nil
		}),
	)

	err := Run(t, app, []string{"run"})
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, got)
}

func TestFlagShortOption(t *testing.T) {
	t.Parallel()

	var got bool
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("verbose", false, WithShort('v')),
		WithRun(func(c *Context) error {
			var args struct {
				Verbose bool `nabat:"verbose"`
			}
			require.NoError(t, c.Bind(&args))
			got = args.Verbose
			return nil
		}),
	)

	err := Run(t, app, []string{"run", "-v"})
	require.NoError(t, err)
	assert.True(t, got)
}

func TestPersistentRootFlagReadableInSubcommand(t *testing.T) {
	t.Parallel()

	var got bool
	var out bytes.Buffer
	app := MustNew("test",
		WithIO(NewIO(strings.NewReader(""), &out, &out)),
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

func TestSelectFlagValidation(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithSelectFlag("env", "staging", []string{"staging", "production"}),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "--env", "invalid"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestWithEnvFlagFallsBackToAlias(t *testing.T) {
	t.Setenv("GH_TOKEN", "gh-token-value")

	var got string
	var out bytes.Buffer
	app := MustNew("test", WithEnvPrefix("NOTSET_"), WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("token", "", WithEnv("token"), WithEnvAlias("GH_TOKEN", "GITHUB_TOKEN")),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "token")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "gh-token-value", got)
}

func TestWithEnvSecondAliasUsedWhenFirstMissing(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "github-value")

	var got string
	var out bytes.Buffer
	app := MustNew("test", WithEnvPrefix("NOTSET_"), WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("token", "", WithEnv("token"), WithEnvAlias("GH_TOKEN", "GITHUB_TOKEN")),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "token")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "github-value", got)
}

func TestWithMultiSelectFlagFromCLI(t *testing.T) {
	t.Parallel()

	var got []string
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithMultiSelectFlag("regions", []string{}, []string{"us-east-1", "eu-west-1", "ap-south-1"}),
		WithRun(func(c *Context) error {
			v, err := BindAs[[]string](c, "regions")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run", "--regions=us-east-1,eu-west-1"}))
	assert.Equal(t, []string{"us-east-1", "eu-west-1"}, got)
}

func TestWithMultiSelectFlagInvalidChoiceRejected(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithMultiSelectFlag("regions", []string{}, []string{"us-east-1", "eu-west-1"}),
		WithRun(func(c *Context) error { return nil }),
	)
	err := Run(t, app, []string{"run", "--regions=invalid"})
	require.Error(t, err)
}

func TestPersistentFlagInheritedByChild(t *testing.T) {
	t.Parallel()

	var got string
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	parent := app.MustCommand("cluster",
		WithFlag("config", "default.yaml", WithPersistent()),
	)
	parent.MustCommand("create",
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "config")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"cluster", "create", "--config=custom.yaml"}))
	assert.Equal(t, "custom.yaml", got)
}

func TestFlagFromEnvInvalidValue(t *testing.T) {
	tests := []struct {
		name        string
		envKey      string
		envVal      string
		opts        []CommandOption
		errContains string
	}{
		{
			name:   "invalid bool in env surfaces bool parse error",
			envKey: "TEST_FLAG",
			envVal: "notabool",
			opts: []CommandOption{
				WithFlag("flag", false, WithEnv("flag")),
				WithRun(func(c *Context) error { return nil }),
			},
			errContains: "bool",
		},
		{
			name:   "invalid int in env surfaces int parse error",
			envKey: "TEST_COUNT",
			envVal: "notanint",
			opts: []CommandOption{
				WithFlag("count", 0, WithEnv("count")),
				WithRun(func(c *Context) error { return nil }),
			},
			errContains: "int",
		},
		{
			name:   "invalid float in env surfaces float parse error",
			envKey: "TEST_RATIO",
			envVal: "notafloat",
			opts: []CommandOption{
				WithFlag("ratio", 0.0, WithEnv("ratio")),
				WithRun(func(c *Context) error { return nil }),
			},
			errContains: "float",
		},
		{
			name:   "invalid duration in env surfaces duration parse error",
			envKey: "TEST_TIMEOUT",
			envVal: "notaduration",
			opts: []CommandOption{
				WithFlag("timeout", time.Duration(0), WithEnv("timeout")),
				WithRun(func(c *Context) error { return nil }),
			},
			errContains: "duration",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)
			var out bytes.Buffer
			app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(NewIO(strings.NewReader(""), &out, &out)))
			app.MustCommand("run", tt.opts...)
			err := Run(t, app, []string{"run"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestFlagFromEnvStringSlice(t *testing.T) {
	t.Setenv("TEST_TAGS", "a,b,c")
	var got []string
	var out bytes.Buffer
	app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("tags", []string{}, WithEnv("tags")),
		WithRun(func(c *Context) error {
			v, err := BindAs[[]string](c, "tags")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func TestValidateDefaultTypeStringSliceInvalidItem(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	_, err := app.Command("run",
		WithMultiSelectFlag("env", []string{"c"}, []string{"a", "b"}),
		WithRun(func(c *Context) error { return nil }),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default value")
}

func TestWithRequiredFlagMissingReturnsError(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("token", "", WithRequired()),
		WithRun(func(c *Context) error { return nil }),
	)
	err := Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required flag")
}

func TestFlagShorthandWorks(t *testing.T) {
	t.Parallel()

	var got bool
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("verbose", false, WithShort('v')),
		WithRun(func(c *Context) error {
			v, err := BindAs[bool](c, "verbose")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run", "-v"}))
	assert.True(t, got)
}

func TestSelectFlagFromEnv(t *testing.T) {
	t.Setenv("TEST_ENV", "production")
	var got string
	var out bytes.Buffer
	app := MustNew("test", WithEnvPrefix("TEST_"), WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithSelectFlag("env", "staging", []string{"staging", "production"}, WithEnv("env")),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "env")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "production", got)
}

func TestPersistentFlagsInCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		appOpts     []Option
		nested      bool
		parentName  string
		childName   string
		topCmd      string
		runArgs     []string
		wantErr     bool
		errContains string
		run         func(*Context) error
	}{
		{
			name: "global string flag uses default in subcommand",
			appOpts: []Option{
				WithFlag("config", "default.yaml", WithPersistent()),
			},
			nested:  false,
			topCmd:  "deploy",
			runArgs: []string{"deploy"},
			run: func(c *Context) error {
				v, err := BindAs[string](c, "config")
				require.NoError(t, err)
				assert.Equal(t, "default.yaml", v)
				return nil
			},
		},
		{
			name: "global string flag set from CLI overrides default",
			appOpts: []Option{
				WithFlag("config", "default.yaml", WithPersistent()),
			},
			nested:  false,
			topCmd:  "deploy",
			runArgs: []string{"deploy", "--config=prod.yaml"},
			run: func(c *Context) error {
				v, err := BindAs[string](c, "config")
				require.NoError(t, err)
				assert.Equal(t, "prod.yaml", v)
				return nil
			},
		},
		{
			name: "global bool flag readable in subcommand",
			appOpts: []Option{
				WithFlag("verbose", false, WithShort('v'), WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run", "--verbose"},
			run: func(c *Context) error {
				v, err := BindAs[bool](c, "verbose")
				require.NoError(t, err)
				assert.True(t, v)
				return nil
			},
		},
		{
			name: "global int flag default available in subcommand",
			appOpts: []Option{
				WithFlag("timeout", 30, WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run"},
			run: func(c *Context) error {
				v, err := BindAs[int](c, "timeout")
				require.NoError(t, err)
				assert.Equal(t, 30, v)
				return nil
			},
		},
		{
			name: "global duration flag default available in nested command",
			appOpts: []Option{
				WithFlag("timeout", 10*time.Second, WithPersistent()),
			},
			nested:     true,
			parentName: "cluster",
			childName:  "create",
			runArgs:    []string{"cluster", "create"},
			run: func(c *Context) error {
				v, err := BindAs[time.Duration](c, "timeout")
				require.NoError(t, err)
				assert.Equal(t, 10*time.Second, v)
				return nil
			},
		},
		{
			name: "global count flag accumulates short flags",
			appOpts: []Option{
				WithFlag("verbose", 0, WithShort('v'), WithCount(), WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run", "-v", "-v"},
			run: func(c *Context) error {
				v, err := BindAs[int](c, "verbose")
				require.NoError(t, err)
				assert.Equal(t, 2, v)
				return nil
			},
		},
		{
			name: "global string slice flag parses comma separated CLI",
			appOpts: []Option{
				WithFlag("tags", []string{}, WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run", "--tags=a,b,c"},
			run: func(c *Context) error {
				v, err := BindAs[[]string](c, "tags")
				require.NoError(t, err)
				assert.Equal(t, []string{"a", "b", "c"}, v)
				return nil
			},
		},
		{
			name: "global select flag rejects invalid choice",
			appOpts: []Option{
				WithSelectFlag("env", "staging", []string{"staging", "production"}, WithPersistent()),
			},
			nested:      false,
			topCmd:      "run",
			runArgs:     []string{"run", "--env=invalid"},
			wantErr:     true,
			errContains: "must be one of",
			run:         func(c *Context) error { return nil },
		},
		{
			name: "global multi-select flag accepts multiple regions",
			appOpts: []Option{
				WithMultiSelectFlag("regions", []string{}, []string{"us-east-1", "eu-west-1", "ap-south-1"}, WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run", "--regions=us-east-1,eu-west-1"},
			run: func(c *Context) error {
				v, err := BindAs[[]string](c, "regions")
				require.NoError(t, err)
				assert.Equal(t, []string{"us-east-1", "eu-west-1"}, v)
				return nil
			},
		},
		{
			name: "global float flag default available in subcommand",
			appOpts: []Option{
				WithFlag("ratio", 0.5, WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run"},
			run: func(c *Context) error {
				v, err := BindAs[float64](c, "ratio")
				require.NoError(t, err)
				assert.Equal(t, 0.5, v)
				return nil
			},
		},
		{
			name: "global int64 flag default available in subcommand",
			appOpts: []Option{
				WithFlag("size", int64(1024), WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run"},
			run: func(c *Context) error {
				v, err := BindAs[int64](c, "size")
				require.NoError(t, err)
				assert.Equal(t, int64(1024), v)
				return nil
			},
		},
		{
			name: "global uint flag default available in subcommand",
			appOpts: []Option{
				WithFlag("port", uint(8080), WithPersistent()),
			},
			nested:  false,
			topCmd:  "run",
			runArgs: []string{"run"},
			run: func(c *Context) error {
				v, err := BindAs[uint](c, "port")
				require.NoError(t, err)
				assert.Equal(t, uint(8080), v)
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			appOpts := append([]Option{WithIO(NewIO(strings.NewReader(""), &out, &out))}, tt.appOpts...)
			app := MustNew("test", appOpts...)
			if tt.nested {
				parent := app.MustCommand(tt.parentName)
				parent.MustCommand(tt.childName, WithRun(tt.run))
			} else {
				app.MustCommand(tt.topCmd, WithRun(tt.run))
			}
			err := Run(t, app, tt.runArgs)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInvalidFlagReturnsStyledError(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithRun(func(c *Context) error { return nil }),
	)
	// Passing an unknown flag fails during Cobra flag parsing before RunE.
	err := Run(t, app, []string{"run", "--unknown-flag"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown flag")
	got := out.String()
	assert.Contains(t, got, "error:")
	assert.Contains(t, got, "test run --help")
	assert.Contains(t, got, "for usage.")
}

func TestFlagReturnsError(t *testing.T) {
	t.Parallel()

	_, err := New("test",
		WithFlag("", ""),
	)
	require.Error(t, err)
}

func TestFlagSuccess(t *testing.T) {
	t.Parallel()

	_, err := New("test",
		WithFlag("verbose", false, WithShort('v')),
	)
	require.NoError(t, err)
}

func TestNormalizeEnvName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "hyphens become underscores and uppercases", input: "some-flag", want: "SOME_FLAG"},
		{name: "dots become underscores and uppercases", input: "some.flag", want: "SOME_FLAG"},
		{name: "trims spaces before normalizing", input: "  spaces  ", want: "SPACES"},
		{name: "preserves existing underscores and uppercases", input: "already_upper", want: "ALREADY_UPPER"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeEnvName(tt.input)
			assert.Equalf(t, tt.want, got, "normalizeEnvName(%q)", tt.input)
		})
	}
}

func TestCloneDefaultSlice(t *testing.T) {
	t.Parallel()

	orig := []string{"a", "b"}
	clonedAny := cloneDefault(orig)
	cloned, ok := clonedAny.([]string)
	require.True(t, ok, "cloneDefault([]string) want []string, got %T", clonedAny)
	cloned[0] = "x"
	assert.Equal(t, "a", orig[0])
}

func TestCloneDefaultScalar(t *testing.T) {
	t.Parallel()

	v := cloneDefault(42)
	assert.Equal(t, 42, v)
}

func TestWithUsage(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("myctl",
		WithIO(NewIO(strings.NewReader(""), &out, &out)),
	)
	app.MustCommand("deploy",
		WithFlag("replicas", 2, WithUsage("Number of replicas")),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "--help"})
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "Number of replicas")
}

func TestDurationZeroDefaultNotShownInHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("myctl", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("timeout", time.Duration(0), WithUsage("request timeout")),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"run", "--help"})
	require.NoError(t, err)
	assert.NotContains(t, out.String(), "default: 0s")
}

func TestWithDefaultGenericInt(t *testing.T) {
	t.Parallel()

	var got int
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithFlag("count", 99),
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

func TestInputPositionalDefault(t *testing.T) {
	t.Parallel()

	var got string
	var out bytes.Buffer
	app := MustNew("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("run",
		WithArg("env", "production"),
		WithRun(func(c *Context) error {
			v, err := BindAs[string](c, "env")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "production", got)
}
