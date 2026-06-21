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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// TestSpinnerRowCreatesOnFirstCall asserts that [Spinner.Row] creates a new
// row on the first call and returns the same pointer on subsequent calls with
// the same key.
func TestSpinnerRowCreatesOnFirstCall(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	r1 := sp.Row("pod/api")
	r2 := sp.Row("pod/api")

	require.NotNil(t, r1)
	assert.Same(t, r1, r2, "Row must return the same pointer for the same key")
}

// TestSpinnerRowDifferentKeys asserts that two different keys produce two
// distinct rows.
func TestSpinnerRowDifferentKeys(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	r1 := sp.Row("pod/api")
	r2 := sp.Row("pod/worker")

	assert.NotSame(t, r1, r2, "different keys must produce distinct rows")
}

// TestSpinnerRowEmptyKey asserts that an empty string is a valid key.
func TestSpinnerRowEmptyKey(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	r := sp.Row("")
	require.NotNil(t, r)
	assert.Same(t, r, sp.Row(""), "empty key must be stable")
}

// TestSpinnerRowSetCells asserts that [SpinnerRow.Set] stores the supplied
// cells and returns the same row pointer for chaining.
func TestSpinnerRowSetCells(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	ret := row.Set("Scheduled", "assigned to node-3")

	assert.Same(t, row, ret, "Set must return the receiver")
	snap := row.snapshot()
	assert.Equal(t, []string{"Scheduled", "assigned to node-3"}, snap.cells)
}

// TestSpinnerRowSetReplacesAllCells asserts that a second [SpinnerRow.Set]
// call replaces all cells, not appends to them.
func TestSpinnerRowSetReplacesAllCells(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	row.Set("first", "second")
	row.Set("only")

	snap := row.snapshot()
	assert.Equal(t, []string{"only"}, snap.cells)
}

// TestSpinnerRowInsertionOrder asserts that rows are stored in the order they
// were first created, regardless of later Set calls.
func TestSpinnerRowInsertionOrder(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	sp.Row("a")
	sp.Row("b")
	sp.Row("c")

	snaps := sp.rowSnapshots()
	require.Len(t, snaps, 3)
	assert.Equal(t, "a", snaps[0].key)
	assert.Equal(t, "b", snaps[1].key)
	assert.Equal(t, "c", snaps[2].key)
}

// TestSpinnerRowSuccessFreezesTimer asserts that [SpinnerRow.Success] sets the
// state, returns the receiver, and freezes the elapsed timer.
func TestSpinnerRowSuccessFreezesTimer(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	time.Sleep(10 * time.Millisecond)
	ret := row.Success()

	assert.Same(t, row, ret)
	snap := row.snapshot()
	assert.Equal(t, RowSuccess, snap.state)
	assert.Greater(t, snap.elapsed, time.Duration(0), "elapsed must be positive after freeze")

	frozen := snap.elapsed
	time.Sleep(20 * time.Millisecond)
	snap2 := row.snapshot()
	assert.Equal(t, frozen, snap2.elapsed, "elapsed must not advance after Success")
}

// TestSpinnerRowErrorSetsState asserts that [SpinnerRow.Error] transitions the
// row to [RowError] and freezes the timer.
func TestSpinnerRowErrorSetsState(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	ret := row.Error()

	assert.Same(t, row, ret)
	snap := row.snapshot()
	assert.Equal(t, RowError, snap.state)
	frozen := snap.elapsed
	time.Sleep(15 * time.Millisecond)
	assert.Equal(t, frozen, row.snapshot().elapsed, "timer must be frozen after Error")
}

// TestSpinnerRowWarnSetsState asserts that [SpinnerRow.Warn] transitions to
// [RowWarning].
func TestSpinnerRowWarnSetsState(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k").Warn()

	assert.Equal(t, RowWarning, row.snapshot().state)
}

// TestSpinnerRowDoneSetsNeutral asserts that [SpinnerRow.Done] transitions to
// [RowDone] with no special color.
func TestSpinnerRowDoneSetsNeutral(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k").Done()

	snap := row.snapshot()
	assert.Equal(t, RowDone, snap.state)
	assert.Nil(t, snap.style, "Done must not set a custom style")
}

// TestSpinnerRowStyleOverride asserts that [SpinnerRow.Style] stores the token
// and returns the receiver.
func TestSpinnerRowStyleOverride(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	ret := row.Style(theme.TextMuted)

	assert.Same(t, row, ret)
	snap := row.snapshot()
	require.NotNil(t, snap.style)
	assert.Equal(t, theme.TextMuted, *snap.style)
}

// TestSpinnerRowChainedMethods asserts that method chaining works end-to-end.
func TestSpinnerRowChainedMethods(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k").Set("Running", "healthy").Success()

	snap := row.snapshot()
	assert.Equal(t, []string{"Running", "healthy"}, snap.cells)
	assert.Equal(t, RowSuccess, snap.state)
}

// TestSpinnerRowConcurrentSafe asserts that concurrent calls to Set, Success,
// and Error on the same row are race-free. Run with -race.
func TestSpinnerRowConcurrentSafe(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func() {
			defer wg.Done()
			row.Set("status", "detail")
			if i%2 == 0 {
				row.Success()
			} else {
				row.Error()
			}
		}()
	}
	wg.Wait()
	// No assertion needed: the race detector is the verifier.
}

// TestSpinnerRowAutoTiming asserts that the elapsed time advances while the
// row is active and stops after a state transition.
func TestSpinnerRowAutoTiming(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	time.Sleep(15 * time.Millisecond)

	snap1 := row.snapshot()
	assert.Greater(t, snap1.elapsed, time.Duration(0))

	row.Done()
	frozen := row.snapshot().elapsed

	time.Sleep(15 * time.Millisecond)
	snap3 := row.snapshot()
	assert.Equal(t, frozen, snap3.elapsed, "elapsed must not advance after Done")
}

// TestFormatElapsed asserts the formatting logic for various durations.
func TestFormatElapsed(t *testing.T) {
	t.Parallel()

	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0.0s"},
		{-time.Second, "0.0s"},
		{500 * time.Millisecond, "0.5s"},
		{3*time.Second + 200*time.Millisecond, "3.2s"},
		{59*time.Second + 900*time.Millisecond, "59.9s"},
		{time.Minute, "1m00s"},
		{time.Minute + 5*time.Second, "1m05s"},
		{90 * time.Second, "1m30s"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, formatElapsed(tc.d), "formatElapsed(%v)", tc.d)
	}
}

// TestPadRight asserts that padRight pads to the requested visible width and
// leaves longer strings unchanged.
func TestPadRight(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc  ", padRight("abc", 5))
	assert.Equal(t, "abcde", padRight("abcde", 5))
	assert.Equal(t, "abcdef", padRight("abcdef", 5), "no truncation")
}

// TestVisibleWidthStripsANSI asserts that visibleWidth counts visible columns
// and ignores ANSI escape sequences.
func TestVisibleWidthStripsANSI(t *testing.T) {
	t.Parallel()

	plain := "hello"
	styled := "\033[32mhello\033[0m"
	assert.Equal(t, 5, visibleWidth(plain))
	assert.Equal(t, 5, visibleWidth(styled), "ANSI codes must not count toward width")
}

// TestComputeColumnWidths asserts that column widths are the per-column maxima
// across all provided snapshots.
func TestComputeColumnWidths(t *testing.T) {
	t.Parallel()

	snaps := []rowSnapshot{
		{key: "abc", cells: []string{"Running", "healthy"}, elapsed: time.Second},
		{key: "pod/worker-xyz", cells: []string{"Failed"}, elapsed: time.Second},
	}
	w := computeColumnWidths(snaps)
	// col0 = key: max("abc"=3, "pod/worker-xyz"=14) = 14
	assert.Equal(t, 14, w[0])
	// col1 = cell0: max("Running"=7, "Failed"=6) = 7
	assert.Equal(t, 7, w[1])
	// col2 = cell1 or elapsed: max("healthy"=7, "1.0s"=4) = 7
	assert.Equal(t, 7, w[2])
	// col3 = elapsed for first row: "1.0s"=4; second row has no cell1 so col3 only from row 0
	assert.Equal(t, 4, w[3])
}

// TestSpinnerRowNonTTYPrintsTable asserts that the non-TTY path prints the
// plain-text row table to ErrOut after fn returns.
func TestSpinnerRowNonTTYPrintsTable(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner("Deploying", func(sp *Spinner) error {
			sp.Row("pod/api").Set("Running", "healthy").Success()
			sp.Row("pod/worker").Set("Failed", "OOMKilled").Error()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "Deploying", "initial title must appear")
	assert.Contains(t, out, "pod/api", "row key must appear in table")
	assert.Contains(t, out, "Running", "cell must appear in table")
	assert.Contains(t, out, "healthy", "cell must appear in table")
	assert.Contains(t, out, "pod/worker", "second row key must appear")
	assert.Contains(t, out, "Failed", "second row cell must appear")
	assert.Contains(t, out, "OOMKilled", "second row detail must appear")
	// Icon symbols must appear.
	assert.Contains(t, out, "✓", "success icon must appear for RowSuccess")
	assert.Contains(t, out, "✗", "error icon must appear for RowError")
}

// TestSpinnerRowNonTTYNoRowsNoTable asserts that when no rows are added the
// non-TTY output is just the initial title (matching legacy behavior).
func TestSpinnerRowNonTTYNoRowsNoTable(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner("Loading", func(_ *Spinner) error {
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Equal(t, "Loading\n", out, "no rows means only the title is printed")
}

// TestSpinnerHeaderAutoCompletionSuccess asserts that when fn returns nil and
// all rows are in RowSuccess the header completion state is RowSuccess.
func TestSpinnerHeaderAutoCompletionSuccess(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	sp.Row("a").Success()
	sp.Row("b").Success()

	assert.Equal(t, RowSuccess, sp.completionState(nil))
}

// TestSpinnerHeaderAutoCompletionError asserts that a non-nil fnErr produces
// RowError regardless of row states.
func TestSpinnerHeaderAutoCompletionError(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	sp.Row("a").Success()

	assert.Equal(t, RowError, sp.completionState(assert.AnError))
}

// TestSpinnerHeaderAutoCompletionRowError asserts that when fn returns nil but
// a row is in RowError the header state is RowError.
func TestSpinnerHeaderAutoCompletionRowError(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	sp.Row("a").Success()
	sp.Row("b").Error()

	assert.Equal(t, RowError, sp.completionState(nil))
}

// TestSpinnerHeaderAutoCompletionRowWarn asserts that when fn returns nil and
// no rows are errors but one is a warning the header state is RowWarning.
func TestSpinnerHeaderAutoCompletionRowWarn(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	sp.Row("a").Success()
	sp.Row("b").Warn()

	assert.Equal(t, RowWarning, sp.completionState(nil))
}

// TestSpinnerRowColumnAlignment asserts that the plain-text table renders
// columns aligned across rows of different widths (no ANSI effects on spacing).
func TestSpinnerRowColumnAlignment(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner("Align test", func(sp *Spinner) error {
			sp.Row("short").Set("Running").Success()
			sp.Row("a-much-longer-key").Set("Failed").Error()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	lines := splitLines(out)
	// Skip the header line; find the two data lines.
	var dataLines []string
	for _, l := range lines {
		if len(l) > 0 && (containsAny(l, "short", "a-much-longer-key")) {
			dataLines = append(dataLines, l)
		}
	}
	require.Len(t, dataLines, 2, "must have exactly two data lines")

	// The key column of the second line must be padded so the status column
	// starts at the same offset in both lines.
	// We verify this by checking that "Running" and "Failed" start at the same
	// position in their respective lines.
	pos1 := indexAfterKey(dataLines[0], "short")
	pos2 := indexAfterKey(dataLines[1], "a-much-longer-key")
	assert.Equal(t, pos1, pos2, "status column must start at the same byte offset in both rows")
}

// TestSpinnerRowCustomIcons asserts that [WithSpinnerIcons] changes the static
// symbols in the non-TTY table output.
func TestSpinnerRowCustomIcons(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner("Deploy", func(sp *Spinner) error {
			sp.Row("svc").Set("ok").Success()
			return nil
		}, WithSpinnerIcons(Icons{Success: "+"}))
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	assert.Contains(t, out, "+", "custom success icon must appear")
	assert.NotContains(t, out, "✓", "default success icon must not appear")
}

// TestSpinnerSetTextWithRowsCoexist asserts that [Spinner.SetText] updates
// the header independently from row state.
func TestSpinnerSetTextWithRowsCoexist(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		return c.Spinner("Phase 1", func(sp *Spinner) error {
			sp.Row("item").Set("Working")
			sp.SetText("Phase 2")
			sp.Row("item").Set("Done").Success()
			return nil
		})
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	out := stderr.String()

	// In non-TTY mode only the initial title appears (Phase 1), but rows appear.
	assert.Contains(t, out, "Phase 1", "initial title must be printed")
	assert.Contains(t, out, "item", "row key must appear")
}

// TestSpinnerRowANSIAwareWidth asserts that column alignment is correct even
// when cell text contains ANSI escape codes.
func TestSpinnerRowANSIAwareWidth(t *testing.T) {
	t.Parallel()

	// Verify padRight handles ANSI-escaped text correctly.
	styled := "\033[32mhello\033[0m" // "hello" in green
	padded := padRight(styled, 10)
	// Visible width of styled is 5; we should get 5 trailing spaces.
	assert.Equal(t, 10, visibleWidth(padded),
		"padded string must have visible width 10")
}

// TestSpinnerRowWarnDoesNotTransitionAlreadyFrozenTimer asserts that calling
// Warn after Success does not change the frozen time.
func TestSpinnerRowStateTransitionDoesNotRefreeze(t *testing.T) {
	t.Parallel()

	sp := &Spinner{}
	row := sp.Row("k")
	time.Sleep(5 * time.Millisecond)
	row.Success()
	frozen := row.snapshot().elapsed

	time.Sleep(10 * time.Millisecond)
	row.Warn() // second transition
	assert.Equal(t, frozen, row.snapshot().elapsed,
		"a second state transition must not change the frozen elapsed time")
}

// TestSpinnerRowNonTTYPropagatesFnError asserts that an error from fn is
// returned even when rows are present.
func TestSpinnerRowNonTTYPropagatesFnError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		got = c.Spinner("Work", func(sp *Spinner) error {
			sp.Row("a").Set("started")
			return assert.AnError
		})
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.ErrorIs(t, got, assert.AnError,
		"Spinner must propagate fn error when rows are present")
}

// TestSpinnerRowNonTTYTableAppearsAfterFnError asserts that the row table is
// still printed when fn returns an error.
func TestSpinnerRowNonTTYTableAppearsAfterFnError(t *testing.T) {
	t.Parallel()

	io, _, _, stderr := testIO()
	app := MustNew("test", WithIO(io))
	var spinErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		spinErr = c.Spinner("Work", func(sp *Spinner) error {
			sp.Row("item").Set("in progress")
			return assert.AnError
		})
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	assert.ErrorIs(t, spinErr, assert.AnError)
	assert.Contains(t, stderr.String(), "item",
		"row table must be printed even when fn returns an error")
}

// TestStaticIcon asserts that staticIcon returns the correct symbol for each
// state.
func TestStaticIcon(t *testing.T) {
	t.Parallel()

	icons := Icons{}
	assert.Equal(t, "✓", staticIcon(RowSuccess, icons))
	assert.Equal(t, "✗", staticIcon(RowError, icons))
	assert.Equal(t, "!", staticIcon(RowWarning, icons))
	assert.Equal(t, "•", staticIcon(RowDone, icons))
	assert.Equal(t, "•", staticIcon(RowActive, icons))
}

// TestStaticIconCustom asserts that custom icons override the defaults.
func TestStaticIconCustom(t *testing.T) {
	t.Parallel()

	icons := Icons{Success: "+", Error: "x", Warning: "~", Done: "-"}
	assert.Equal(t, "+", staticIcon(RowSuccess, icons))
	assert.Equal(t, "x", staticIcon(RowError, icons))
	assert.Equal(t, "~", staticIcon(RowWarning, icons))
	assert.Equal(t, "-", staticIcon(RowDone, icons))
}

// TestSpinnerViewHeaderHasSpaceAfterIcon asserts that when the spinner
// completes, the header renders with a space between the completion icon and
// the title text (e.g. "✓ Checking services", not "✓Checking services").
func TestSpinnerViewHeaderHasSpaceAfterIcon(t *testing.T) {
	t.Parallel()

	m := spinnerModel{
		handle: &Spinner{text: "Checking services"},
		done:   true,
	}
	view := m.View().Content
	// The header must have a space after the icon before the title.
	assert.NotContains(t, view, "✓Checking",
		"success icon must be followed by a space before the title")
	assert.NotContains(t, view, "✗Checking",
		"error icon must be followed by a space before the title")
	assert.NotContains(t, view, "!Checking",
		"warning icon must be followed by a space before the title")
}

// TestSpinnerRowSmartTruncation asserts that [spinnerModel.visibleRows]
// prioritizes errors and active rows over completed ones when the display
// height is limited.
func TestSpinnerRowSmartTruncation(t *testing.T) {
	t.Parallel()

	snaps := []rowSnapshot{
		{key: "done-1", state: RowDone},
		{key: "done-2", state: RowDone},
		{key: "done-3", state: RowDone},
		{key: "active", state: RowActive},
		{key: "error", state: RowError},
	}

	m := spinnerModel{height: 6} // reserves 3 for header+summary+margin => maxRows=3
	visible, hidden := m.visibleRows(snaps)

	require.Len(t, visible, 3)
	assert.Equal(t, 2, hidden)

	var keys []string
	for _, s := range visible {
		keys = append(keys, s.key)
	}
	// Error and active rows must always be included.
	assert.Contains(t, keys, "error")
	assert.Contains(t, keys, "active")
}

func splitLines(s string) []string {
	var lines []string
	for _, l := range strings.SplitAfter(s, "\n") {
		if l != "" && l != "\n" {
			lines = append(lines, strings.TrimRight(l, "\n"))
		}
	}
	return lines
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// indexAfterKey returns the byte index at which the content after key starts.
// It finds key in line and returns the index of the first non-space character
// after it. Used to compare column alignment across rows.
func indexAfterKey(line, key string) int {
	idx := strings.Index(line, key)
	if idx < 0 {
		return -1
	}
	after := idx + len(key)
	for after < len(line) && line[after] == ' ' {
		after++
	}
	return after
}
