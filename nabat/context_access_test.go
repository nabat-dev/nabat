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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func TestBindAs(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithArg("name", "world"),
		nabat.WithFlag("count", 42),
		nabat.WithRun(func(c *nabat.Context) error {
			name, err := nabat.BindAs[string](c, "name")
			require.NoError(t, err)
			assert.Equal(t, "world", name)
			count, err := nabat.BindAs[int](c, "count")
			require.NoError(t, err)
			assert.Equal(t, 42, count)
			_, err = nabat.BindAs[int](c, "missing")
			require.Error(t, err)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestWithCountRequiresIntFlag(t *testing.T) {
	t.Parallel()
	_, err := nabat.New("x",
		nabat.WithCommand("v",
			nabat.WithFlag("verbose", "x", nabat.WithCount())))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithCount() requires an int flag default")
}

func TestWithCountFlag(t *testing.T) {
	t.Parallel()

	var got int
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithFlag("verbose", 0, nabat.WithShort('v'), nabat.WithCount()),
		nabat.WithRun(func(c *nabat.Context) error {
			v, err := nabat.BindAs[int](c, "verbose")
			require.NoError(t, err)
			got = v
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run", "-v", "-v", "-v"}))
	assert.Equal(t, 3, got)
}

func TestBindOptionalAbsentFieldKeepsZeroValue(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithArg("required", "", nabat.WithRequired()),
		nabat.WithArg("optional", ""),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Required string `nabat:"required"`
				Optional string `nabat:"optional"`
			}
			require.NoError(t, c.Bind(&args))
			assert.Equal(t, "hello", args.Required)
			assert.Equal(t, "", args.Optional)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run", "hello"}))
}

func TestBindOptionalBoolAbsentUsesDefault(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithFlag("verbose", false),
		nabat.WithFlag("dry-run", false),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Verbose bool `nabat:"verbose"`
				DryRun  bool `nabat:"dry-run"`
			}
			require.NoError(t, c.Bind(&args))
			assert.True(t, args.Verbose)
			assert.False(t, args.DryRun)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run", "--verbose"}))
}

func TestBindOptionalDurationAbsentIsZero(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithFlag("timeout", time.Duration(0)),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Timeout time.Duration `nabat:"timeout"`
			}
			require.NoError(t, c.Bind(&args))
			assert.Equal(t, time.Duration(0), args.Timeout)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestBindPointerNilWhenOnlyDefault(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithFlag("timeout", 30*time.Second),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Timeout *time.Duration `nabat:"timeout"`
			}
			require.NoError(t, c.Bind(&args))
			assert.Nil(t, args.Timeout)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestBindPointerSetWhenExplicit(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithFlag("timeout", time.Duration(0)),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Timeout *time.Duration `nabat:"timeout"`
			}
			require.NoError(t, c.Bind(&args))
			require.NotNil(t, args.Timeout)
			assert.Equal(t, 5*time.Minute, *args.Timeout)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run", "--timeout=5m"}))
}

func TestExplicit(t *testing.T) {
	tests := []struct {
		name    string
		setEnv  func(t *testing.T)
		appOpts []nabat.Option
		cmdOpts []nabat.CommandOption
		key     string
		runArgs []string
		want    bool
	}{
		{
			name: "true when positional arg provides value",
			cmdOpts: []nabat.CommandOption{
				nabat.WithArg("name", ""),
			},
			key:     "name",
			runArgs: []string{"run", "alice"},
			want:    true,
		},
		{
			name: "false when only default is used for input",
			cmdOpts: []nabat.CommandOption{
				nabat.WithArg("name", "world"),
			},
			key:     "name",
			runArgs: []string{"run"},
			want:    false,
		},
		{
			name: "true when value comes from environment",
			setEnv: func(t *testing.T) {
				t.Helper()
				t.Setenv("TEST_NAME", "from-env")
			},
			appOpts: []nabat.Option{nabat.WithEnvPrefix("TEST_")},
			cmdOpts: []nabat.CommandOption{
				nabat.WithArg("name", "", nabat.WithEnv("name")),
			},
			key:     "name",
			runArgs: []string{"run"},
			want:    true,
		},
		{
			name: "false when optional flag not set",
			cmdOpts: []nabat.CommandOption{
				nabat.WithFlag("output", ""),
			},
			key:     "output",
			runArgs: []string{"run"},
			want:    false,
		},
		{
			name: "true when optional flag explicitly set on CLI",
			cmdOpts: []nabat.CommandOption{
				nabat.WithFlag("output", ""),
			},
			key:     "output",
			runArgs: []string{"run", "--output=file.txt"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				tt.setEnv(t)
			} else {
				t.Parallel()
			}

			io, _, _, _ := nabattest.NewIO()
			appOpts := append([]nabat.Option{nabat.WithIO(io)}, tt.appOpts...)
			app := nabat.MustNew("test", appOpts...)
			want := tt.want
			key := tt.key
			app.MustCommand("run", append(tt.cmdOpts, nabat.WithRun(func(c *nabat.Context) error {
				assert.Equal(t, want, c.Explicit(key))
				return nil
			}))...)
			require.NoError(t, nabattest.Run(t, app, tt.runArgs))
		})
	}
}
