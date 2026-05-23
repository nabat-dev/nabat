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
	"log/slog"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"charm.land/lipgloss/v2/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

// TestOutputStreamRouting locks in the POSIX-correct routing: Success, Info,
// Warn, and Error all go to stderr (diagnostics). Print/Println/Printf and
// the structured/encoded output methods carry the command's product on stdout.
func TestOutputStreamRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		action       func(*nabat.Context)
		wantInStdout string
		wantInStderr string
		notInStdout  string
		notInStderr  string
	}{
		{
			name:         "Success goes to stderr",
			action:       func(c *nabat.Context) { c.Success("done") },
			wantInStderr: "done",
			notInStdout:  "done",
		},
		{
			name:         "Info goes to stderr",
			action:       func(c *nabat.Context) { c.Info("hello") },
			wantInStderr: "hello",
			notInStdout:  "hello",
		},
		{
			name:         "Println goes to stdout",
			action:       func(c *nabat.Context) { c.Println("line") },
			wantInStdout: "line",
			notInStderr:  "line",
		},
		{
			name:         "Warn goes to stderr (POSIX)",
			action:       func(c *nabat.Context) { c.Warn("careful") },
			wantInStderr: "careful",
			notInStdout:  "careful",
		},
		{
			name:         "Error goes to stderr (POSIX)",
			action:       func(c *nabat.Context) { c.Error("nope") },
			wantInStderr: "nope",
			notInStdout:  "nope",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, stdout, stderr := nabattest.NewIO()
			app := nabat.MustNew("test", nabat.WithIO(io))
			act := tt.action
			app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
				act(c)
				return nil
			}))
			require.NoError(t, nabattest.Run(t, app, []string{"run"}))
			if tt.wantInStdout != "" {
				assert.Contains(t, stdout.String(), tt.wantInStdout, "expected substring in stdout")
			}
			if tt.wantInStderr != "" {
				assert.Contains(t, stderr.String(), tt.wantInStderr, "expected substring in stderr")
			}
			if tt.notInStdout != "" {
				assert.NotContains(t, stdout.String(), tt.notInStdout, "substring should not appear in stdout")
			}
			if tt.notInStderr != "" {
				assert.NotContains(t, stderr.String(), tt.notInStderr, "substring should not appear in stderr")
			}
		})
	}
}

func TestContextOutputHelpers(t *testing.T) {
	t.Parallel()

	// stream selects which buffer the assertion targets. Success/Warn/Error/Info
	// land on stderr; Print/Printf land on stdout.
	const (
		streamStdout = "stdout"
		streamStderr = "stderr"
	)

	tests := []struct {
		name        string
		action      func(*nabat.Context)
		stream      string
		contains    []string
		notContains []string
	}{
		{
			name: "Success prints symbol and message with fields",
			action: func(c *nabat.Context) {
				c.Success("done", "key", "val")
			},
			stream:   streamStderr,
			contains: []string{"✓", "done"},
		},
		{
			name:     "Warn prints semantic label and message",
			action:   func(c *nabat.Context) { c.Warn("caution") },
			stream:   streamStderr,
			contains: []string{"⚠", "caution"},
		},
		{
			name: "Error prints error symbol and values",
			action: func(c *nabat.Context) {
				c.Error("failed", "reason", "timeout")
			},
			stream:   streamStderr,
			contains: []string{"✗", "timeout"},
		},
		{
			name:     "Info prints semantic label and KV",
			action:   func(c *nabat.Context) { c.Info("status", "phase", "ready") },
			stream:   streamStderr,
			contains: []string{"•", "status"},
		},
		{
			name:     "Print writes plain message",
			action:   func(c *nabat.Context) { c.Print("hello world") },
			stream:   streamStdout,
			contains: []string{"hello world"},
		},
		{
			name:     "Printf formats plain message",
			action:   func(c *nabat.Context) { c.Printf("count=%d", 42) },
			stream:   streamStdout,
			contains: []string{"count=42"},
		},
		{
			name: "Success skips pairs with empty key",
			action: func(c *nabat.Context) {
				c.Success("ok", "", "hidden", "visible", "yes")
			},
			stream:      streamStderr,
			contains:    []string{"visible"},
			notContains: []string{"hidden"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, stdout, stderr := nabattest.NewIO()
			app := nabat.MustNew("test", nabat.WithIO(io))
			act := tt.action
			app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
				act(c)
				return nil
			}))
			require.NoError(t, nabattest.Run(t, app, []string{"test"}))
			got := stdout.String()
			if tt.stream == streamStderr {
				got = stderr.String()
			}
			for _, sub := range tt.contains {
				assert.Contains(t, got, sub)
			}
			for _, sub := range tt.notContains {
				assert.NotContains(t, got, sub)
			}
		})
	}
}

func TestTableOutput(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		c.Table([]string{"Name", "Age"}, [][]string{{"Alice", "30"}})
		return nil
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Alice")
}

func TestTableWithOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		c.Table([]string{"Name", "Age"}, [][]string{{"Bob", "25"}},
			nabat.WithTableBorder(nabat.BorderRounded()),
			nabat.WithTableWidth(40),
		)
		return nil
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Bob")
}

func TestListOutput(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		c.List([]string{"one", "two", "three"})
		return nil
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	got := stdout.String()
	assert.Contains(t, got, "one")
	assert.Contains(t, got, "two")
}

func TestTreeOutput(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		c.Tree("root", []nabat.TreeNode{{Value: "child1"}, {Value: "child2"}})
		return nil
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	got := stdout.String()
	assert.Contains(t, got, "root")
	assert.Contains(t, got, "child1")
}

func TestJSONOutput(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		return c.JSON(map[string]string{"key": "value"})
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	got := stdout.String()
	assert.Contains(t, got, `"key"`)
	assert.Contains(t, got, `"value"`)
}

func TestContextLoggerWithoutExtension(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		assert.False(t, c.Logger().Enabled(t.Context(), slog.LevelError))
		assert.NotSame(t, slog.Default(), c.Logger())
		return nil
	}))

	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestMarkdownNonTTYRendersContent(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		return c.Markdown("# Hello\n\nThis is **markdown**.\n")
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "Hello")
	assert.Contains(t, stdout.String(), "markdown")
}

func TestMarkdownEmptyContentNoOutput(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		return c.Markdown("")
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Empty(t, stdout.String())
}

func TestProgressBarNonTTYPrintsPlainLines(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		bar, err := c.ProgressBar(3)
		if err != nil {
			return err
		}
		bar.Increment()
		bar.Increment()
		bar.Done()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
	assert.NotEmpty(t, lines)
	for _, line := range lines {
		assert.Regexp(t, `\[\d+/3\]`, line)
	}
}

func TestProgressBarSetClampsToTotal(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		bar, err := c.ProgressBar(5)
		if err != nil {
			return err
		}
		bar.Set(999)
		bar.Done()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stderr.String(), "[5/5]")
}

func TestProgressBarAddClampsToTotal(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		bar, err := c.ProgressBar(3)
		if err != nil {
			return err
		}
		bar.Add(100)
		bar.Done()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stderr.String(), "[3/3]")
}

func TestTableBorderStyleOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Table(
			[]string{"Name", "Age"},
			[][]string{{"Alice", "30"}},
			nabat.WithTableBorderStyle(lipgloss.NewStyle()),
			nabat.WithTableHeaderStyle(lipgloss.NewStyle()),
			nabat.WithTableCellStyle(lipgloss.NewStyle()),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "Alice")
}

func TestTableStyleFuncAndWrapOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Table(
			[]string{"Name"},
			[][]string{{"Bob"}},
			nabat.WithTableStyleFunc(func(row, col int) lipgloss.Style { return lipgloss.NewStyle() }),
			nabat.WithTableWrap(true),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "Bob")
}

func TestTableBorderSideOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Table(
			[]string{"X"},
			[][]string{{"1"}},
			nabat.WithTableBorderTop(false),
			nabat.WithTableBorderBottom(false),
			nabat.WithTableBorderLeft(false),
			nabat.WithTableBorderRight(false),
			nabat.WithTableBorderHeader(false),
			nabat.WithTableBorderColumn(false),
			nabat.WithTableBorderRow(false),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "X")
}

func TestListStyleOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.List(
			[]string{"item1", "item2"},
			nabat.WithListEnumerator(list.Bullet),
			nabat.WithListItemStyle(lipgloss.NewStyle()),
			nabat.WithListEnumeratorStyle(lipgloss.NewStyle()),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "item1")
}

func TestListStyleFuncOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.List(
			[]string{"a", "b"},
			nabat.WithListItemStyleFunc(func(items list.Items, index int) lipgloss.Style {
				return lipgloss.NewStyle()
			}),
			nabat.WithListEnumeratorStyleFunc(func(items list.Items, index int) lipgloss.Style {
				return lipgloss.NewStyle()
			}),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "a")
}

func TestTreeStyleOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		rootStyle := lipgloss.NewStyle()
		itemStyle := lipgloss.NewStyle()
		enumStyle := lipgloss.NewStyle()
		indentStyle := lipgloss.NewStyle()
		c.Tree("root", []nabat.TreeNode{{Value: "child"}},
			nabat.WithTreeEnumerator(tree.DefaultEnumerator),
			nabat.WithTreeIndenter(tree.DefaultIndenter),
			nabat.WithTreeRootStyle(rootStyle),
			nabat.WithTreeItemStyle(itemStyle),
			nabat.WithTreeEnumeratorStyle(enumStyle),
			nabat.WithTreeIndenterStyle(indentStyle),
			nabat.WithTreeWidth(80),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "root")
}

func TestTreeStyleFuncOptions(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Tree("root", []nabat.TreeNode{{Value: "child"}},
			nabat.WithTreeItemStyleFunc(func(children tree.Children, i int) lipgloss.Style {
				return lipgloss.NewStyle()
			}),
			nabat.WithTreeEnumeratorStyleFunc(func(children tree.Children, i int) lipgloss.Style {
				return lipgloss.NewStyle()
			}),
			nabat.WithTreeIndenterStyleFunc(func(children tree.Children, i int) lipgloss.Style {
				return lipgloss.NewStyle()
			}),
		)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "child")
}

func TestTreeWithNestedChildren(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		c.Tree("root", []nabat.TreeNode{
			{
				Value: "parent",
				Children: []nabat.TreeNode{
					{Value: "grandchild"},
				},
			},
		})
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stdout.String(), "grandchild")
}

func TestProgressBarWithWidthOption(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		bar, err := c.ProgressBar(3, nabat.WithProgressBarWidth(60))
		if err != nil {
			return err
		}
		bar.Increment()
		bar.Done()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, stderr.String(), "[1/3]")
}

func TestWithSpinnerTypeOption(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		return c.Spinner("working", func() error {
			return nil
		}, nabat.WithSpinnerType(nabat.SpinnerDots()))
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
}

func TestSpinnerNonInteractive(t *testing.T) {
	t.Parallel()

	var called bool
	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		return c.Spinner("loading", func() error {
			called = true
			return nil
		})
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestSpinnerPropagatesError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test", nabat.WithRun(func(c *nabat.Context) error {
		return c.Spinner("loading", func() error {
			return context.DeadlineExceeded
		})
	}))

	err := nabattest.Run(t, app, []string{"test"})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestSpinnerClosureCapturesContext(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("test",
		nabat.WithArg("env", "staging"),
		nabat.WithRun(func(c *nabat.Context) error {
			return c.Spinner("working", func() error {
				env, err := nabat.BindAs[string](c, "env")
				require.NoError(t, err)
				c.Print(env)
				return nil
			})
		}),
	)

	err := nabattest.Run(t, app, []string{"test"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "staging")
}
