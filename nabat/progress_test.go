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
	"sync"
	"testing"

	"charm.land/bubbles/v2/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// TestProgressBar_validates asserts that [Context.ProgressBar] aggregates option
// errors and a non-positive total into a [ConfigErrors] without constructing the
// bar, mirroring the validation contract of [Context.Select] and friends.
func TestProgressBar_validates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		run    func(c *Context) error
		substr string
	}{
		{
			name: "non-positive total returns error",
			run: func(c *Context) error {
				_, err := c.ProgressBar(0)
				return err
			},
			substr: "total must be > 0",
		},
		{
			name: "WithProgressBarWidth zero is rejected",
			run: func(c *Context) error {
				_, err := c.ProgressBar(3, WithProgressBarWidth(0))
				return err
			},
			substr: "WithProgressBarWidth",
		},
		{
			name: "nil option is rejected",
			run: func(c *Context) error {
				_, err := c.ProgressBar(3, nil)
				return err
			},
			substr: "progress bar option",
		},
		{
			name: "WithProgressBarSpring rejects non-positive frequency",
			run: func(c *Context) error {
				_, err := c.ProgressBar(3, WithProgressBarSpring(0, 1))
				return err
			},
			substr: "WithProgressBarSpring",
		},
		{
			name: "WithProgressBarSpring rejects non-positive damping",
			run: func(c *Context) error {
				_, err := c.ProgressBar(3, WithProgressBarSpring(1, 0))
				return err
			},
			substr: "WithProgressBarSpring",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			io, _, _, _ := testIO()
			app := MustNew("test", WithIO(io))
			app.MustCommand("run", WithRun(tc.run))
			err := Run(t, app, []string{"run"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.substr)
		})
	}
}

// TestProgressBar_clampsNegativeSteps asserts [ProgressBar.Add] and
// [ProgressBar.Set] keep the internal position within [0, total].
func TestProgressBar_clampsNegativeSteps(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		bar, err := c.ProgressBar(10)
		require.NoError(t, err)

		bar.Set(5)
		bar.Add(-100)
		assert.Equal(t, 0, bar.current, "large negative Add clamps to 0")

		bar.Set(-1)
		assert.Equal(t, 0, bar.current, "negative Set clamps to 0")

		bar.Set(3)
		bar.Add(-1)
		assert.Equal(t, 2, bar.current, "moderate negative Add subtracts within range")

		bar.Add(100)
		assert.Equal(t, 10, bar.current, "large positive Add clamps to total")

		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
}

// TestProgressBarConcurrentIncrement asserts that concurrent calls to
// [ProgressBar.Increment] from many goroutines are race-free and produce the
// expected final count. Run with -race to catch missing synchronization on
// the internal current/total fields.
func TestProgressBarConcurrentIncrement(t *testing.T) {
	t.Parallel()

	const goroutines = 32
	const incrementsPerGoroutine = 100
	total := goroutines * incrementsPerGoroutine

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("test", WithRun(func(c *Context) error {
		bar, err := c.ProgressBar(total)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(goroutines)
		for range goroutines {
			go func() {
				defer wg.Done()
				for range incrementsPerGoroutine {
					bar.Increment()
				}
			}()
		}
		wg.Wait()
		bar.Done()

		assert.Equal(t, total, bar.current,
			"all increments should be recorded")
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"test"}))
}

// TestProgressBar_hidePercentageTTY asserts [WithoutProgressBarPercentage]
// omits the numeric percentage when stderr is a terminal.
func TestProgressBar_hidePercentageTTY(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	io.SetStderrTTY(true)

	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		bar, err := c.ProgressBar(4, WithoutProgressBarPercentage())
		require.NoError(t, err)
		bar.Increment()
		bar.Done()
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()
	assert.NotContains(t, out, "%")
}

// TestBuildProgress_modelUsesThemeEmptyColor asserts [buildProgressModel] assigns
// [theme.TextMuted] to the bubbles empty segment when stderr is a TTY and the user
// did not pass [WithoutProgressBarTheme].
func TestBuildProgress_modelUsesThemeEmptyColor(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	io.SetStderrTTY(true)

	var model progress.Model
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		model = buildProgressModel(c, &progressBarConfig{width: 40})
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))

	rt := app.Theme()
	assert.Equal(t,
		rt.Style(theme.TextMuted).GetForeground(),
		model.EmptyColor,
		"empty track color should follow theme TextMuted",
	)
}

// TestBuildProgress_modelWithoutThemeLeavesLibraryDefaults asserts
// [WithoutProgressBarTheme] skips post-New theme field assignments so empty
// coloring stays the bubbles library default instead of [theme.TextMuted].
func TestBuildProgress_modelWithoutThemeLeavesLibraryDefaults(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	io.SetStderrTTY(true)

	var model progress.Model
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		cfg := &progressBarConfig{width: 40, withoutTheme: true}
		model = buildProgressModel(c, cfg)
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))

	rt := app.Theme()
	assert.NotEqual(t,
		rt.Style(theme.TextMuted).GetForeground(),
		model.EmptyColor,
		"without-theme path should not assign TextMuted to EmptyColor",
	)
}
