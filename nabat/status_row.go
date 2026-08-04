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
	"time"

	"nabat.dev/theme"
)

// StatusRow is a single keyed row in a [Status] live display. The key from
// [Status.Row] identifies the row for dedup but is not shown; use
// [StatusRow.Label] for the visible first column. All exported methods are
// safe for concurrent use.
//
// Example:
//
//	row.Label(name).Set(cells...).Success()
type StatusRow struct {
	mu       sync.Mutex
	key      string
	label    *string // nil = use key as column 0
	icon     *string // nil = derive from state; "" = clear override
	cells    []string
	state    RowState
	style    *theme.Token // nil = derive from state
	priority *int         // nil = unprioritized
	hidden   bool
	created  time.Time
	frozen   bool          // true once a terminal/hide call freezes the timer
	frozenAt time.Duration // elapsed at freeze; meaningful only when frozen
}

// Label sets the visible first column. When nil (never called), the key is
// shown instead. Calling Label with an empty string is valid and renders
// column 0 as blank. It is safe to call from any goroutine; last write wins.
func (r *StatusRow) Label(s string) *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.label = &s
	return r
}

// Set replaces the row's display cells. It does not change the row's
// [RowState] or visibility; use [StatusRow.Success], [StatusRow.Error],
// [StatusRow.Warn], [StatusRow.Done], or [StatusRow.Active] to change state.
// It is safe to call from any goroutine; last write wins.
func (r *StatusRow) Set(cells ...string) *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cells = append([]string(nil), cells...)
	return r
}

// Icon overrides the symbol shown at the start of the row. The override wins
// over the state-derived default (see [Icons]). Call Icon("") to clear the
// override and fall back to the state default. It is safe to call from any
// goroutine; last write wins.
func (r *StatusRow) Icon(s string) *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.icon = &s
	return r
}

// Style overrides the theme color used for this row. It does not change the
// row's [RowState] or icon. It is safe to call from any goroutine; last
// write wins.
//
// Example:
//
//	row.Set("Pending", "quota reached").Style(theme.TextMuted)
func (r *StatusRow) Style(t theme.Token) *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.style = &t
	return r
}

// Priority sets the row's sort position. Rows with a lower number appear
// before rows with a higher number. Rows without a priority appear after all
// prioritized rows in insertion order. It is safe to call from any goroutine;
// last write wins.
func (r *StatusRow) Priority(n int) *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.priority = &n
	return r
}

// Success marks the row completed: stops animation, success icon,
// [theme.StatusSuccess] style, and freezes the elapsed timer. Goroutine-safe.
func (r *StatusRow) Success() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowSuccess
	r.freezeTimer()
	return r
}

// Error marks the row failed: stops animation, error icon, [theme.StatusError]
// style, and freezes the elapsed timer. Goroutine-safe.
func (r *StatusRow) Error() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowError
	r.freezeTimer()
	return r
}

// Warn marks the row as a warning: stops animation, warning icon,
// [theme.StatusWarning] style, and freezes the elapsed timer. Goroutine-safe.
func (r *StatusRow) Warn() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowWarning
	r.freezeTimer()
	return r
}

// Done marks neutral completion: stops animation, done icon, freezes the
// elapsed timer, no special theme color. Goroutine-safe.
func (r *StatusRow) Done() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowDone
	r.freezeTimer()
	return r
}

// Active returns the row to [RowActive]: restarts animation, resumes the
// elapsed timer, and shows the row if hidden. Goroutine-safe.
func (r *StatusRow) Active() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowActive
	r.resumeTimer()
	r.hidden = false
	return r
}

// Hide removes the row from the display without deleting it from the dedup
// map. Restore with [StatusRow.Show] or [StatusRow.Active]. Goroutine-safe.
func (r *StatusRow) Hide() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hidden = true
	r.freezeTimer()
	return r
}

// Show makes a hidden row visible again. It does not change the row's
// [RowState] or timer. It is safe to call from any goroutine.
func (r *StatusRow) Show() *StatusRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hidden = false
	return r
}

// freezeTimer records the elapsed duration if the timer is still ticking.
// The caller must hold r.mu. A freeze at zero elapsed is preserved via
// [StatusRow.frozen] so the timer does not keep advancing.
func (r *StatusRow) freezeTimer() {
	if !r.frozen {
		r.frozenAt = time.Since(r.created)
		r.frozen = true
	}
}

// resumeTimer clears the frozen marker so the timer ticks again.
// The caller must hold r.mu.
func (r *StatusRow) resumeTimer() {
	r.frozen = false
	r.frozenAt = 0
}

// rowSnapshot is a consistent point-in-time read of a [StatusRow].
type rowSnapshot struct {
	displayKey string // label if set, else key
	cells      []string
	state      RowState
	icon       *string      // nil = derive from state
	style      *theme.Token // nil = derive from state
	elapsed    time.Duration
}

// snapshot returns a consistent read of all visible fields under the lock.
// It resolves displayKey from label or key so renderers need no fallback logic.
func (r *StatusRow) snapshot() rowSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	dk := r.key
	if r.label != nil {
		dk = *r.label
	}

	cells := append([]string(nil), r.cells...)

	var elapsed time.Duration
	if r.frozen {
		elapsed = r.frozenAt
	} else {
		elapsed = time.Since(r.created)
	}

	var styleCopy *theme.Token
	if r.style != nil {
		t := *r.style
		styleCopy = &t
	}

	var iconCopy *string
	if r.icon != nil {
		s := *r.icon
		iconCopy = &s
	}

	return rowSnapshot{
		displayKey: dk,
		cells:      cells,
		state:      r.state,
		icon:       iconCopy,
		style:      styleCopy,
		elapsed:    elapsed,
	}
}

// rowColumns returns the display columns for snap in render order:
// displayKey, user cells, elapsed time. All values are raw (unstyled) strings.
// When noElapsed is true, the elapsed column is omitted.
func rowColumns(snap rowSnapshot, noElapsed bool) []string {
	cap := 1 + len(snap.cells)
	if !noElapsed {
		cap++
	}
	cols := make([]string, 0, cap)
	cols = append(cols, snap.displayKey)
	cols = append(cols, snap.cells...)
	if !noElapsed {
		cols = append(cols, formatElapsed(snap.elapsed))
	}
	return cols
}

// computeColumnWidths returns the maximum visible width for each column
// position across all snapshots. The returned slice length equals the maximum
// number of columns seen in any single row.
func computeColumnWidths(snaps []rowSnapshot, noElapsed bool) []int {
	var widths []int
	for _, snap := range snaps {
		cols := rowColumns(snap, noElapsed)
		for i, col := range cols {
			w := visibleWidth(col)
			if i >= len(widths) {
				widths = append(widths, w)
			} else if w > widths[i] {
				widths[i] = w
			}
		}
	}
	return widths
}

// resolvedIcon returns the icon string for a snapshot, checking the per-row
// override first, then falling back to the state-derived icon from icons.
func resolvedIcon(snap rowSnapshot, icons Icons) string {
	if snap.icon != nil && *snap.icon != "" {
		return *snap.icon
	}
	return staticIcon(snap.state, icons)
}
