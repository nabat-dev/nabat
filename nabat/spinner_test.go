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
	"errors"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

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

func TestSpinnerFastPathNoAnimation(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		io, _, _, stderr := testTTYIO()
		app := MustNew("test", WithIO(io))
		app.MustCommand("run", WithRun(func(c *Context) error {
			return c.Spinner(func(_ *Spinner) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			}, WithTitle("Quick"), WithSpinnerDelay(200*time.Millisecond))
		}))
		require.NoError(t, Run(t, app, []string{"run"}))

		out := stderr.String()
		assert.Contains(t, out, "✓")
		assert.Contains(t, out, "Quick")
		assert.NotContains(t, out, "\r", "fast path must not use carriage-return animation")
	})
}

func TestSpinnerSlowPathAnimates(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		io, _, _, stderr := testTTYIO()
		app := MustNew("test", WithIO(io))
		app.MustCommand("run", WithRun(func(c *Context) error {
			return c.Spinner(func(_ *Spinner) error {
				time.Sleep(150 * time.Millisecond)
				return nil
			}, WithTitle("Slow"), WithSpinnerDelay(20*time.Millisecond))
		}))
		require.NoError(t, Run(t, app, []string{"run"}))

		out := stderr.String()
		assert.Contains(t, out, "✓")
		assert.Contains(t, out, "Slow")
		assert.Contains(t, out, "\r", "slow path must animate with carriage returns")
	})
}

func TestSpinnerSetTextUpdatesHeader(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		io, _, _, stderr := testTTYIO()
		app := MustNew("test", WithIO(io))
		app.MustCommand("run", WithRun(func(c *Context) error {
			return c.Spinner(func(sp *Spinner) error {
				sp.SetText("Building")
				time.Sleep(80 * time.Millisecond)
				return nil
			}, WithTitle("Start"), WithSpinnerDelay(10*time.Millisecond))
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		assert.Contains(t, stderr.String(), "Building")
	})
}

func TestSpinnerErrorShowsXIcon(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		io, _, _, stderr := testTTYIO()
		app := MustNew("test", WithIO(io))
		var got error
		app.MustCommand("run", WithRun(func(c *Context) error {
			got = c.Spinner(func(_ *Spinner) error {
				return errors.New("boom")
			}, WithTitle("Fail"), WithSpinnerDelay(200*time.Millisecond))
			return nil
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		require.Error(t, got)
		assert.Contains(t, stderr.String(), "✗")
		assert.Contains(t, stderr.String(), "Fail")
	})
}

func TestSpinnerDelayOptionZeroStartsImmediately(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		io, _, _, stderr := testTTYIO()
		app := MustNew("test", WithIO(io))
		app.MustCommand("run", WithRun(func(c *Context) error {
			return c.Spinner(func(_ *Spinner) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			}, WithTitle("Now"), WithSpinnerDelay(0))
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		assert.Contains(t, stderr.String(), "\r",
			"WithSpinnerDelay(0) must start animation immediately")
	})
}

func TestSpinnerNegativeDelayRejected(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		got = c.Spinner(func(_ *Spinner) error { return nil },
			WithSpinnerDelay(-1))
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.Contains(t, got.Error(), "WithSpinnerDelay")
}

// failWriter always returns err from Write.
type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

// failOnceWriter fails the first Write, then succeeds.
type failOnceWriter struct {
	err  error
	once bool
}

func (w *failOnceWriter) Write(p []byte) (int, error) {
	if !w.once {
		w.once = true
		return 0, w.err
	}
	return len(p), nil
}

func TestSpinnerPropagatesLiveWriteError(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		writeErr := errors.New("stderr write failed")
		ios := NewIO(&bytes.Buffer{}, &bytes.Buffer{}, failWriter{err: writeErr})
		ios.SetStderrTTY(true)

		app := MustNew("test", WithIO(ios))
		var got error
		app.MustCommand("run", WithRun(func(c *Context) error {
			got = c.Spinner(func(_ *Spinner) error { return nil },
				WithTitle("Write"), WithSpinnerDelay(0))
			return nil
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		assert.ErrorIs(t, got, writeErr)
	})
}

func TestSpinnerLiveWriteErrorPreservedWhenFinalWriteSucceeds(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		writeErr := errors.New("live frame failed")
		ios := NewIO(&bytes.Buffer{}, &bytes.Buffer{}, &failOnceWriter{err: writeErr})
		ios.SetStderrTTY(true)

		app := MustNew("test", WithIO(ios))
		var got error
		app.MustCommand("run", WithRun(func(c *Context) error {
			got = c.Spinner(func(_ *Spinner) error { return nil },
				WithTitle("Write"), WithSpinnerDelay(0))
			return nil
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		assert.ErrorIs(t, got, writeErr,
			"live write error must surface even when the final line succeeds")
	})
}

func TestSpinnerFnErrorPreferredOverWriteError(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		writeErr := errors.New("stderr write failed")
		fnErr := errors.New("work failed")
		ios := NewIO(&bytes.Buffer{}, &bytes.Buffer{}, failWriter{err: writeErr})
		ios.SetStderrTTY(true)

		app := MustNew("test", WithIO(ios))
		var got error
		app.MustCommand("run", WithRun(func(c *Context) error {
			got = c.Spinner(func(_ *Spinner) error { return fnErr },
				WithTitle("Write"), WithSpinnerDelay(0))
			return nil
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		assert.ErrorIs(t, got, fnErr)
		assert.NotErrorIs(t, got, writeErr)
	})
}

func TestSpinnerContextCancellation(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		io, _, _, _ := testTTYIO()
		ctx, cancel := context.WithCancel(t.Context())
		app := MustNew("test", WithIO(io))
		var got error
		app.MustCommand("run", WithRun(func(c *Context) error {
			c.SetContext(ctx)
			got = c.Spinner(func(_ *Spinner) error {
				cancel()
				time.Sleep(30 * time.Millisecond)
				return nil
			}, WithTitle("Cancel"), WithSpinnerDelay(0))
			return nil
		}))
		require.NoError(t, Run(t, app, []string{"run"}))
		assert.ErrorIs(t, got, context.Canceled)
	})
}
