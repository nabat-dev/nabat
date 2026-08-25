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

// TestStatusNonTTYTitlePrinted asserts that the title appears once in stderr
// when WithTitle is set.
func TestStatusNonTTYTitlePrinted(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(_ *Status) error {
			return nil
		}, WithTitle("Deploying"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Contains(t, stderr.String(), "Deploying")
}

// TestStatusNonTTYNoTitleNoOutput asserts that without WithTitle no title is
// printed before the row table.
func TestStatusNonTTYNoTitleNoOutput(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(_ *Status) error {
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "", stderr.String(), "no output when no title and no rows")
}

// TestStatusNonTTYRowTablePrinted asserts that the row table is printed after
// fn returns.
func TestStatusNonTTYRowTablePrinted(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("pod/api").Set("Running", "healthy").Success()
			st.Row("pod/worker").Set("Failed", "OOMKilled").Error()
			return nil
		}, WithTitle("Deploy"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "pod/api")
	assert.Contains(t, out, "Running")
	assert.Contains(t, out, "healthy")
	assert.Contains(t, out, "pod/worker")
	assert.Contains(t, out, "Failed")
	assert.Contains(t, out, "OOMKilled")
	assert.Contains(t, out, "✓", "success icon must appear")
	assert.Contains(t, out, "✗", "error icon must appear")
}

// TestStatusNonTTYNoRowsNoTable asserts that when no rows are added no table
// is printed.
func TestStatusNonTTYNoRowsNoTable(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(_ *Status) error {
			return nil
		}, WithTitle("Loading"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "Loading\n", stderr.String(), "only title, no table")
}

// TestStatusNonTTYLabelAppears asserts that the label overrides the key in
// the rendered table.
func TestStatusNonTTYLabelAppears(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("uid-123").Label("my-deployment").Set("Running").Success()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "my-deployment", "label must appear instead of key")
	assert.NotContains(t, out, "uid-123", "internal key must not appear")
}

// TestStatusNonTTYColumnHeadersAppear asserts that column headers appear in
// the non-TTY table output.
func TestStatusNonTTYColumnHeadersAppear(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("k").Set("Running").Success()
			return nil
		}, WithColumns("OBJECT", "STATUS"), WithTitle("Events"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "OBJECT")
	assert.Contains(t, out, "STATUS")
	assert.Contains(t, out, "AGE", "auto-appended AGE header must appear")
}

// TestStatusNonTTYWithoutElapsedNoAGE asserts that WithoutElapsed removes the
// AGE column and its header.
func TestStatusNonTTYWithoutElapsedNoAGE(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("k").Set("Running").Success()
			return nil
		}, WithColumns("OBJECT", "STATUS"), WithoutElapsed())
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "OBJECT")
	assert.Contains(t, out, "STATUS")
	assert.NotContains(t, out, "AGE", "AGE must not appear when WithoutElapsed is set")
}

// TestStatusNonTTYCustomIconsAppear asserts that WithStatusIcons changes the
// symbols in the table.
func TestStatusNonTTYCustomIconsAppear(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("k").Set("ok").Success()
			return nil
		}, WithStatusIcons(Icons{Success: "+"}))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "+", "custom success icon must appear")
	assert.NotContains(t, out, "✓", "default success icon must not appear")
}

// TestStatusNonTTYPerRowIconOverride asserts that per-row Icon override
// appears in the table.
func TestStatusNonTTYPerRowIconOverride(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("k").Set("ok").Icon("★").Success()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "★", "per-row icon override must appear")
}

// TestStatusNonTTYHiddenRowsExcluded asserts that hidden rows do not appear
// in the table.
func TestStatusNonTTYHiddenRowsExcluded(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("visible").Set("ok").Success()
			st.Row("hidden").Set("secret").Success().Hide()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "visible")
	assert.NotContains(t, out, "hidden")
	assert.NotContains(t, out, "secret")
}

// TestStatusNonTTYPropagatesFnError asserts fn error is returned.
func TestStatusNonTTYPropagatesFnError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		got = c.Status(func(_ *Status) error {
			return assert.AnError
		})
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.ErrorIs(t, got, assert.AnError)
}

// TestStatusNonTTYTableAppearsAfterFnError asserts that the row table is
// still printed even when fn returns an error.
func TestStatusNonTTYTableAppearsAfterFnError(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		err := c.Status(func(st *Status) error {
			st.Row("item").Set("in progress")
			return assert.AnError
		})
		assert.Error(t, err)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Contains(t, stderr.String(), "item",
		"row table must be printed even when fn returns an error")
}

// TestStatusNonTTYColumnAlignment asserts that the plain-text table renders
// columns aligned across rows of different widths.
func TestStatusNonTTYColumnAlignment(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("short").Set("Running").Success()
			st.Row("a-much-longer-key").Set("Failed").Error()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	lines := splitLinesStatus(out)
	var dataLines []string
	for _, l := range lines {
		if len(l) > 0 && containsAnyStatus(l, "short", "a-much-longer-key") {
			dataLines = append(dataLines, l)
		}
	}
	require.Len(t, dataLines, 2, "must have exactly two data lines")

	pos1 := indexAfterKeyStatus(dataLines[0], "short")
	pos2 := indexAfterKeyStatus(dataLines[1], "a-much-longer-key")
	assert.Equal(t, pos1, pos2, "status column must start at the same byte offset")
}

// TestStatusFnAlwaysCalled asserts that fn is always invoked.
func TestStatusFnAlwaysCalled(t *testing.T) {
	t.Parallel()

	var called bool
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(_ *Status) error {
			called = true
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.True(t, called)
}

// TestStatusWithSpinnerTypeOption asserts that WithSpinnerType can be passed
// to Context.Status without error.
func TestStatusWithSpinnerTypeOption(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(_ *Status) error {
			return nil
		}, WithSpinnerType(SpinnerDots()))
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
}

// TestStatusWithSpinnerIconsOption asserts that WithSpinnerIcons can be passed
// to Context.Status without error.
func TestStatusWithSpinnerIconsOption(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("k").Success()
			return nil
		}, WithSpinnerIcons(Icons{Success: "+"}))
	}))
	require.NoError(t, Run(t, app, []string{"run"}))

	// Non-TTY output should show the custom icon.
	io2, _, _, stderr2 := testIO()
	app2 := MustNew("test", WithIO(io2))
	app2.MustCommand("run", WithRun(func(c *Context) error {
		return c.Status(func(st *Status) error {
			st.Row("k").Success()
			return nil
		}, WithSpinnerIcons(Icons{Success: "+"}))
	}))
	require.NoError(t, Run(t, app2, []string{"run"}))
	assert.Contains(t, stderr2.String(), "+")
}

func containsAnyStatus(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) > 0 {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func indexAfterKeyStatus(line, key string) int {
	idx := -1
	for i := 0; i <= len(line)-len(key); i++ {
		if line[i:i+len(key)] == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return -1
	}
	after := idx + len(key)
	for after < len(line) && line[after] == ' ' {
		after++
	}
	return after
}
