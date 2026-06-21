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

// TestStatusRowCreatesOnFirstCall asserts that [Status.Row] creates a new row
// on first call and returns the same pointer on subsequent calls with the same
// key.
func TestStatusRowCreatesOnFirstCall(t *testing.T) {
	t.Parallel()

	st := &Status{}
	r1 := st.Row("pod/api")
	r2 := st.Row("pod/api")

	require.NotNil(t, r1)
	assert.Same(t, r1, r2, "Row must return the same pointer for the same key")
}

// TestStatusRowDifferentKeys asserts that two different keys produce two
// distinct rows.
func TestStatusRowDifferentKeys(t *testing.T) {
	t.Parallel()

	st := &Status{}
	r1 := st.Row("pod/api")
	r2 := st.Row("pod/worker")

	assert.NotSame(t, r1, r2, "different keys must produce distinct rows")
}

// TestStatusRowEmptyKey asserts that an empty string is a valid key.
func TestStatusRowEmptyKey(t *testing.T) {
	t.Parallel()

	st := &Status{}
	r := st.Row("")
	require.NotNil(t, r)
	assert.Same(t, r, st.Row(""), "empty key must be stable")
}

// TestStatusRowInsertionOrder asserts that rows appear in insertion order when
// no priority is set.
func TestStatusRowInsertionOrder(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a")
	st.Row("b")
	st.Row("c")

	snaps := st.rowSnapshots()
	require.Len(t, snaps, 3)
	assert.Equal(t, "a", snaps[0].displayKey)
	assert.Equal(t, "b", snaps[1].displayKey)
	assert.Equal(t, "c", snaps[2].displayKey)
}

// TestStatusRowLabelOverridesKey asserts that [StatusRow.Label] sets the
// displayKey in the snapshot and overrides the internal key.
func TestStatusRowLabelOverridesKey(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("uid-123").Label("pod/api")

	snap := st.rowSnapshots()
	require.Len(t, snap, 1)
	assert.Equal(t, "pod/api", snap[0].displayKey)
}

// TestStatusRowLabelFallsBackToKey asserts that when Label is never called the
// key is used as the display value.
func TestStatusRowLabelFallsBackToKey(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("pod/api")

	snap := st.rowSnapshots()
	assert.Equal(t, "pod/api", snap[0].displayKey)
}

// TestStatusRowLabelEmptyStringValid asserts that Label("") is valid and
// renders as a blank first column.
func TestStatusRowLabelEmptyStringValid(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Label("")

	snap := row.snapshot()
	// displayKey must be the empty string (not key "k").
	assert.Equal(t, "", snap.displayKey)
}

// TestStatusRowLabelUpdatable asserts that Label can be called multiple times
// and the last value wins.
func TestStatusRowLabelUpdatable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	row.Label("first")
	row.Label("second")

	assert.Equal(t, "second", row.snapshot().displayKey)
}

// TestStatusRowLabelChainable asserts that Label returns the receiver.
func TestStatusRowLabelChainable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	ret := row.Label("name")

	assert.Same(t, row, ret)
}

// TestStatusRowSetCells asserts that [StatusRow.Set] stores cells and returns
// the receiver.
func TestStatusRowSetCells(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	ret := row.Set("Running", "healthy")

	assert.Same(t, row, ret, "Set must return the receiver")
	assert.Equal(t, []string{"Running", "healthy"}, row.snapshot().cells)
}

// TestStatusRowSetReplacesAllCells asserts that a second Set call replaces all
// cells.
func TestStatusRowSetReplacesAllCells(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	row.Set("first", "second")
	row.Set("only")

	assert.Equal(t, []string{"only"}, row.snapshot().cells)
}

// TestStatusRowIconOverridesStateDefault asserts that Icon sets the per-row
// override and resolvedIcon returns it.
func TestStatusRowIconOverridesStateDefault(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Icon("★")

	snap := row.snapshot()
	result := resolvedIcon(snap, Icons{})
	assert.Equal(t, "★", result)
}

// TestStatusRowIconClearWithEmpty asserts that Icon("") removes the override
// so the state default is used again.
func TestStatusRowIconClearWithEmpty(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Icon("★").Icon("")

	snap := row.snapshot()
	// icon field must be a pointer to "" (clear), not nil.
	require.NotNil(t, snap.icon)
	assert.Equal(t, "", *snap.icon)
	// resolvedIcon falls back to state default.
	result := resolvedIcon(snap, Icons{})
	assert.Equal(t, "•", result, "RowActive falls back to doneIcon in staticIcon")
}

// TestStatusRowIconChainable asserts that Icon returns the receiver.
func TestStatusRowIconChainable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	ret := row.Icon("*")

	assert.Same(t, row, ret)
}

// TestStatusRowIconUpdatable asserts that Icon can be called multiple times
// and the last value wins.
func TestStatusRowIconUpdatable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Icon("A").Icon("B")

	snap := row.snapshot()
	require.NotNil(t, snap.icon)
	assert.Equal(t, "B", *snap.icon)
}

// TestStatusRowStyleOverride asserts that [StatusRow.Style] stores the token
// and returns the receiver.
func TestStatusRowStyleOverride(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	ret := row.Style(theme.TextMuted)

	assert.Same(t, row, ret)
	snap := row.snapshot()
	require.NotNil(t, snap.style)
	assert.Equal(t, theme.TextMuted, *snap.style)
}

// TestStatusRowPrioritySort asserts that rows with lower priority numbers
// appear before rows with higher ones, before unprioritized rows.
func TestStatusRowPrioritySort(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a").Label("a") // no priority
	st.Row("b").Label("b").Priority(2)
	st.Row("c").Label("c").Priority(1)

	snaps := st.rowSnapshots()
	require.Len(t, snaps, 3)
	assert.Equal(t, "c", snaps[0].displayKey, "priority 1 must come first")
	assert.Equal(t, "b", snaps[1].displayKey, "priority 2 must come second")
	assert.Equal(t, "a", snaps[2].displayKey, "unprioritized must come last")
}

// TestStatusRowPriorityTieBreaksInsertionOrder asserts that equal priorities
// preserve insertion order.
func TestStatusRowPriorityTieBreaksInsertionOrder(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("x").Priority(1)
	st.Row("y").Priority(1)

	snaps := st.rowSnapshots()
	require.Len(t, snaps, 2)
	assert.Equal(t, "x", snaps[0].displayKey)
	assert.Equal(t, "y", snaps[1].displayKey)
}

// TestStatusRowPriorityUpdatable asserts that Priority can be changed after
// a row is created and the sort order updates on the next snapshot.
func TestStatusRowPriorityUpdatable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a").Priority(10)
	st.Row("b").Priority(1)

	// Swap priorities.
	st.Row("a").Priority(1)
	st.Row("b").Priority(10)

	snaps := st.rowSnapshots()
	assert.Equal(t, "a", snaps[0].displayKey, "a must now come first after priority update")
	assert.Equal(t, "b", snaps[1].displayKey)
}

// TestStatusRowPriorityChainable asserts that Priority returns the receiver.
func TestStatusRowPriorityChainable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	ret := row.Priority(5)

	assert.Same(t, row, ret)
}

// TestStatusRowHideExcludesFromSnapshots asserts that hidden rows do not
// appear in rowSnapshots.
func TestStatusRowHideExcludesFromSnapshots(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a")
	st.Row("b").Hide()
	st.Row("c")

	snaps := st.rowSnapshots()
	require.Len(t, snaps, 2)
	assert.Equal(t, "a", snaps[0].displayKey)
	assert.Equal(t, "c", snaps[1].displayKey)
}

// TestStatusRowShowMakesHiddenRowVisible asserts that Show restores a hidden
// row to the display.
func TestStatusRowShowMakesHiddenRowVisible(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a").Hide()
	st.Row("a").Show()

	snaps := st.rowSnapshots()
	require.Len(t, snaps, 1)
	assert.Equal(t, "a", snaps[0].displayKey)
}

// TestStatusRowHideChainable asserts that Hide and Show return the receiver.
func TestStatusRowHideShowChainable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	retHide := row.Hide()
	retShow := row.Show()

	assert.Same(t, row, retHide)
	assert.Same(t, row, retShow)
}

// TestStatusRowHideFreezesTimer asserts that calling Hide freezes the elapsed
// timer.
func TestStatusRowHideFreezesTimer(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	time.Sleep(5 * time.Millisecond)
	row.Hide()
	frozen := row.snapshot().elapsed

	time.Sleep(15 * time.Millisecond)
	assert.Equal(t, frozen, row.snapshot().elapsed, "timer must be frozen after Hide")
}

// TestStatusRowHideNotCountedForTruncation asserts hidden rows are not
// counted in rowSnapshots (they are pre-filtered).
func TestStatusRowHideNotCountedForTruncation(t *testing.T) {
	t.Parallel()

	st := &Status{}
	for i := range 5 {
		st.Row(string(rune('a' + i)))
	}
	st.Row("b").Hide()
	st.Row("d").Hide()

	snaps := st.rowSnapshots()
	assert.Len(t, snaps, 3, "only visible rows should appear in snapshots")
}

// TestStatusRowActiveResumesTimer asserts that Active clears the frozen timer
// so elapsed advances again.
func TestStatusRowActiveResumesTimer(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	time.Sleep(5 * time.Millisecond)
	row.Done()
	frozen := row.snapshot().elapsed

	row.Active()
	time.Sleep(10 * time.Millisecond)

	snap := row.snapshot()
	assert.Greater(t, snap.elapsed, frozen, "timer must advance again after Active")
}

// TestStatusRowActiveShowsHiddenRow asserts that Active makes a hidden row
// visible again.
func TestStatusRowActiveShowsHiddenRow(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("k").Hide()
	st.Row("k").Active()

	snaps := st.rowSnapshots()
	require.Len(t, snaps, 1)
}

// TestStatusRowActiveSetsState asserts that Active transitions back to
// RowActive.
func TestStatusRowActiveSetsState(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Success().Active()

	assert.Equal(t, RowActive, row.snapshot().state)
}

// TestStatusRowActiveChainable asserts that Active returns the receiver.
func TestStatusRowActiveChainable(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	ret := row.Active()

	assert.Same(t, row, ret)
}

// TestStatusRowSuccessFreezesTimer asserts that Success freezes the elapsed
// timer.
func TestStatusRowSuccessFreezesTimer(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	time.Sleep(10 * time.Millisecond)
	row.Success()

	frozen := row.snapshot().elapsed
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, frozen, row.snapshot().elapsed, "timer must not advance after Success")
}

// TestStatusRowErrorSetsState asserts Error transitions to RowError.
func TestStatusRowErrorSetsState(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Error()

	assert.Equal(t, RowError, row.snapshot().state)
}

// TestStatusRowWarnSetsState asserts Warn transitions to RowWarning.
func TestStatusRowWarnSetsState(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Warn()

	assert.Equal(t, RowWarning, row.snapshot().state)
}

// TestStatusRowDoneSetsNeutral asserts Done transitions to RowDone with no
// custom style.
func TestStatusRowDoneSetsNeutral(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k").Done()

	snap := row.snapshot()
	assert.Equal(t, RowDone, snap.state)
	assert.Nil(t, snap.style, "Done must not set a custom style")
}

// TestStatusRowStateTransitionDoesNotRefreeze asserts that a second terminal
// state call does not change the already-frozen elapsed time.
func TestStatusRowStateTransitionDoesNotRefreeze(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	time.Sleep(5 * time.Millisecond)
	row.Success()
	frozen := row.snapshot().elapsed

	time.Sleep(10 * time.Millisecond)
	row.Warn()
	assert.Equal(t, frozen, row.snapshot().elapsed,
		"second state transition must not change frozen elapsed time")
}

// TestStatusRowChainedMethods asserts method chaining works end-to-end.
func TestStatusRowChainedMethods(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("uid").Label("pod/api").Set("Running", "healthy").Priority(1).Success()

	snap := row.snapshot()
	assert.Equal(t, "pod/api", snap.displayKey)
	assert.Equal(t, []string{"Running", "healthy"}, snap.cells)
	assert.Equal(t, RowSuccess, snap.state)
}

// TestStatusRowAutoTiming asserts the elapsed timer ticks while active and
// stops after a state transition.
func TestStatusRowAutoTiming(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")
	time.Sleep(15 * time.Millisecond)

	snap1 := row.snapshot()
	assert.Greater(t, snap1.elapsed, time.Duration(0))

	row.Done()
	frozen := row.snapshot().elapsed
	time.Sleep(15 * time.Millisecond)
	assert.Equal(t, frozen, row.snapshot().elapsed, "elapsed must not advance after Done")
}

// TestRowColumnsIncludesElapsed asserts that rowColumns appends elapsed when
// noElapsed is false.
func TestRowColumnsIncludesElapsed(t *testing.T) {
	t.Parallel()

	snap := rowSnapshot{
		displayKey: "pod/api",
		cells:      []string{"Running"},
		elapsed:    time.Second,
	}
	cols := rowColumns(snap, false)
	require.Len(t, cols, 3, "displayKey + cell + elapsed")
	assert.Equal(t, "pod/api", cols[0])
	assert.Equal(t, "Running", cols[1])
	assert.Equal(t, "1.0s", cols[2])
}

// TestRowColumnsOmitsElapsed asserts that rowColumns omits elapsed when
// noElapsed is true.
func TestRowColumnsOmitsElapsed(t *testing.T) {
	t.Parallel()

	snap := rowSnapshot{
		displayKey: "pod/api",
		cells:      []string{"Running"},
		elapsed:    time.Second,
	}
	cols := rowColumns(snap, true)
	require.Len(t, cols, 2, "only displayKey + cell")
}

// TestComputeColumnWidthsWithElapsed asserts the per-column max widths across
// multiple snapshots.
func TestComputeColumnWidthsWithElapsed(t *testing.T) {
	t.Parallel()

	snaps := []rowSnapshot{
		{displayKey: "abc", cells: []string{"Running", "healthy"}, elapsed: time.Second},
		{displayKey: "pod/worker-xyz", cells: []string{"Failed"}, elapsed: time.Second},
	}
	w := computeColumnWidths(snaps, false)
	assert.Equal(t, 14, w[0], "col0: max of 'abc'(3) and 'pod/worker-xyz'(14)")
	assert.Equal(t, 7, w[1], "col1: max of 'Running'(7) and 'Failed'(6)")
	assert.Equal(t, 7, w[2], "col2: max of 'healthy'(7) and '1.0s'(4)")
	assert.Equal(t, 4, w[3], "col3: only from first row '1.0s'(4)")
}

// TestComputeColumnWidthsNoElapsed asserts widths omit the elapsed column
// when noElapsed is true.
func TestComputeColumnWidthsNoElapsed(t *testing.T) {
	t.Parallel()

	snaps := []rowSnapshot{
		{displayKey: "abc", cells: []string{"Running"}, elapsed: time.Second},
	}
	w := computeColumnWidths(snaps, true)
	assert.Len(t, w, 2, "displayKey + one cell, no elapsed")
}

// TestResolvedIconPerRowOverrideWins asserts that a non-empty per-row icon
// overrides state defaults.
func TestResolvedIconPerRowOverrideWins(t *testing.T) {
	t.Parallel()

	iconStr := "★"
	snap := rowSnapshot{state: RowError, icon: &iconStr}
	assert.Equal(t, "★", resolvedIcon(snap, Icons{}))
}

// TestResolvedIconEmptyOverrideFallsBackToState asserts that Icon("") falls
// back to the state default.
func TestResolvedIconEmptyOverrideFallsBackToState(t *testing.T) {
	t.Parallel()

	empty := ""
	snap := rowSnapshot{state: RowSuccess, icon: &empty}
	assert.Equal(t, "✓", resolvedIcon(snap, Icons{}))
}

// TestResolvedIconNilUsesStateDefault asserts that a nil icon field uses the
// state-derived default.
func TestResolvedIconNilUsesStateDefault(t *testing.T) {
	t.Parallel()

	snap := rowSnapshot{state: RowError}
	assert.Equal(t, "✗", resolvedIcon(snap, Icons{}))
}

// TestStatusRowConcurrentSafe asserts that concurrent calls to Set, Label,
// Icon, Priority, Hide, Show, Success, and Error on the same row are
// race-free. Run with -race.
func TestStatusRowConcurrentSafe(t *testing.T) {
	t.Parallel()

	st := &Status{}
	row := st.Row("k")

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		go func() {
			defer wg.Done()
			row.Label("label")
			row.Set("status", "detail")
			row.Priority(i % 3)
			switch {
			case i%4 == 0:
				row.Hide()
			case i%4 == 1:
				row.Show()
			case i%2 == 0:
				row.Success()
			default:
				row.Error()
			}
		}()
	}
	wg.Wait()
}

// TestStatusRowSnapshotConcurrentSafe asserts that concurrent calls to
// snapshot and rowSnapshots are race-free.
func TestStatusRowSnapshotConcurrentSafe(t *testing.T) {
	t.Parallel()

	st := &Status{}
	for i := range 5 {
		st.Row(string(rune('a' + i)))
	}

	var wg sync.WaitGroup
	wg.Add(10)
	for range 10 {
		go func() {
			defer wg.Done()
			_ = st.rowSnapshots()
		}()
	}
	wg.Wait()
}

// TestFormatElapsedStatus asserts the formatting logic for various durations.
func TestFormatElapsedStatus(t *testing.T) {
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

// TestPadRightStatus asserts that padRight pads and leaves longer strings
// unchanged.
func TestPadRightStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "abc  ", padRight("abc", 5))
	assert.Equal(t, "abcde", padRight("abcde", 5))
	assert.Equal(t, "abcdef", padRight("abcdef", 5), "no truncation")
}

// TestVisibleWidthStripsANSIStatus asserts that visibleWidth ignores ANSI
// sequences.
func TestVisibleWidthStripsANSIStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, visibleWidth("hello"))
	assert.Equal(t, 5, visibleWidth("\033[32mhello\033[0m"), "ANSI codes must not count")
}

// TestStaticIconStatus asserts staticIcon returns the correct symbol.
func TestStaticIconStatus(t *testing.T) {
	t.Parallel()

	icons := Icons{}
	assert.Equal(t, "✓", staticIcon(RowSuccess, icons))
	assert.Equal(t, "✗", staticIcon(RowError, icons))
	assert.Equal(t, "!", staticIcon(RowWarning, icons))
	assert.Equal(t, "•", staticIcon(RowDone, icons))
	assert.Equal(t, "•", staticIcon(RowActive, icons))
}

// TestStaticIconCustomStatus asserts custom icons override defaults.
func TestStaticIconCustomStatus(t *testing.T) {
	t.Parallel()

	icons := Icons{Success: "+", Error: "x", Warning: "~", Done: "-"}
	assert.Equal(t, "+", staticIcon(RowSuccess, icons))
	assert.Equal(t, "x", staticIcon(RowError, icons))
	assert.Equal(t, "~", staticIcon(RowWarning, icons))
	assert.Equal(t, "-", staticIcon(RowDone, icons))
}

// TestPadRightANSIAwareStatus asserts padRight works correctly with ANSI-
// styled text.
func TestPadRightANSIAwareStatus(t *testing.T) {
	t.Parallel()

	styled := "\033[32mhello\033[0m"
	padded := padRight(styled, 10)
	assert.Equal(t, 10, visibleWidth(padded),
		"padded string must have visible width 10")
}

// TestStatusSetCompletionOverridesFnError asserts that SetCompletion takes
// precedence over the fn error.
func TestStatusSetCompletionOverridesFnError(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.SetCompletion(RowSuccess)

	// Even with a non-nil error, the override wins.
	assert.Equal(t, RowSuccess, st.completionState(assert.AnError))
}

// TestStatusCompletionDeriveFromFnError asserts that when SetCompletion is
// not called, a non-nil fn error produces RowError.
func TestStatusCompletionDeriveFromFnError(t *testing.T) {
	t.Parallel()

	st := &Status{}
	assert.Equal(t, RowError, st.completionState(assert.AnError))
}

// TestStatusCompletionDeriveFromFnSuccess asserts that when SetCompletion is
// not called and fn returns nil, the state is RowSuccess.
func TestStatusCompletionDeriveFromFnSuccess(t *testing.T) {
	t.Parallel()

	st := &Status{}
	assert.Equal(t, RowSuccess, st.completionState(nil))
}

// TestStatusCompletionRowStatesDoNotBubble asserts that individual row error
// states do not bubble up to the header completion state.
func TestStatusCompletionRowStatesDoNotBubble(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a").Error()
	st.Row("b").Success()

	// fn returned nil; no SetCompletion call.
	assert.Equal(t, RowSuccess, st.completionState(nil),
		"row errors must not bubble up to header completion state")
}

// TestStatusAllRowSnapshotsIncludesHidden asserts that allRowSnapshots
// includes hidden rows.
func TestStatusAllRowSnapshotsIncludesHidden(t *testing.T) {
	t.Parallel()

	st := &Status{}
	st.Row("a")
	st.Row("b").Hide()

	all := st.allRowSnapshots()
	assert.Len(t, all, 2)
}

// TestHeaderColumnsAppendsAGE asserts that headerColumns appends an "AGE"
// column when noElapsed is false.
func TestHeaderColumnsAppendsAGE(t *testing.T) {
	t.Parallel()

	h := headerColumns([]string{"NAME", "STATUS"}, false)
	assert.Equal(t, []string{"NAME", "STATUS", "AGE"}, h)
}

// TestHeaderColumnsNoElapsed asserts that headerColumns omits AGE when
// noElapsed is true.
func TestHeaderColumnsNoElapsed(t *testing.T) {
	t.Parallel()

	h := headerColumns([]string{"NAME", "STATUS"}, true)
	assert.Equal(t, []string{"NAME", "STATUS"}, h)
}

// TestHeaderColumnsNilWhenEmpty asserts that nil is returned when columns is
// empty.
func TestHeaderColumnsNilWhenEmpty(t *testing.T) {
	t.Parallel()

	h := headerColumns(nil, false)
	assert.Nil(t, h)
}

// TestStatusModelVisibleRowsTruncatesInsertion asserts that statusModel
// truncates to insertionOrder (first maxRows visible) when no terminal height
// is set.
func TestStatusModelVisibleRowsNoHeight(t *testing.T) {
	t.Parallel()

	snaps := []rowSnapshot{
		{displayKey: "a"},
		{displayKey: "b"},
		{displayKey: "c"},
	}
	m := statusModel{height: 0}
	visible, hidden := m.visibleRows(snaps)

	assert.Len(t, visible, 3)
	assert.Equal(t, 0, hidden)
}

// TestStatusModelVisibleRowsTruncates asserts that rows are truncated to
// maxRows (height minus reserved lines).
func TestStatusModelVisibleRowsTruncates(t *testing.T) {
	t.Parallel()

	snaps := make([]rowSnapshot, 0, 10)
	for i := range 10 {
		snaps = append(snaps, rowSnapshot{displayKey: string(rune('a' + i))})
	}
	// height 6: reserve 3 => maxRows=3
	m := statusModel{height: 6, cfg: &statusConfig{}}
	visible, hidden := m.visibleRows(snaps)

	assert.Len(t, visible, 3)
	assert.Equal(t, 7, hidden)
	// Must preserve display order (first 3 by priority/insertion).
	assert.Equal(t, "a", visible[0].displayKey)
}

// TestStatusSetTitle asserts that SetTitle updates the header title.
func TestStatusSetTitle(t *testing.T) {
	t.Parallel()

	st := &Status{headerTitle: "initial"}
	st.SetTitle("updated")

	assert.Equal(t, "updated", st.getTitle())
}

func splitLinesStatus(s string) []string {
	var lines []string
	for _, l := range strings.SplitAfter(s, "\n") {
		if l != "" && l != "\n" {
			lines = append(lines, strings.TrimRight(l, "\n"))
		}
	}
	return lines
}
