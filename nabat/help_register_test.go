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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newHelpTestApp builds a help-test app with a fresh IOStreams bundle. The
// returned stdout/stderr buffers carry the streams the help renderer (stdout)
// and other diagnostics (stderr) would land on.
func newHelpTestApp(t *testing.T, opts ...Option) (*App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	io, _, stdout, stderr := testIO()
	all := append([]Option{WithIO(io)}, opts...)
	app, err := New("myctl", all...)
	require.NoError(t, err)
	return app, stdout, stderr
}

func TestHelpDefaultFlagInstalled(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newHelpTestApp(t)
	app.MustCommand("run", WithRun(func(c *Context) error { return nil }))

	require.NoError(t, app.RunArgs(context.Background(), "run", "--help"))
	assert.Contains(t, stdout.String(), "Usage:")
}

func TestHelpDefaultSubcommandAbsent(t *testing.T) {
	t.Parallel()

	app, _, _ := newHelpTestApp(t)
	app.MustCommand("run", WithRun(func(c *Context) error { return nil }))

	err := app.RunArgs(context.Background(), "help", "run")
	require.Error(t, err, "help subcommand should be absent without WithHelpCommand")
}

func TestHelpDefaultFlagWorks(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newHelpTestApp(t)
	app.MustCommand("run", WithRun(func(c *Context) error { return nil }))

	require.NoError(t, app.RunArgs(context.Background(), "run", "--help"))
	assert.Contains(t, stdout.String(), "Usage:")
}

func TestWithHelpCommandInstallsSubcommand(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newHelpTestApp(t, WithHelpCommand())
	app.MustCommand("run", WithRun(func(c *Context) error { return nil }))

	require.NoError(t, app.RunArgs(context.Background(), "help", "run"))
	assert.Contains(t, stdout.String(), "run")
}

func TestWithHelpCommandName(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newHelpTestApp(t, WithHelpCommand(WithHelpCommandName("aide")))
	app.MustCommand("run", WithRun(func(c *Context) error { return nil }))

	require.NoError(t, app.RunArgs(context.Background(), "aide", "run"))
	assert.Contains(t, stdout.String(), "run")
}

func TestWithHelpFlagName(t *testing.T) {
	t.Parallel()

	app, _, _ := newHelpTestApp(t,
		WithHelpFlagName("aide"),
		WithHelpShorthand('a'),
	)

	require.NotNil(t, app.UnsafeRoot().PersistentFlags().Lookup("aide"))
	require.NotNil(t, app.UnsafeRoot().PersistentFlags().ShorthandLookup("a"))
}

func TestWithHelpFlagNameInvokesRenderer(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newHelpTestApp(t, WithHelpFlagName("info"))
	app.MustCommand("run", WithRun(func(c *Context) error {
		t.Fatal("run handler should not execute when help flag is set")
		return nil
	}))

	require.NoError(t, app.RunArgs(context.Background(), "run", "--info"))
	assert.Contains(t, stdout.String(), "Usage:")
}

func TestWithoutHelpShorthand(t *testing.T) {
	t.Parallel()

	app, err := New("myctl", WithoutHelpShorthand())
	require.NoError(t, err)

	assert.Nil(t, app.UnsafeRoot().PersistentFlags().ShorthandLookup("h"))
	assert.NotNil(t, app.UnsafeRoot().PersistentFlags().Lookup("help"))
}

func TestWithoutHelpDisablesEverything(t *testing.T) {
	t.Parallel()

	app, err := New("myctl", WithoutHelp())
	require.NoError(t, err)

	root := app.UnsafeRoot()
	f := root.PersistentFlags().Lookup("help")
	if f != nil {
		assert.NotEqual(t, "show help for this command", f.Usage)
	}
}

func TestWithoutHelpFlagKeepsSubcommand(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newHelpTestApp(t, WithoutHelpFlag(), WithHelpCommand())
	app.MustCommand("run", WithRun(func(c *Context) error { return nil }))

	require.NoError(t, app.RunArgs(context.Background(), "help", "run"))
	assert.Contains(t, stdout.String(), "Usage:")
}

func TestWithHelpCommandNameEmpty(t *testing.T) {
	t.Parallel()

	_, err := New("myctl", WithHelpCommand(WithHelpCommandName("")))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestWithHelpCommandRejectsNilSubOption(t *testing.T) {
	t.Parallel()

	_, err := New("myctl", WithHelpCommand(nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestWithHelpFlagNameEmpty(t *testing.T) {
	t.Parallel()

	_, err := New("myctl", WithHelpFlagName(""))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestWithoutHelpRejectsCombinationAfter(t *testing.T) {
	t.Parallel()

	_, err := New("myctl", WithHelpCommand(), WithoutHelp())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestWithoutHelpRejectsCombinationBefore(t *testing.T) {
	t.Parallel()

	_, err := New("myctl", WithoutHelp(), WithHelpFlagName("info"))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestWithoutHelpShorthandRejectsCombinationWithShorthand(t *testing.T) {
	t.Parallel()

	_, err := New("myctl", WithHelpShorthand('?'), WithoutHelpShorthand())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestWithoutHelpAlone(t *testing.T) {
	t.Parallel()

	app, err := New("myctl", WithoutHelp())
	require.NoError(t, err)
	require.NotNil(t, app)
}
