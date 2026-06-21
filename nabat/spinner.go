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
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"

	"nabat.dev/theme"

	tea "charm.land/bubbletea/v2"
)

// SpinnerType selects spinner frames for [WithSpinnerType] and [Context.Spinner].
type SpinnerType spinner.Spinner

// Spinner presets for use with [WithSpinnerType] and [Context.Spinner].
// These are aliases for charm.land/bubbles/v2/spinner types, so callers do
// not need to import the spinner package directly.
//
// Available presets:
//
//   - [SpinnerLine]      — rotating line (| / - \)
//   - [SpinnerDots]      — Braille-dot animation (default)
//   - [SpinnerMiniDot]   — single Braille dot
//   - [SpinnerJump]      — jumping dot animation
//   - [SpinnerPoints]    — three pulsing points
//   - [SpinnerPulse]     — pulsing block
//   - [SpinnerGlobe]     — rotating globe emoji
//   - [SpinnerMoon]      — moon-phase animation
//   - [SpinnerMonkey]    — see/hear/speak-no-evil monkey emoji
//   - [SpinnerMeter]     — filling meter animation
//   - [SpinnerHamburger] — three-line hamburger animation
//   - [SpinnerEllipsis]  — animated ellipsis (., .., ...)

// SpinnerLine returns the rotating-line spinner preset (| / - \).
func SpinnerLine() SpinnerType { return SpinnerType(spinner.Line) }

// SpinnerDots returns the Braille-dot spinner preset (default for
// [Context.Spinner]).
func SpinnerDots() SpinnerType { return SpinnerType(spinner.Dot) }

// SpinnerMiniDot returns the single-Braille-dot spinner preset.
func SpinnerMiniDot() SpinnerType { return SpinnerType(spinner.MiniDot) }

// SpinnerJump returns the jumping-dot spinner preset.
func SpinnerJump() SpinnerType { return SpinnerType(spinner.Jump) }

// SpinnerPoints returns the three-pulsing-points spinner preset.
func SpinnerPoints() SpinnerType { return SpinnerType(spinner.Points) }

// SpinnerPulse returns the pulsing-block spinner preset.
func SpinnerPulse() SpinnerType { return SpinnerType(spinner.Pulse) }

// SpinnerGlobe returns the rotating-globe-emoji spinner preset.
func SpinnerGlobe() SpinnerType { return SpinnerType(spinner.Globe) }

// SpinnerMoon returns the moon-phase-animation spinner preset.
func SpinnerMoon() SpinnerType { return SpinnerType(spinner.Moon) }

// SpinnerMonkey returns the see/hear/speak-no-evil-monkey-emoji spinner preset.
func SpinnerMonkey() SpinnerType { return SpinnerType(spinner.Monkey) }

// SpinnerMeter returns the filling-meter-animation spinner preset.
func SpinnerMeter() SpinnerType { return SpinnerType(spinner.Meter) }

// SpinnerHamburger returns the three-line-hamburger-animation spinner preset.
func SpinnerHamburger() SpinnerType { return SpinnerType(spinner.Hamburger) }

// SpinnerEllipsis returns the animated-ellipsis spinner preset (., .., ...).
func SpinnerEllipsis() SpinnerType { return SpinnerType(spinner.Ellipsis) }

type spinnerConfig struct {
	spinnerType SpinnerType
	icons       Icons
}

// SpinnerOption configures [Context.Spinner].
type SpinnerOption func(*spinnerConfig) error

// WithSpinnerType sets the spinner animation preset.
//
// Example:
//
//	c.Spinner("Deploying", func(sp *nabat.Spinner) error {
//		return deploy()
//	}, WithSpinnerType(SpinnerDots()))
func WithSpinnerType(t SpinnerType) SpinnerOption {
	return func(c *spinnerConfig) error {
		c.spinnerType = t
		return nil
	}
}

// WithSpinnerIcons overrides the default row state icons for a [Context.Spinner]
// call. Only the non-empty fields in icons are used; empty fields keep their
// built-in defaults ("✓", "✗", "!", "•").
//
// Example:
//
//	c.Spinner("Deploying", fn, nabat.WithSpinnerIcons(nabat.Icons{
//	    Success: "+",
//	    Error:   "x",
//	}))
func WithSpinnerIcons(icons Icons) SpinnerOption {
	return func(c *spinnerConfig) error {
		c.icons = icons
		return nil
	}
}

// Spinner is the live handle passed to the [Context.Spinner] callback. Call
// [Spinner.SetText] to update the header title. Call [Spinner.Row] to add or
// update a keyed status row beneath the header.
//
// See also [ProgressBar] for step-based progress reporting.
type Spinner struct {
	mu     sync.Mutex
	text   string
	rows   []*SpinnerRow          // insertion-ordered
	rowIdx map[string]*SpinnerRow // key -> row
	icons  Icons
	th     theme.ResolvedTheme
	start  time.Time
}

// SetText replaces the spinner header title. It is safe to call from any
// goroutine; the last writer wins. A call after the work function returns is a
// harmless no-op.
func (s *Spinner) SetText(text string) {
	s.mu.Lock()
	s.text = text
	s.mu.Unlock()
}

func (s *Spinner) title() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.text
}

// Row returns the keyed row for key, creating it on first call and returning
// the same row on every subsequent call. Rows are displayed beneath the header
// in creation order. Row is safe to call from any goroutine.
//
// Example:
//
//	row := sp.Row("pod/api-abc")
//	row.Set("Scheduled", "assigned to node-3")
//	// ... later ...
//	row.Success()
func (s *Spinner) Row(key string) *SpinnerRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rowIdx[key]; ok {
		return r
	}
	if s.rowIdx == nil {
		s.rowIdx = make(map[string]*SpinnerRow)
	}
	r := &SpinnerRow{
		key:     key,
		created: time.Now(),
	}
	s.rows = append(s.rows, r)
	s.rowIdx[key] = r
	return r
}

// rowSnapshots returns point-in-time copies of all rows in insertion order.
func (s *Spinner) rowSnapshots() []rowSnapshot {
	s.mu.Lock()
	rows := append([]*SpinnerRow(nil), s.rows...)
	s.mu.Unlock()
	snaps := make([]rowSnapshot, 0, len(rows))
	for _, r := range rows {
		snaps = append(snaps, r.snapshot())
	}
	return snaps
}

// completionState determines the header icon after the work function returns.
// It returns [RowError] when fnErr is non-nil or any row is in [RowError]
// state, [RowWarning] when any row is in [RowWarning] state, and [RowSuccess]
// otherwise.
func (s *Spinner) completionState(fnErr error) RowState {
	if fnErr != nil {
		return RowError
	}
	s.mu.Lock()
	rows := append([]*SpinnerRow(nil), s.rows...)
	s.mu.Unlock()
	hasWarn := false
	for _, r := range rows {
		r.mu.Lock()
		st := r.state
		r.mu.Unlock()
		if st == RowError {
			return RowError
		}
		if st == RowWarning {
			hasWarn = true
		}
	}
	if hasWarn {
		return RowWarning
	}
	return RowSuccess
}

// renderPlainTable returns an aligned plain-text table of all rows with no
// ANSI styling. It is used by the non-TTY fallback path.
func (s *Spinner) renderPlainTable() string {
	snaps := s.rowSnapshots()
	if len(snaps) == 0 {
		return ""
	}
	widths := computeColumnWidths(snaps)
	var sb strings.Builder
	for _, snap := range snaps {
		icon := staticIcon(snap.state, s.icons)
		sb.WriteString(" ")
		sb.WriteString(icon)
		sb.WriteString("  ")
		cols := rowColumns(snap)
		for i, col := range cols {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			sb.WriteString(padRight(col, w))
			if i < len(cols)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Spinner runs fn while showing an animated spinner on stderr in interactive
// terminals. The callback receives a [*Spinner] handle; call [Spinner.SetText]
// to update the header title and [Spinner.Row] to add or update keyed status
// rows beneath it.
//
// When no rows are added, the display is a single animated line (the original
// behavior). When rows are added each row shows its own spinner animation and
// an auto-incrementing elapsed timer; calling [SpinnerRow.Success],
// [SpinnerRow.Error], [SpinnerRow.Warn], or [SpinnerRow.Done] freezes the
// timer and replaces the row's spinner with a static icon.
//
// On completion the final state persists as static text in the terminal
// scrollback. The header icon is derived automatically from the work result
// and from any row error states.
//
// When stderr is not a terminal, the title is printed once as a plain line and
// fn runs without animation. If rows were added, their final state is printed
// as a plain-text aligned table after fn returns.
//
// Errors:
//   - any error returned by fn
//   - [*ConfigErrors] from option validation
//   - [context.Canceled] when the user interrupts with ctrl+c or the context
//     is canceled
func (c *Context) Spinner(title string, fn func(*Spinner) error, opts ...SpinnerOption) error {
	cfg := &spinnerConfig{spinnerType: SpinnerDots()}
	var errs ConfigErrors
	for i, opt := range opts {
		if opt == nil {
			errs.AddErr(fmtErrInvalidOption("spinner option", i))
			continue
		}
		if err := opt(cfg); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return &errs
	}

	handle := &Spinner{
		text:  title,
		icons: cfg.icons,
		th:    c.app.Theme(),
		start: time.Now(),
	}

	// Non-TTY: print title once then run fn; append final row table if any.
	if !c.io.IsStderrTTY() {
		if title != "" {
			if _, err := fmt.Fprintln(c.io.ErrOut, title); err != nil {
				return err
			}
		}
		fnErr := fn(handle)
		if table := handle.renderPlainTable(); table != "" {
			if _, wErr := fmt.Fprint(c.io.ErrOut, table); wErr != nil && fnErr == nil {
				return wErr
			}
		}
		return fnErr
	}

	rt := c.app.Theme()
	info := rt.Style(theme.StatusInfo)
	activeStyle := rt.Style(theme.SpinnerActive)
	model := newSpinnerModel(
		spinner.New(
			spinner.WithSpinner(spinner.Spinner(cfg.spinnerType)),
			spinner.WithStyle(info),
		),
		handle,
		info,
		activeStyle,
		rt,
		fn,
	)

	final, runErr := tea.NewProgram(
		model,
		tea.WithContext(c),
		tea.WithOutput(c.io.RawErrOut()),
		tea.WithInput(c.io.RawIn()),
	).Run()

	if m, ok := final.(spinnerModel); ok && m.err != nil {
		return m.err
	}
	switch {
	case errors.Is(runErr, tea.ErrInterrupted):
		return context.Canceled
	case errors.Is(runErr, tea.ErrProgramKilled):
		if cerr := c.Err(); cerr != nil {
			return cerr
		}
		return context.Canceled
	}
	return runErr
}
