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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestContext builds a [Context] suitable for unit tests that need to
// exercise accessor or bind logic without going through [App.Run]. The
// returned Context has both values and set maps initialized so callers do
// not have to remember which fields newContext populates.
func newTestContext(values map[string]any) *Context {
	c := &Context{
		values: map[string]any{},
		set:    map[string]bool{},
	}
	for k, v := range values {
		c.values[k] = v
		c.set[k] = true
	}
	return c
}

func TestBindReturnsErrorOnTypeMismatch(t *testing.T) {
	t.Parallel()

	c := newTestContext(map[string]any{"x": 42})
	var args struct {
		X string `nabat:"x"`
	}
	err := c.Bind(&args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected string, got int")
}

func TestContextNilWrappedGoContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T, c *Context)
	}{
		{
			name: "Deadline reports no deadline when ctx is nil",
			run: func(t *testing.T, c *Context) {
				t.Helper()
				_, ok := c.Deadline()
				assert.False(t, ok)
			},
		},
		{
			name: "Done returns nil when ctx is nil",
			run: func(t *testing.T, c *Context) {
				t.Helper()
				assert.Nil(t, c.Done())
			},
		},
		{
			name: "Err returns nil when ctx is nil",
			run: func(t *testing.T, c *Context) {
				t.Helper()
				assert.NoError(t, c.Err())
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var c Context
			tt.run(t, &c)
		})
	}
}

func TestCountAsInputIsRejected(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run",
		WithFlag("v", 0, WithCount()),
		WithRun(func(c *Context) error { return nil }),
	)

	// Count kind cannot be used as a positional arg.
	def := argDef{name: "v", valueType: vtCount()}
	assert.Error(t, def.validate())
}
