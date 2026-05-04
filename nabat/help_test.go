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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpArgDefaultSuppression verifies that the args section of --help omits
// "(default: <zero>)" for every supported [ArgValue] kind. Adding a new kind
// means adding a row here and a case in [isZeroDefault].
func TestHelpArgDefaultSuppression(t *testing.T) {
	t.Parallel()

	type row struct {
		opt CommandOption
	}

	rows := []struct {
		label      string
		zero       row
		nonZero    CommandOption
		nonZeroVal string
	}{
		{
			label:      "string",
			zero:       row{opt: WithArg("v", "")},
			nonZero:    WithArg("v", "x"),
			nonZeroVal: "x",
		},
		{
			label:      "string-multiline",
			zero:       row{opt: WithArg("v", "")},
			nonZero:    WithArg("v", "body"),
			nonZeroVal: "body",
		},
		{
			label:      "string-file",
			zero:       row{opt: WithArg("v", "")},
			nonZero:    WithArg("v", "/tmp/x"),
			nonZeroVal: "/tmp/x",
		},
		{
			label:      "bool",
			zero:       row{opt: WithArg("v", false)},
			nonZero:    WithArg("v", true),
			nonZeroVal: "true",
		},
		{
			label:      "int",
			zero:       row{opt: WithArg("v", 0)},
			nonZero:    WithArg("v", 5),
			nonZeroVal: "5",
		},
		{
			label:      "int64",
			zero:       row{opt: WithArg("v", int64(0))},
			nonZero:    WithArg("v", int64(7)),
			nonZeroVal: "7",
		},
		{
			label:      "uint",
			zero:       row{opt: WithArg("v", uint(0))},
			nonZero:    WithArg("v", uint(9)),
			nonZeroVal: "9",
		},
		{
			label:      "float64",
			zero:       row{opt: WithArg("v", 0.0)},
			nonZero:    WithArg("v", 1.5),
			nonZeroVal: "1.5",
		},
		{
			label:      "duration",
			zero:       row{opt: WithArg("v", time.Duration(0))},
			nonZero:    WithArg("v", 30*time.Second),
			nonZeroVal: "30s",
		},
		{
			label:      "stringSlice",
			zero:       row{opt: WithArg("v", []string(nil))},
			nonZero:    WithArg("v", []string{"a"}),
			nonZeroVal: "[a]",
		},
		{
			// select with empty default requires WithRequired() so the
			// empty default is allowed by the spec validator.
			label:      "select",
			zero:       row{opt: WithSelectArg("v", "", []string{"a", "b"}, WithRequired())},
			nonZero:    WithSelectArg("v", "a", []string{"a", "b"}),
			nonZeroVal: "a",
		},
		{
			label:      "multiSelect",
			zero:       row{opt: WithMultiSelectArg("v", nil, []string{"a", "b"})},
			nonZero:    WithMultiSelectArg("v", []string{"a"}, []string{"a", "b"}),
			nonZeroVal: "[a]",
		},
	}

	for _, r := range rows {
		t.Run(r.label+"/zero", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			app, err := New("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
			require.NoError(t, err)
			app.MustCommand("run", r.zero.opt, WithRun(func(c *Context) error { return nil }))
			require.NoError(t, app.RunArgs(context.Background(), "run", "--help"))
			assert.NotContains(t, out.String(), "(default:",
				"%s zero default should not render", r.label)
		})

		t.Run(r.label+"/nonZero", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			app, err := New("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
			require.NoError(t, err)
			app.MustCommand("run", r.nonZero, WithRun(func(c *Context) error { return nil }))
			require.NoError(t, app.RunArgs(context.Background(), "run", "--help"))
			assert.Contains(t, out.String(), "(default: "+r.nonZeroVal+")",
				"%s non-zero default should render", r.label)
		})
	}
}

// TestHelpDeployRegression locks in the original report from the issue: a
// required select arg with an empty default no longer renders "(required)
// (default: )".
func TestHelpDeployRegression(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app, err := New("deployctl", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	require.NoError(t, err)
	app.MustCommand("deploy",
		WithSelectArg("environment", "", []string{"staging", "production"},
			WithRequired(),
			WithEnv("environment"),
		),
		WithRun(func(c *Context) error { return nil }),
	)
	require.NoError(t, app.RunArgs(context.Background(), "deploy", "--help"))

	got := out.String()
	assert.Contains(t, got, "(required)")
	assert.Contains(t, got, "(env: DEPLOYCTL_ENVIRONMENT)")
	assert.NotContains(t, got, "(default:")
}

// TestHelpHiddenFlagsRendering covers two related rules of the help renderer:
// user flags marked with [WithHidden] never appear, while the built-in
// --help flag (and its shorthand) IS visible — appearing under Flags: on the
// root and under Global Flags: on subcommands. The "Global Flags:" wording
// matches Cobra's default help template and Docker's CLI.
func TestHelpHiddenFlagsRendering(t *testing.T) {
	t.Parallel()

	t.Run("user_hidden_flag_omitted", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		app, err := New("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
		require.NoError(t, err)
		app.MustCommand("run",
			WithFlag("visible", false),
			WithFlag("secret", false, WithHidden()),
			WithRun(func(c *Context) error { return nil }),
		)
		require.NoError(t, app.RunArgs(context.Background(), "run", "--help"))

		got := out.String()
		assert.Contains(t, got, "--visible")
		assert.NotContains(t, got, "--secret")
	})

	t.Run("builtin_help_visible_on_root", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		app, err := New("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
		require.NoError(t, err)
		require.NoError(t, app.RunArgs(context.Background(), "--help"))

		got := out.String()
		assert.Contains(t, got, "--help, -h")
		assert.Contains(t, got, "show help for this command")
	})

	t.Run("builtin_help_visible_in_inherited_flags", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		app, err := New("test", WithIO(NewIO(strings.NewReader(""), &out, &out)))
		require.NoError(t, err)
		app.MustCommand("run",
			WithFlag("verbose", false, WithShort('v')),
			WithRun(func(c *Context) error { return nil }),
		)
		require.NoError(t, app.RunArgs(context.Background(), "run", "--help"))

		got := out.String()
		assert.Contains(t, got, "Global Flags:")
		assert.Contains(t, got, "--help, -h")
		assert.Contains(t, got, "--verbose, -v")
	})

	t.Run("renamed_help_visible_alias_hidden", func(t *testing.T) {
		t.Parallel()
		var out bytes.Buffer
		app, err := New("test",
			WithIO(NewIO(strings.NewReader(""), &out, &out)),
			WithHelpFlagName("info"),
			WithHelpShorthand('i'),
		)
		require.NoError(t, err)
		require.NoError(t, app.RunArgs(context.Background(), "--info"))

		got := out.String()
		// Primary (renamed) flag is visible.
		assert.Contains(t, got, "--info, -i")
		// The redundant --help alias stays hidden so it does not
		// duplicate the primary flag in rendered output.
		assert.NotContains(t, got, "--help")
	})
}
