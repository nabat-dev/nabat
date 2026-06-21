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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/x/ansi"

	"nabat.dev/theme"
)

// RowState represents the lifecycle state of a [SpinnerRow].
type RowState int

const (
	// RowActive is the initial state: the row icon spins and the elapsed
	// timer ticks.
	RowActive RowState = iota

	// RowSuccess marks completed work: the spinner stops, the icon becomes
	// the success symbol, the row uses [theme.StatusSuccess] color, and the
	// timer freezes.
	RowSuccess

	// RowError marks a failure: the spinner stops, the icon becomes the
	// error symbol, the row uses [theme.StatusError] color, and the timer
	// freezes.
	RowError

	// RowWarning marks degraded or partial completion: the spinner stops,
	// the icon becomes the warning symbol, the row uses
	// [theme.StatusWarning] color, and the timer freezes.
	RowWarning

	// RowDone marks neutral completion: the spinner stops, the icon
	// becomes the done symbol, no special color is applied, and the timer
	// freezes.
	RowDone
)

// Icons configures the symbols shown for each terminal [RowState] in a
// [Spinner] live display. All fields fall back to Unicode symbols when left
// empty; only set the fields you want to override.
//
// Example:
//
//	WithSpinnerIcons(nabat.Icons{Success: "+", Error: "x"})
type Icons struct {
	// Success is the icon for [RowSuccess] rows. Default: "✓".
	Success string

	// Error is the icon for [RowError] rows. Default: "✗".
	Error string

	// Warning is the icon for [RowWarning] rows. Default: "!".
	Warning string

	// Done is the icon for [RowDone] rows. Default: "•".
	Done string
}

func (ic Icons) successIcon() string {
	if ic.Success != "" {
		return ic.Success
	}
	return "✓"
}

func (ic Icons) errorIcon() string {
	if ic.Error != "" {
		return ic.Error
	}
	return "✗"
}

func (ic Icons) warningIcon() string {
	if ic.Warning != "" {
		return ic.Warning
	}
	return "!"
}

func (ic Icons) doneIcon() string {
	if ic.Done != "" {
		return ic.Done
	}
	return "•"
}

// staticIcon returns the plain (unstyled) symbol for state.
func staticIcon(state RowState, icons Icons) string {
	switch state {
	case RowSuccess:
		return icons.successIcon()
	case RowError:
		return icons.errorIcon()
	case RowWarning:
		return icons.warningIcon()
	default:
		return icons.doneIcon()
	}
}

// SpinnerRow is a single keyed row in a [Spinner] live display. It tracks
// display cells, a [RowState], an optional style override, and an elapsed
// timer that starts when the row is created and freezes on any state
// transition.
//
// All exported methods are safe for concurrent use from multiple goroutines.
type SpinnerRow struct {
	mu       sync.Mutex
	key      string
	cells    []string
	state    RowState
	style    *theme.Token // nil means derive color from state
	created  time.Time
	frozenAt time.Duration // zero means timer is still ticking
}

// Set replaces the row's display cells. It is safe to call from any goroutine.
// Set does not change the row's [RowState]; use [SpinnerRow.Success],
// [SpinnerRow.Error], [SpinnerRow.Warn], or [SpinnerRow.Done] to mark
// completion.
func (r *SpinnerRow) Set(cells ...string) *SpinnerRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := append([]string(nil), cells...)
	r.cells = cp
	return r
}

// Success marks the row as successfully completed. The spinner stops, the icon
// becomes the success symbol (default "✓"), the row style becomes
// [theme.StatusSuccess], and the elapsed timer freezes. It is safe to call
// from any goroutine.
func (r *SpinnerRow) Success() *SpinnerRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowSuccess
	r.freezeTimer()
	return r
}

// Error marks the row as failed. The spinner stops, the icon becomes the error
// symbol (default "✗"), the row style becomes [theme.StatusError], and the
// elapsed timer freezes. It is safe to call from any goroutine.
func (r *SpinnerRow) Error() *SpinnerRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowError
	r.freezeTimer()
	return r
}

// Warn marks the row as a warning or partial failure. The spinner stops, the
// icon becomes the warning symbol (default "!"), the row style becomes
// [theme.StatusWarning], and the elapsed timer freezes. It is safe to call
// from any goroutine.
func (r *SpinnerRow) Warn() *SpinnerRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowWarning
	r.freezeTimer()
	return r
}

// Done marks the row with neutral completion. The spinner stops, the icon
// becomes the done symbol (default "•"), the elapsed timer freezes, and no
// special theme color is applied. It is safe to call from any goroutine.
func (r *SpinnerRow) Done() *SpinnerRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = RowDone
	r.freezeTimer()
	return r
}

// Style overrides the theme color used for this row. It does not change the
// row's [RowState] or icon; it only replaces the color applied on the next
// render. It is safe to call from any goroutine.
//
// Example:
//
//	row.Set("Pending", "quota reached").Style(theme.TextMuted)
func (r *SpinnerRow) Style(t theme.Token) *SpinnerRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.style = &t
	return r
}

// freezeTimer records the elapsed duration once. The caller must hold r.mu.
func (r *SpinnerRow) freezeTimer() {
	if r.frozenAt == 0 {
		r.frozenAt = time.Since(r.created)
	}
}

// rowSnapshot is a consistent point-in-time read of a [SpinnerRow].
type rowSnapshot struct {
	key     string
	cells   []string
	state   RowState
	style   *theme.Token
	elapsed time.Duration
}

// snapshot returns a consistent read of all fields under the lock.
func (r *SpinnerRow) snapshot() rowSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	cells := append([]string(nil), r.cells...)
	var elapsed time.Duration
	if r.frozenAt != 0 {
		elapsed = r.frozenAt
	} else {
		elapsed = time.Since(r.created)
	}
	var styleCopy *theme.Token
	if r.style != nil {
		t := *r.style
		styleCopy = &t
	}
	return rowSnapshot{
		key:     r.key,
		cells:   cells,
		state:   r.state,
		style:   styleCopy,
		elapsed: elapsed,
	}
}

// rowColumns returns the display columns for snap in render order:
// key, user cells, elapsed time. All values are raw (unstyled) strings.
func rowColumns(snap rowSnapshot) []string {
	cols := make([]string, 0, 2+len(snap.cells))
	cols = append(cols, snap.key)
	cols = append(cols, snap.cells...)
	cols = append(cols, formatElapsed(snap.elapsed))
	return cols
}

// computeColumnWidths returns the maximum visible width for each column
// position across all snapshots. The returned slice length equals the
// maximum number of columns seen in any single row.
func computeColumnWidths(snaps []rowSnapshot) []int {
	var widths []int
	for _, snap := range snaps {
		cols := rowColumns(snap)
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

// formatElapsed formats a duration for the timing column.
// Durations under a minute show one decimal place (e.g. "3.2s").
// Durations of a minute or more show minutes and whole seconds (e.g. "1m05s").
func formatElapsed(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}

// padRight pads s with trailing spaces so its visible character width equals
// width. If s is already at or beyond width, it is returned unchanged.
// ANSI escape codes in s are not counted toward the width.
func padRight(s string, width int) string {
	vw := visibleWidth(s)
	if vw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-vw)
}

// visibleWidth returns the number of terminal columns s occupies, ignoring
// ANSI escape sequences and accounting for East Asian wide characters.
func visibleWidth(s string) int {
	return ansi.StringWidth(s)
}
