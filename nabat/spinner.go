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

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

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

// Spinner is the live handle passed to the [Context.Spinner] callback.
// Call [Spinner.SetText] to replace the spinner title during execution.
// See also [ProgressBar] for step-based progress reporting.
type Spinner struct {
	mu   sync.Mutex
	text string
}

// SetText replaces the entire spinner title. It is safe to call from any
// goroutine; the last writer wins. A call after the work function returns
// is a harmless no-op.
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

// spinnerDoneMsg carries the result of the worker function back to the
// Bubble Tea event loop.
type spinnerDoneMsg struct{ err error }

// spinnerModel is the owned Bubble Tea model. It wraps a bubbles spinner,
// reads the current title from the handle each View(), and dispatches the
// worker in Init so the tea event loop drives animation concurrently.
type spinnerModel struct {
	spin       spinner.Model
	handle     *Spinner
	titleStyle lipgloss.Style
	action     func(*Spinner) error
	err        error
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		func() tea.Msg { return spinnerDoneMsg{err: m.action(m.handle)} },
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
		m.err = msg.err
		return m, tea.Quit
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Interrupt
		}
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() tea.View {
	out := m.spin.View()
	if t := m.handle.title(); t != "" {
		out += m.titleStyle.Render(t)
	}
	return tea.NewView(out)
}

// Spinner runs fn while showing a spinner on stderr in interactive terminals.
// The callback receives a [*Spinner] handle; call [Spinner.SetText] to update
// the title live. Callers that do not need updates ignore the handle:
//
//	c.Spinner("Loading", func(_ *nabat.Spinner) error { return load() })
//
// When stderr is not a terminal, the title is printed once as a plain line and
// fn runs without animation; [Spinner.SetText] calls are no-ops in that path.
// On finish the spinner line is cleared automatically.
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

	handle := &Spinner{text: title}

	// Non-TTY: skip animation; print the title once as a log breadcrumb.
	if !c.io.IsStderrTTY() {
		if title != "" {
			if _, err := fmt.Fprintln(c.io.ErrOut, title); err != nil {
				return err
			}
		}
		return fn(handle)
	}

	info := c.app.Theme().Style(theme.StatusInfo)
	model := spinnerModel{
		spin: spinner.New(
			spinner.WithSpinner(spinner.Spinner(cfg.spinnerType)),
			spinner.WithStyle(info),
		),
		handle:     handle,
		titleStyle: info,
		action:     fn,
	}

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
