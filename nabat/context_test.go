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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabattest"
)

func TestContextArgs(t *testing.T) {
	t.Parallel()

	var got []string
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("echo",
		nabat.WithRun(func(c *nabat.Context) error {
			got = c.Args()
			return nil
		}),
	)

	require.NoError(t, nabattest.Run(t, app, []string{"echo", "a", "b"}))
	assert.Equal(t, []string{"a", "b"}, got)
}

func TestContextPassthrough(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		cmdArgs  []string
		wantArgs []string
		wantPass []string
		wantHas  bool
	}{
		{
			name:     "no dash: Args has all positional, HasPassthrough is false",
			cmdArgs:  []string{"exec", "redis"},
			wantArgs: []string{"redis"},
			wantPass: []string{},
			wantHas:  false,
		},
		{
			name:     "with dash and passthrough args",
			cmdArgs:  []string{"exec", "redis", "--", "redis-cli", "-h", "localhost"},
			wantArgs: []string{"redis"},
			wantPass: []string{"redis-cli", "-h", "localhost"},
			wantHas:  true,
		},
		{
			name:     "dash with nothing after: empty Passthrough but HasPassthrough true",
			cmdArgs:  []string{"exec", "redis", "--"},
			wantArgs: []string{"redis"},
			wantPass: []string{},
			wantHas:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotArgs []string
			var gotPass []string
			var gotHas bool
			io, _, _, _ := nabattest.NewIO()
			app := nabat.MustNew("test", nabat.WithIO(io))
			app.MustCommand("exec",
				nabat.WithArg("service", ""),
				nabat.WithPassthrough("command [args...]", "command to run once ready"),
				nabat.WithRun(func(c *nabat.Context) error {
					gotArgs = c.Args()
					gotPass = c.Passthrough()
					gotHas = c.HasPassthrough()
					return nil
				}),
			)
			require.NoError(t, nabattest.Run(t, app, tt.cmdArgs))
			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.Equal(t, tt.wantPass, gotPass)
			assert.Equal(t, tt.wantHas, gotHas)
		})
	}
}

func TestArgsAlwaysNonNil(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithRun(func(c *nabat.Context) error {
			args := c.Args()
			assert.NotNil(t, args)
			assert.Empty(t, args)

			pass := c.Passthrough()
			assert.NotNil(t, pass)
			assert.Empty(t, pass)
			assert.False(t, c.HasPassthrough())
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestWithPassthroughEmptyLabelError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	_, err := app.Command("exec", nabat.WithPassthrough(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "label cannot be empty")
}

func TestPassthroughWithoutDeclarationStillWorks(t *testing.T) {
	t.Parallel()

	// WithPassthrough() is optional for help; the split always happens.
	var gotPass []string
	var gotHas bool
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithRun(func(c *nabat.Context) error {
			gotPass = c.Passthrough()
			gotHas = c.HasPassthrough()
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run", "--", "foo", "bar"}))
	assert.Equal(t, []string{"foo", "bar"}, gotPass)
	assert.True(t, gotHas)
}

func TestBindSucceeds(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithArg("env", "staging"),
		nabat.WithFlag("replicas", 2),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Env      string `nabat:"env"`
				Replicas int    `nabat:"replicas"`
			}
			require.NoError(t, c.Bind(&args))
			assert.Equal(t, "staging", args.Env)
			assert.Equal(t, 2, args.Replicas)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestBindReturnsErrorOnNonPointer(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct{ Name string }
			err := c.Bind(args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "pointer to struct")
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestBindUnknownTagReturnsError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithArg("name", "alice"),
		nabat.WithRun(func(c *nabat.Context) error {
			var args struct {
				Name    string `nabat:"name"`
				Missing string `nabat:"missing"` // not declared as arg/flag
			}
			err := c.Bind(&args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `tag "missing" does not match any declared arg or flag`)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestBindNilTargetReturnsError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithRun(func(c *nabat.Context) error {
			return c.Bind(nil)
		}),
	)
	err := nabattest.Run(t, app, []string{"run"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bind target")
}

func TestBindEmbeddedStruct(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run",
		nabat.WithArg("env", "prod"),
		nabat.WithRun(func(c *nabat.Context) error {
			type Base struct {
				Env string `nabat:"env"`
			}
			var args struct {
				Base
			}
			require.NoError(t, c.Bind(&args))
			assert.Equal(t, "prod", args.Env)
			return nil
		}),
	)
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestContextDeadlineFromGoContext(t *testing.T) {
	t.Parallel()

	dl := time.Now().Add(time.Hour)
	goCtx, cancel := context.WithDeadline(context.Background(), dl)
	defer cancel()

	var got time.Time
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		got, _ = c.Deadline()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}, nabattest.WithContext(goCtx)))
	assert.Equal(t, dl.Unix(), got.Unix())
}

func TestContextCtx(t *testing.T) {
	t.Parallel()

	type ctxKey string
	goCtx := context.WithValue(context.Background(), ctxKey("hello"), "world")

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))

	var gotVal string
	app.MustCommand("test",
		nabat.WithRun(func(c *nabat.Context) error {
			v, ok := c.Value(ctxKey("hello")).(string)
			if !ok {
				return fmt.Errorf("expected string in context, got %T", c.Value(ctxKey("hello")))
			}
			gotVal = v
			return nil
		}),
	)

	require.NoError(t, nabattest.Run(t, app, []string{"test"}, nabattest.WithContext(goCtx)))
	assert.Equal(t, "world", gotVal)
}

func TestNonInteractiveContext(t *testing.T) {
	t.Parallel()

	var isInteractiveResult bool
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		isInteractiveResult = c.IsInteractive()
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"test"}))
	assert.False(t, isInteractiveResult)
}
