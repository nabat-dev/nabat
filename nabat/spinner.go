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
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"

	"nabat.dev/theme"

	tea "charm.land/bubbletea/v2"
)

// SpinnerType selects spinner frames for [WithSpinnerType], [Context.Spinner],
// and [Context.Status].
type SpinnerType spinner.Spinner

// Spinner presets for use with [WithSpinnerType].
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
// [Context.Spinner] and [Context.Status]).
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

// SpinnerOption configures [Context.Spinner].
type SpinnerOption interface {
	applySpinner(*spinnerConfig) error
}

// StatusOption configures [Context.Status].
type StatusOption interface {
	applyStatus(*statusConfig) error
}

// spinnerStatusOption satisfies both [SpinnerOption] and [StatusOption].
// Returned by [WithSpinnerType] and [WithSpinnerIcons] so those options can
// be passed to either [Context.Spinner] or [Context.Status].
type spinnerStatusOption interface {
	SpinnerOption
	StatusOption
}

// sharedOption is the concrete type behind [spinnerStatusOption].
type sharedOption struct {
	spinnerFn func(*spinnerConfig) error
	statusFn  func(*statusConfig) error
}

func (o sharedOption) applySpinner(c *spinnerConfig) error { return o.spinnerFn(c) }
func (o sharedOption) applyStatus(c *statusConfig) error   { return o.statusFn(c) }

// statusOnlyOption wraps a status-specific function as a [StatusOption].
type statusOnlyOption func(*statusConfig) error

func (f statusOnlyOption) applyStatus(c *statusConfig) error { return f(c) }

type spinnerConfig struct {
	spinnerType SpinnerType
	icons       Icons
}

// WithSpinnerType sets the spinner animation preset. This option can be passed
// to both [Context.Spinner] and [Context.Status].
//
// Example:
//
//	c.Spinner("Deploying", fn, WithSpinnerType(SpinnerDots()))
//	c.Status(fn, WithTitle("Deploying"), WithSpinnerType(SpinnerDots()))
func WithSpinnerType(t SpinnerType) spinnerStatusOption {
	return sharedOption{
		spinnerFn: func(c *spinnerConfig) error { c.spinnerType = t; return nil },
		statusFn:  func(c *statusConfig) error { c.spinnerType = t; return nil },
	}
}

// WithSpinnerIcons overrides the default row state icons. Only the non-empty
// fields in icons are used; empty fields keep their built-in defaults
// ("✓", "✗", "!", "•"). This option can be passed to both [Context.Spinner]
// and [Context.Status].
//
// Example:
//
//	c.Spinner("Deploying", fn, WithSpinnerIcons(Icons{
//	    Success: "+",
//	    Error:   "x",
//	}))
func WithSpinnerIcons(icons Icons) spinnerStatusOption {
	return sharedOption{
		spinnerFn: func(c *spinnerConfig) error { c.icons = icons; return nil },
		statusFn:  func(c *statusConfig) error { c.icons = icons; return nil },
	}
}

// Spinner is the live handle passed to the [Context.Spinner] callback. Call
// [Spinner.SetText] to update the animated header title.
//
// For multi-row live displays use [Context.Status] instead.
type Spinner struct {
	mu    sync.Mutex
	text  string
	icons Icons
	th    theme.ResolvedTheme
	start time.Time
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

// completionState determines the header icon after the work function returns.
// Returns [RowError] when fnErr is non-nil, [RowSuccess] otherwise.
func (s *Spinner) completionState(fnErr error) RowState {
	if fnErr != nil {
		return RowError
	}
	return RowSuccess
}

// Spinner runs fn while showing an animated single-line spinner on stderr in
// interactive terminals. The callback receives a [*Spinner] handle; call
// [Spinner.SetText] to update the header title while the work runs.
//
// On completion the final state persists as static text in the terminal
// scrollback. The header icon is "✓" on success and "✗" on error.
//
// When stderr is not a terminal, the title is printed once as a plain line and
// fn runs without animation.
//
// For multi-row live status displays use [Context.Status] instead.
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
		if err := opt.applySpinner(cfg); err != nil {
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

	// Non-TTY path: print title and run fn with no animation.
	if !c.io.IsStderrTTY() {
		if title != "" {
			if _, err := fmt.Fprintln(c.io.ErrOut, title); err != nil {
				return err
			}
		}
		return fn(handle)
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
