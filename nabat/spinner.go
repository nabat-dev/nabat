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
	"fmt"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"

	"nabat.dev/theme"
)

// defaultSpinnerDelay is how long work must run before the animated spinner
// appears. Faster completions print a static success/error line instead, so
// short idempotent paths never open a TUI or leak terminal probe replies.
const defaultSpinnerDelay = 200 * time.Millisecond

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

// spinnerOnlyOption wraps a spinner-specific function as a [SpinnerOption].
type spinnerOnlyOption func(*spinnerConfig) error

func (f spinnerOnlyOption) applySpinner(c *spinnerConfig) error { return f(c) }

type spinnerConfig struct {
	title       string
	spinnerType SpinnerType
	icons       Icons
	delay       time.Duration
	delaySet    bool
}

// WithSpinnerType sets the spinner animation preset. This option can be passed
// to both [Context.Spinner] and [Context.Status].
//
// Example:
//
//	c.Spinner(fn, WithTitle("Deploying"), WithSpinnerType(SpinnerDots()))
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
//	c.Spinner(fn, WithTitle("Deploying"), WithSpinnerIcons(Icons{
//	    Success: "+",
//	    Error:   "x",
//	}))
func WithSpinnerIcons(icons Icons) spinnerStatusOption {
	return sharedOption{
		spinnerFn: func(c *spinnerConfig) error { c.icons = icons; return nil },
		statusFn:  func(c *statusConfig) error { c.icons = icons; return nil },
	}
}

// WithSpinnerDelay sets how long work must run before the animated spinner
// appears. The default is 200ms. Pass 0 to start animation immediately.
// Completing before the delay prints a static success/error line on stderr
// and never starts the animation.
//
// Example:
//
//	c.Spinner(fn, WithTitle("Deploying"), WithSpinnerDelay(0))
func WithSpinnerDelay(d time.Duration) SpinnerOption {
	return spinnerOnlyOption(func(c *spinnerConfig) error {
		if d < 0 {
			return fmt.Errorf("nabat: WithSpinnerDelay: delay must be >= 0")
		}
		c.delay = d
		c.delaySet = true
		return nil
	})
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

func (s *Spinner) completionIcon(fnErr error) string {
	state := s.completionState(fnErr)
	glyph := staticIcon(state, s.icons)
	switch state {
	case RowSuccess:
		return s.th.Style(theme.StatusSuccess).Render(glyph)
	case RowError:
		return s.th.Style(theme.StatusError).Render(glyph)
	case RowWarning:
		return s.th.Style(theme.StatusWarning).Render(glyph)
	default:
		return s.th.Style(theme.TextMuted).Render(glyph)
	}
}

// Spinner runs fn while showing an animated single-line spinner on stderr in
// interactive terminals. The callback receives a [*Spinner] handle; call
// [Spinner.SetText] to update the header title while the work runs.
//
// Animation starts only after a short delay (default 200ms; see
// [WithSpinnerDelay]). If fn returns before the delay, Spinner prints a static
// success or error line on stderr and never starts the animation. This avoids
// terminal capability-probe leaks from short-lived TUI programs.
//
// On a TTY, fn runs in a separate goroutine while the calling goroutine drives
// the animation loop. Spinner returns only after fn returns. Context
// cancellation is observed by the animation loop, but does not preempt fn;
// the loop waits for fn to finish, then returns [context.Canceled] when fn
// itself returned nil.
//
// On completion the final state persists as static text in the terminal
// scrollback. The header icon is "✓" on success and "✗" on error.
//
// When stderr is not a terminal, the title is printed once as a plain line and
// fn runs without animation on the calling goroutine.
//
// For multi-row live status displays use [Context.Status] instead.
//
// Errors:
//   - any error returned by fn (preferred over write or cancel errors)
//   - errors from writing the spinner line to stderr
//   - [*ConfigErrors] from option validation
//   - [context.Canceled] when the context is canceled and fn returned nil
//
// Pass [WithTitle] for the initial header text. The signature matches
// [Context.Status]: callback first, then options.
func (c *Context) Spinner(fn func(*Spinner) error, opts ...SpinnerOption) error {
	// Callback-first avoids a go1.27 printf-vet false positive when Context
	// also has generic methods (title string was treated as a format).

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

	delay := defaultSpinnerDelay
	if cfg.delaySet {
		delay = cfg.delay
	}

	handle := &Spinner{
		text:  cfg.title,
		icons: cfg.icons,
		th:    c.app.Theme(),
		start: time.Now(),
	}

	// Non-TTY path: print title and run fn with no animation.
	if !c.io.IsStderrTTY() {
		if cfg.title != "" {
			if _, err := fmt.Fprintln(c.io.ErrOut, cfg.title); err != nil {
				return err
			}
		}
		return fn(handle)
	}

	return c.runTTYSpinner(handle, cfg, delay, fn)
}

// runTTYSpinner animates a single line on stderr using ANSI rewrite, without
// Bubble Tea. Work runs in a goroutine; animation starts after delay.
func (c *Context) runTTYSpinner(
	handle *Spinner,
	cfg *spinnerConfig,
	delay time.Duration,
	fn func(*Spinner) error,
) error {
	done := make(chan error, 1)
	go func() {
		done <- fn(handle)
	}()

	w := c.io.RawErrOut()
	sp := spinner.Spinner(cfg.spinnerType)
	frames := sp.Frames
	if len(frames) == 0 {
		frames = spinner.Dot.Frames
	}
	fps := sp.FPS
	if fps <= 0 {
		fps = 100 * time.Millisecond
	}

	titleStyle := handle.th.Style(theme.StatusInfo)
	activeStyle := handle.th.Style(theme.SpinnerActive)

	var (
		shown    bool
		frameIdx int
	)

	writeLive := func() error {
		prefix := activeStyle.Render(frames[frameIdx%len(frames)])
		title := handle.title()
		var err error
		if title != "" {
			_, err = fmt.Fprintf(w, "\r\033[K%s %s", prefix, titleStyle.Render(title))
		} else {
			_, err = fmt.Fprintf(w, "\r\033[K%s", prefix)
		}
		return err
	}

	writeDone := func(fnErr error, animated bool) error {
		icon := handle.completionIcon(fnErr)
		title := handle.title()
		var err error
		if animated {
			if title != "" {
				_, err = fmt.Fprintf(w, "\r\033[K%s %s\n", icon, titleStyle.Render(title))
			} else {
				_, err = fmt.Fprintf(w, "\r\033[K%s\n", icon)
			}
			return err
		}
		// Fast path: static acknowledgment line, never animated.
		if title != "" {
			_, err = fmt.Fprintf(w, "%s %s\n", icon, titleStyle.Render(title))
		} else {
			_, err = fmt.Fprintf(w, "%s\n", icon)
		}
		return err
	}

	// finish prefers fnErr, then a prior live-frame write error, then the
	// final-line write error, then context cancellation.
	finish := func(fnErr error, animated bool, cancel bool, liveErr error) error {
		wErr := writeDone(fnErr, animated)
		if fnErr != nil {
			return fnErr
		}
		if liveErr != nil {
			return liveErr
		}
		if wErr != nil {
			return wErr
		}
		if cerr := c.Err(); cerr != nil {
			return cerr
		}
		if cancel {
			return context.Canceled
		}
		return nil
	}

	var delayC <-chan time.Time
	var delayTimer *time.Timer
	if delay <= 0 {
		// Start animation on the first loop iteration.
		shown = true
		if err := writeLive(); err != nil {
			return finish(<-done, false, false, err)
		}
	} else {
		delayTimer = time.NewTimer(delay)
		delayC = delayTimer.C
	}
	if delayTimer != nil {
		defer delayTimer.Stop()
	}

	ticker := time.NewTicker(fps)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			return finish(err, shown, false, nil)
		case <-c.Done():
			return finish(<-done, shown, true, nil)
		case <-delayC:
			delayC = nil
			if !shown {
				shown = true
				if err := writeLive(); err != nil {
					return finish(<-done, false, false, err)
				}
			}
		case <-ticker.C:
			if !shown {
				continue
			}
			frameIdx++
			if err := writeLive(); err != nil {
				return finish(<-done, shown, false, err)
			}
		}
	}
}
