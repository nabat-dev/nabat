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
	"sync"

	"charm.land/bubbles/v2/progress"

	"nabat.dev/theme"
)

// Fill presets for use with [WithProgressBarFillCharacters].
// These mirror [charm.land/bubbles/v2/progress] defaults so callers do not need
// to import the progress package for common glyphs. Half-block is the bubbles
// default fill rune for finer color blending than full block.
//
// Available presets:
//
//   - [ProgressFillHalfBlock] — left half block (▌)
//   - [ProgressFillFullBlock] — full block (█)
//   - [ProgressEmptyLight] — light shade (░)
var (
	ProgressFillHalfBlock = progress.DefaultFullCharHalfBlock
	ProgressFillFullBlock = progress.DefaultFullCharFullBlock
	ProgressEmptyLight    = progress.DefaultEmptyCharBlock
)

// ProgressBar tracks a finite number of steps and writes progress to the
// command's stderr ([Context.IO.ErrOut]).
//
// Per clig.dev guidance, progress indicators belong on stderr so they do
// not corrupt machine-readable data on stdout when the user pipes the
// command's output. When stderr is a terminal, the bar updates in place;
// otherwise each update prints a "[current/total]" line.
//
// Fill colors and percentage styling come from the active [App.Theme] when
// stderr is a terminal and [WithoutProgressBarTheme] is not set: a blend
// between [theme.AccentPrimary] and [theme.TextLink], empty segments use
// [theme.TextMuted], and the numeric percentage uses [theme.TextSecondary].
// To customize colors, use [WithTheme], [WithThemeOverride], or a custom
// [theme.Theme] — not per-bar color arguments.
//
// Nabat renders with [progress.Model.ViewAs] only. Options such as
// [WithProgressBarSpring] tune the embedded bubbles model but do not run
// spring animation here; smooth animation requires Bubble Tea's
// [progress.Model.Update] and frame messages.
//
// ProgressBar is safe for concurrent use after construction. Callers may
// invoke [ProgressBar.Increment], [ProgressBar.Add], [ProgressBar.Set],
// and [ProgressBar.Done] from multiple goroutines.
type ProgressBar struct {
	mu      sync.Mutex
	model   progress.Model
	out     *writer
	total   int
	current int
	tty     bool
}

// ProgressBarOption configures [Context.ProgressBar].
//
// Options return an error so validation failures aggregate into a [ConfigErrors]
// at [Context.ProgressBar] call time, matching the pattern used by [FieldOption].
type ProgressBarOption func(*progressBarConfig) error

type progressBarConfig struct {
	width int

	withoutTheme bool

	fillCharsSet bool
	fillFull     rune
	fillEmpty    rune

	withoutPercentage bool

	scaledBlendSet bool
	scaledBlend    bool

	springSet     bool
	springFreq    float64
	springDamping float64
}

// WithProgressBarWidth sets the character width of the bar. w must be > 0.
//
// Errors:
//   - "nabat: WithProgressBarWidth: w must be > 0" when w <= 0
func WithProgressBarWidth(w int) ProgressBarOption {
	return func(c *progressBarConfig) error {
		if w <= 0 {
			return fmt.Errorf("nabat: WithProgressBarWidth: w must be > 0, got %d", w)
		}
		c.width = w
		return nil
	}
}

// WithoutProgressBarTheme skips colors from [App.Theme] when stderr is a
// terminal. The bar uses the bubbles library default gradient instead.
//
// Non-terminal stderr already renders "[current/total]" lines without themed
// ANSI styling.
func WithoutProgressBarTheme() ProgressBarOption {
	return func(c *progressBarConfig) error {
		c.withoutTheme = true
		return nil
	}
}

// WithProgressBarFillCharacters sets the runes for filled and empty segments.
//
// Example:
//
//	WithProgressBarFillCharacters(nabat.ProgressFillFullBlock, nabat.ProgressEmptyLight)
func WithProgressBarFillCharacters(full, empty rune) ProgressBarOption {
	return func(c *progressBarConfig) error {
		c.fillCharsSet = true
		c.fillFull = full
		c.fillEmpty = empty
		return nil
	}
}

// WithoutProgressBarPercentage hides the trailing numeric percentage on the bar.
//
// Example:
//
//	WithoutProgressBarPercentage()
func WithoutProgressBarPercentage() ProgressBarOption {
	return func(c *progressBarConfig) error {
		c.withoutPercentage = true
		return nil
	}
}

// WithProgressBarScaledBlend sets whether a multi-color blend scales to the
// filled width only (enabled) or spans the full bar width until 100% (disabled,
// the bubbles default).
//
// Example:
//
//	WithProgressBarScaledBlend(true)
func WithProgressBarScaledBlend(enabled bool) ProgressBarOption {
	return func(c *progressBarConfig) error {
		c.scaledBlendSet = true
		c.scaledBlend = enabled
		return nil
	}
}

// WithProgressBarSpring sets frequency and damping for the bubbles progress
// spring (see harmonica). This configures the embedded model only; Nabat does
// not run Bubble Tea frame ticks, so the bar does not animate springs —
// rendering uses [progress.Model.ViewAs] with your step percentage.
//
// Errors:
//   - "nabat: WithProgressBarSpring: frequency must be > 0" when frequency <= 0
//   - "nabat: WithProgressBarSpring: damping must be > 0" when damping <= 0
//
// Example:
//
//	WithProgressBarSpring(25.0, 1.0)
func WithProgressBarSpring(frequency, damping float64) ProgressBarOption {
	return func(c *progressBarConfig) error {
		if frequency <= 0 {
			return fmt.Errorf("nabat: WithProgressBarSpring: frequency must be > 0, got %v", frequency)
		}
		if damping <= 0 {
			return fmt.Errorf("nabat: WithProgressBarSpring: damping must be > 0, got %v", damping)
		}
		c.springSet = true
		c.springFreq = frequency
		c.springDamping = damping
		return nil
	}
}

// ProgressBar creates a bar for total logical steps. Call [ProgressBar.Increment],
// [ProgressBar.Add], or [ProgressBar.Set] to advance it, then [ProgressBar.Done]
// when finished.
//
// total must be > 0; option errors and an invalid total are aggregated into a
// [ConfigErrors] returned without constructing the bar.
//
// Errors:
//   - "nabat: ProgressBar: total must be > 0" when total <= 0
//   - errors returned by any [ProgressBarOption]
//
// Example:
//
//	bar, err := c.ProgressBar(len(files))
//	if err != nil {
//	    return err
//	}
//	for _, f := range files {
//	    process(f)
//	    bar.Increment()
//	}
//	bar.Done()
func (c *Context) ProgressBar(total int, opts ...ProgressBarOption) (*ProgressBar, error) {
	cfg := &progressBarConfig{width: 40}
	var errs ConfigErrors
	if total <= 0 {
		errs.AddErr(fmt.Errorf("nabat: ProgressBar: total must be > 0, got %d", total))
	}
	for i, opt := range opts {
		if opt == nil {
			errs.AddErr(fmtErrInvalidOption("progress bar option", i))
			continue
		}
		if err := opt(cfg); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return nil, &errs
	}

	model := buildProgressModel(c, cfg)

	bar := &ProgressBar{
		model: model,
		out:   &writer{w: c.io.ErrOut},
		total: total,
		tty:   c.io.IsStderrTTY(),
	}
	bar.render()
	return bar, nil
}

func buildProgressModel(c *Context, cfg *progressBarConfig) progress.Model {
	useTheme := !cfg.withoutTheme && c.io.IsStderrTTY()

	opts := make([]progress.Option, 0, 8)
	opts = append(opts, progress.WithWidth(cfg.width))

	if useTheme {
		rt := c.app.Theme()
		opts = append(opts, progress.WithColors(
			rt.Style(theme.AccentPrimary).GetForeground(),
			rt.Style(theme.TextLink).GetForeground(),
		))
	} else {
		opts = append(opts, progress.WithDefaultBlend())
	}

	if cfg.fillCharsSet {
		opts = append(opts, progress.WithFillCharacters(cfg.fillFull, cfg.fillEmpty))
	}
	if cfg.withoutPercentage {
		opts = append(opts, progress.WithoutPercentage())
	}
	if cfg.scaledBlendSet {
		opts = append(opts, progress.WithScaled(cfg.scaledBlend))
	}
	if cfg.springSet {
		opts = append(opts, progress.WithSpringOptions(cfg.springFreq, cfg.springDamping))
	}

	model := progress.New(opts...)
	if useTheme {
		rt := c.app.Theme()
		model.EmptyColor = rt.Style(theme.TextMuted).GetForeground()
		model.PercentageStyle = rt.Style(theme.TextSecondary)
	}
	return model
}

// Increment advances the bar by one step, equivalent to Add(1).
func (p *ProgressBar) Increment() {
	p.Add(1)
}

// Add advances the bar by n steps. The position stays clamped between 0 and the
// configured total.
func (p *ProgressBar) Add(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current += n
	if p.current < 0 {
		p.current = 0
	}
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// Set sets the bar to n completed steps. n is clamped to [0, total].
func (p *ProgressBar) Set(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < 0 {
		n = 0
	}
	p.current = min(n, p.total)
	p.render()
}

// Done marks the bar as complete and moves to the next line. It uses the same
// mutex as [ProgressBar.Increment], [ProgressBar.Add], and [ProgressBar.Set];
// call it once when work is finished.
func (p *ProgressBar) Done() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = p.total
	if p.tty {
		p.out.printf("\r%s\n", p.model.ViewAs(1.0))
	} else {
		p.out.printf("[%d/%d]\n", p.current, p.total)
	}
}

// render writes one frame of the bar. The caller must hold p.mu.
//
// [Context.ProgressBar] guarantees p.total > 0, so no zero-divisor guard is
// needed.
func (p *ProgressBar) render() {
	pct := float64(p.current) / float64(p.total)
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	if p.tty {
		p.out.printf("\r%s", p.model.ViewAs(pct))
	} else {
		p.out.printf("[%d/%d]\n", p.current, p.total)
	}
}
