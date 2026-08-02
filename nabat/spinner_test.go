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
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSpinnerRunsFnWhenStderrNotTTY asserts that when stderr is not a terminal
// [Context.Spinner] runs fn and prints the initial title once to ErrOut without
// attempting animation.
func TestSpinnerRunsFnWhenStderrNotTTY(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()

	ran := false
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner(func(_ *Spinner) error {
			ran = true
			return nil
		}, WithTitle("Building"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.True(t, ran, "fn must run in the non-TTY fallback path")
	assert.Contains(t, stderr.String(), "Building",
		"the initial title must be printed to ErrOut in the non-TTY path")
}

// TestSpinnerSetTextIsNoOpInFallback asserts that calling [Spinner.SetText]
// inside fn does not panic and does not produce extra output on ErrOut when
// stderr is not a terminal.
func TestSpinnerSetTextIsNoOpInFallback(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()

	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner(func(sp *Spinner) error {
			sp.SetText("Step 2")
			sp.SetText("Step 3")
			return nil
		}, WithTitle("Step 1"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))

	// Only the initial title should appear; SetText updates are silent.
	out := stderr.String()
	assert.Equal(t, 1, strings.Count(out, "\n"),
		"only one line (the initial title) should be written to ErrOut")
	assert.Contains(t, out, "Step 1")
	assert.NotContains(t, out, "Step 2")
	assert.NotContains(t, out, "Step 3")
}

// TestSpinnerPropagatesFnError asserts that an error returned by fn is
// returned unchanged by [Context.Spinner].
func TestSpinnerPropagatesFnError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("build failed")
	io, _, _, _ := testIO()

	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		got = c.Spinner(func(_ *Spinner) error {
			return sentinel
		}, WithTitle("Building"))
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.ErrorIs(t, got, sentinel,
		"Spinner must propagate fn's error unchanged")
}

// TestSpinnerAggregatesOptionErrors asserts that a nil [SpinnerOption] entry
// causes [Context.Spinner] to return a [*ConfigErrors] containing
// [ErrInvalidOption] and [ErrNilOption] before running fn.
func TestSpinnerAggregatesOptionErrors(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()

	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		got = c.Spinner(func(_ *Spinner) error { return nil }, nil)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)

	var cfgErr *ConfigErrors
	require.ErrorAs(t, got, &cfgErr,
		"nil spinner option must return *ConfigErrors")
	assert.True(t, errors.Is(got, ErrInvalidOption))
	assert.True(t, errors.Is(got, ErrNilOption))
}

// TestSpinnerSetTextConcurrentIsSafe asserts that concurrent calls to
// [Spinner.SetText] from multiple goroutines are race-free. Run with -race.
func TestSpinnerSetTextConcurrentIsSafe(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()

	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner(func(sp *Spinner) error {
			const goroutines = 16
			var wg sync.WaitGroup
			for i := range goroutines {
				wg.Go(func() {
					sp.SetText(strings.Repeat("x", i))
				})
			}
			wg.Wait()
			return nil
		}, WithTitle("Working"))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
}

// TestSpinnerEmptyTitleSkipsFallbackLine asserts that when the title is empty
// and stderr is not a terminal, no line is written to ErrOut.
func TestSpinnerEmptyTitleSkipsFallbackLine(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()

	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner(func(_ *Spinner) error { return nil })
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Empty(t, stderr.String(),
		"empty title must not produce any ErrOut output")
}
