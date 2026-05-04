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

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"

	"nabat.dev/theme"
)

// runAdhocPromptConfig applies [FieldOption] values to a [promptConfig],
// handles the non-interactive fallback path, and returns the promptConfig
// ready for widget construction.
func runAdhocPromptConfig[T any](c *Context, label, title string, opts []FieldOption[T]) (promptConfig, error) {
	var pc promptConfig
	if err := applyFieldOptions(label, opts, &pc); err != nil {
		return promptConfig{}, err
	}
	if !c.interactive {
		if pc.hasFallback {
			return pc, nil
		}
		return promptConfig{}, fmt.Errorf("nabat: %s requires interactive terminal", label)
	}
	pc.text = title
	return pc, nil
}

func (a *App) applyHuhTheme(form *huh.Form) {
	huhTheme := a.cfg.resolvedTheme.Huh()
	if huhTheme == nil {
		return
	}
	form.WithTheme(huhTheme)
}

// SpinnerType selects spinner frames for [WithSpinnerType] and [Context.Spinner].
type SpinnerType spinner.Type

// Spinner presets for use with [WithSpinnerType] and [Context.Spinner].
// These are aliases for charm.land/huh/v2/spinner types, so callers do not
// need to import the spinner package directly.
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
func SpinnerDots() SpinnerType { return SpinnerType(spinner.Dots) }

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
//	c.Spinner("Deploying...", func() error {
//		return nil
//	}, WithSpinnerType(SpinnerDots()))
func WithSpinnerType(t SpinnerType) SpinnerOption {
	return func(c *spinnerConfig) error {
		c.spinnerType = t
		return nil
	}
}

// Spinner runs fn while showing a spinner on stderr in interactive terminals.
//
// Errors:
//   - any error returned by fn
//   - errors from applying [SpinnerOption] values
//   - errors from the underlying spinner implementation
func (c *Context) Spinner(title string, fn func() error, opts ...SpinnerOption) error {
	cfg := &spinnerConfig{
		spinnerType: SpinnerDots(),
	}
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

	s := spinner.New().
		Title(title).
		Type(spinner.Type(cfg.spinnerType)).
		WithOutput(c.io.RawErrOut()).
		WithInput(c.io.RawIn()).
		WithTheme(spinner.ThemeFunc(func(isDark bool) *spinner.Styles {
			styles := spinner.ThemeDefault(isDark)
			info := c.app.Theme().Style(theme.StatusInfo)
			styles.Spinner = info
			styles.Title = info
			return styles
		})).
		Context(c).
		ActionWithErr(func(_ context.Context) error {
			return fn()
		})
	return s.Run()
}

// Input asks for one string value when [Context.IsInteractive] is true.
//
// Errors:
//   - "nabat: input requires interactive terminal" when not interactive and
//     no [WithDefault] is set
//   - [*ConfigErrors] from option validation
//   - errors from the prompt layer
func (c *Context) Input(prompt string, opts ...FieldOption[string]) (string, error) {
	pc, err := runAdhocPromptConfig(c, "input", prompt, opts)
	if err != nil {
		return "", err
	}
	if !c.interactive {
		v, ok := pc.fallback.(string)
		if !ok {
			return "", fmt.Errorf("nabat: input: expected string default, got %T", pc.fallback)
		}
		return v, nil
	}
	var out string
	f := buildInputField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return "", runErr
	}
	return out, nil
}

// Confirm asks a yes/no question when [Context.IsInteractive] is true.
//
// Errors:
//   - "nabat: confirm requires interactive terminal" when not interactive and
//     no [WithDefault] is set
//   - [*ConfigErrors] from option validation
//   - errors from the prompt layer
func (c *Context) Confirm(prompt string, opts ...FieldOption[bool]) (bool, error) {
	pc, err := runAdhocPromptConfig(c, "confirm", prompt, opts)
	if err != nil {
		return false, err
	}
	if !c.interactive {
		v, ok := pc.fallback.(bool)
		if !ok {
			return false, fmt.Errorf("nabat: confirm: expected bool default, got %T", pc.fallback)
		}
		return v, nil
	}
	var out bool
	f := buildConfirmField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return false, runErr
	}
	return out, nil
}

// TextInput collects multi-line text when interactive. This is equivalent to
// c.Input with [WithMultiline], provided as a convenience method.
//
// Errors:
//   - "nabat: text input requires interactive terminal" when not interactive
//     and no [WithDefault] is set
//   - [*ConfigErrors] from option validation
//   - errors from the prompt layer
func (c *Context) TextInput(prompt string, opts ...FieldOption[string]) (string, error) {
	// Inject WithMultiline as the first option so the caller doesn't have to.
	allOpts := make([]FieldOption[string], 0, len(opts)+1)
	allOpts = append(allOpts, WithMultiline())
	allOpts = append(allOpts, opts...)
	pc, err := runAdhocPromptConfig(c, "text input", prompt, allOpts)
	if err != nil {
		return "", err
	}
	if !c.interactive {
		v, ok := pc.fallback.(string)
		if !ok {
			return "", fmt.Errorf("nabat: text input: expected string default, got %T", pc.fallback)
		}
		return v, nil
	}
	var out string
	f := buildTextField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return "", runErr
	}
	return out, nil
}

// FilePicker collects a file path when interactive. This is equivalent to
// c.Input with [WithFilePicker], provided as a convenience method.
//
// Errors:
//   - "nabat: file picker requires interactive terminal" when not interactive
//     and no [WithDefault] is set
//   - [*ConfigErrors] from option validation
//   - errors from the prompt layer
func (c *Context) FilePicker(prompt string, opts ...FieldOption[string]) (string, error) {
	allOpts := make([]FieldOption[string], 0, len(opts)+1)
	allOpts = append(allOpts, WithFilePicker())
	allOpts = append(allOpts, opts...)
	pc, err := runAdhocPromptConfig(c, "file picker", prompt, allOpts)
	if err != nil {
		return "", err
	}
	if !c.interactive {
		v, ok := pc.fallback.(string)
		if !ok {
			return "", fmt.Errorf("nabat: file picker: expected string default, got %T", pc.fallback)
		}
		return v, nil
	}
	var out string
	f := buildFileField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return "", runErr
	}
	return out, nil
}

// Select asks for one choice from choices when interactive. E is inferred
// from the choices slice, enabling typed enum selects.
//
// Select is a package-level function (not a method) because Go does not
// allow type parameters on methods.
//
// Errors:
//   - "nabat: select requires interactive terminal" when not interactive and
//     no [WithDefault] set via a [FieldOption][E]
//   - [*ConfigErrors] from option validation
//   - errors from the prompt layer
func Select[E comparable](c *Context, prompt string, choices []E, defaultVal E, opts ...SelectOption) (E, error) {
	var pc promptConfig
	if err := applySelectOptions("select", opts, &pc); err != nil {
		return defaultVal, err
	}
	if !c.interactive {
		return defaultVal, nil
	}
	options := make([]huh.Option[E], 0, len(choices))
	for _, ch := range choices {
		options = append(options, huh.NewOption(fmt.Sprint(ch), ch))
	}
	out := defaultVal
	f := huh.NewSelect[E]().Title(prompt).Options(options...).Value(&out)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	f = f.Filtering(pc.filtering)
	if pc.height > 0 {
		f = f.Height(pc.height)
	}
	if pc.validate != nil {
		fn := pc.validate
		f = f.Validate(func(v E) error { return fn(v) })
	}
	if err := c.app.runPromptField(f); err != nil {
		return defaultVal, err
	}
	return out, nil
}

// MultiSelect asks for multiple choices when interactive. E is inferred from
// the choices slice.
//
// MultiSelect is a package-level function (not a method) because Go does not
// allow type parameters on methods.
//
// Errors:
//   - "nabat: multi-select requires interactive terminal" when not interactive
//   - [*ConfigErrors] from option validation
//   - errors from the prompt layer
func MultiSelect[E comparable](c *Context, prompt string, choices, defaultVal []E, opts ...MultiSelectOption) ([]E, error) {
	var pc promptConfig
	if err := applySelectOptions("multi-select", asSelectOptions(opts), &pc); err != nil {
		return defaultVal, err
	}
	if !c.interactive {
		return defaultVal, nil
	}
	options := make([]huh.Option[E], 0, len(choices))
	for _, ch := range choices {
		options = append(options, huh.NewOption(fmt.Sprint(ch), ch))
	}
	out := append([]E(nil), defaultVal...)
	f := huh.NewMultiSelect[E]().Title(prompt).Options(options...).Value(&out)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	f = f.Filterable(pc.filtering)
	if pc.height > 0 {
		f = f.Height(pc.height)
	}
	if pc.limit > 0 {
		f = f.Limit(pc.limit)
	}
	if pc.validate != nil {
		fn := pc.validate
		f = f.Validate(func(v []E) error { return fn(v) })
	}
	if err := c.app.runPromptField(f); err != nil {
		return defaultVal, err
	}
	return out, nil
}
