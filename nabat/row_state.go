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
	"time"

	"github.com/charmbracelet/x/ansi"
)

// RowState represents the lifecycle state of a [StatusRow].
type RowState int

const (
	// RowActive is the initial state: the row icon animates and the elapsed
	// timer ticks.
	RowActive RowState = iota

	// RowSuccess marks completed work: the animation stops, the icon becomes
	// the success symbol, the row uses [theme.StatusSuccess] color, and the
	// timer freezes.
	RowSuccess

	// RowError marks a failure: the animation stops, the icon becomes the
	// error symbol, the row uses [theme.StatusError] color, and the timer
	// freezes.
	RowError

	// RowWarning marks degraded or partial completion: the animation stops,
	// the icon becomes the warning symbol, the row uses [theme.StatusWarning]
	// color, and the timer freezes.
	RowWarning

	// RowDone marks neutral completion: the animation stops, the icon
	// becomes the done symbol, no special color is applied, and the timer
	// freezes.
	RowDone
)

// Icons configures the symbols shown for each terminal [RowState] in a
// [Status] or [Spinner] display. Product chrome chips use the separate
// [Icon] type ([IconSuccess], [NewIcon], …); [WithSpinnerIcons] does not
// change [Context.Badge] output.
//
// All fields fall back to Unicode symbols when left empty; only set the
// fields you want to override.
//
// Example:
//
//	nabat.WithSpinnerIcons(nabat.Icons{Success: "+", Error: "x"})
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
