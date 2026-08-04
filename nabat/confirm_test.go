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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmWithYesBypasses(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var ok bool
	app.MustCommand("run", WithRun(func(c *Context) error {
		var err error
		ok, err = c.Confirm("Delete?", WithYes(true))
		return err
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.True(t, ok)
}

func TestConfirmWithYesFalseDoesNotBypass(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Confirm("Delete?", WithYes(false))
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.ErrorIs(t, got, ErrConfirmationRequired)
}

func TestConfirmNonInteractiveReturnsConfirmationError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Confirm("Delete release?")
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)

	var ce *ConfirmationError
	require.ErrorAs(t, got, &ce)
	assert.Equal(t, "Delete release?", ce.Prompt)
	assert.ErrorIs(t, got, ErrConfirmationRequired)
}

func TestConfirmBypassHintInError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Confirm("Delete?", WithBypassHint("--yes"))
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.Contains(t, got.Error(), "--yes")
}

func TestConfirmWithConfirmValueMatchProceeds(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var ok bool
	app.MustCommand("run", WithRun(func(c *Context) error {
		var err error
		ok, err = c.Confirm("Destroy prod?",
			WithConfirmValue("production"),
			WithConfirmInput("production"),
		)
		return err
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.True(t, ok)
}

func TestConfirmWithConfirmValueMismatchErrors(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Confirm("Destroy prod?",
			WithConfirmValue("production"),
			WithConfirmInput("staging"),
			WithBypassHint("--confirm=production"),
		)
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.ErrorIs(t, got, ErrConfirmationRequired)
	assert.Contains(t, got.Error(), "--confirm=production")
}

func TestConfirmWithYesDoesNotBypassTypeToConfirm(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Confirm("Destroy prod?",
			WithYes(true),
			WithConfirmValue("production"),
			WithConfirmInput("staging"),
			WithBypassHint("--confirm=production"),
		)
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.ErrorIs(t, got, ErrConfirmationRequired)
	assert.Contains(t, got.Error(), "--confirm=production")
}

func TestConfirmWithDefaultIgnoredWhenConfirmValueSet(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Confirm("Destroy prod?",
			WithConfirmValue("production"),
			WithDefault(true),
		)
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.ErrorIs(t, got, ErrConfirmationRequired)
}

func TestConfirmWithDefaultStillWorksNonInteractive(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var ok bool
	app.MustCommand("run", WithRun(func(c *Context) error {
		var err error
		ok, err = c.Confirm("Continue?", WithDefault(false))
		return err
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.False(t, ok)
}

func TestConfirmationErrorUnwrap(t *testing.T) {
	t.Parallel()
	err := &ConfirmationError{Prompt: "x", BypassHint: "--yes"}
	assert.True(t, errors.Is(err, ErrConfirmationRequired))
	assert.Equal(t, "nabat: confirmation required (pass --yes to proceed)", err.Error())
}
