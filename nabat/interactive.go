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

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// runAdhocPromptConfig applies [FieldOption] values to a [promptConfig],
// handles the non-interactive fallback path, and returns the promptConfig
// ready for widget construction.
func runAdhocPromptConfig[T any](c *Context, label, title string, opts []FieldOption[T]) (promptConfig, error) {
	var pc promptConfig
	if err := applyFieldOptions(label, opts, &pc); err != nil {
		return promptConfig{}, err
	}
	pc.contextDir = c.Dir()
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

// applyHuhThemeForAdHocPrompt applies the resolved theme to form with all
// border sides stripped. Single-field ad-hoc prompts have no grouping to
// communicate, so the left border drawn by huh.ThemeBase is visual noise.
// [Form] and [Context.UnsafeForm] keep [applyHuhTheme]; their per-field
// borders remain meaningful.
func (a *App) applyHuhThemeForAdHocPrompt(form *huh.Form) {
	base := a.cfg.resolvedTheme.Huh()
	if base == nil {
		// Fall back explicitly so the border strip still applies rather than
		// letting huh silently reintroduce ThemeCharm's BorderLeft.
		base = huh.ThemeFunc(huh.ThemeCharm)
	}
	noBorder := lipgloss.Border{}
	form.WithTheme(huh.ThemeFunc(func(isDark bool) *huh.Styles {
		s := *base.Theme(isDark) // shallow copy; prevents mutation of cached *Styles
		s.Focused.Base = s.Focused.Base.Border(noBorder, false, false, false, false)
		s.Focused.Card = s.Focused.Card.Border(noBorder, false, false, false, false)
		s.Blurred.Base = s.Blurred.Base.Border(noBorder, false, false, false, false)
		s.Blurred.Card = s.Blurred.Card.Border(noBorder, false, false, false, false)
		return &s
	}))
}

// Input asks for one string value when [Context.IsInteractive] is true.
// Non-interactive use without [WithDefault] fails. Bad options yield
// [*ConfigErrors]; the prompt layer may also fail.
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
	applyInitial(&out, pc)
	f := buildInputField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return "", runErr
	}
	return out, nil
}

// Confirm asks a yes/no question when interactive. Use [WithYes] for moderate
// `--yes` bypass. Use [WithConfirmValue] and [WithConfirmInput] for
// type-to-confirm; [WithYes] and [WithDefault] do not bypass that path.
//
// Example:
//
//	ok, err := c.Confirm("Delete?", WithYes(flags.Yes), WithBypassHint("--yes"))
//
// It may return [*ConfirmationError], [*ConfigErrors], or a prompt error.
func (c *Context) Confirm(prompt string, opts ...FieldOption[bool]) (bool, error) {
	// Apply options without the early non-interactive fallback path so
	// WithYes / WithConfirmValue can run first.
	var pc promptConfig
	if err := applyFieldOptions("confirm", opts, &pc); err != nil {
		return false, err
	}
	pc.text = prompt

	// Severe path first: type-to-confirm is never skipped by WithYes.
	if pc.confirmValue != "" {
		if pc.confirmInput == pc.confirmValue {
			return true, nil
		}
		if !c.interactive {
			return false, confirmationError(prompt, pc.bypassHint)
		}
		typePrompt := fmt.Sprintf("Type %q to confirm", pc.confirmValue)
		typed, err := c.Input(typePrompt, WithValidate(func(s string) error {
			if s != pc.confirmValue {
				return fmt.Errorf("expected %q", pc.confirmValue)
			}
			return nil
		}))
		if err != nil {
			return false, err
		}
		return typed == pc.confirmValue, nil
	}

	if pc.bypassYes {
		return true, nil
	}

	if !c.interactive {
		if pc.hasFallback {
			v, ok := pc.fallback.(bool)
			if !ok {
				return false, fmt.Errorf("nabat: confirm: expected bool default, got %T", pc.fallback)
			}
			return v, nil
		}
		return false, confirmationError(prompt, pc.bypassHint)
	}

	var out bool
	applyInitial(&out, pc)
	f := buildConfirmField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return false, runErr
	}
	return out, nil
}

// TextInput collects multi-line text when interactive. Equivalent to
// [Context.Input] with [WithMultiline]. Non-interactive use requires
// [WithDefault]; otherwise returns an error.
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
	applyInitial(&out, pc)
	f := buildTextField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return "", runErr
	}
	return out, nil
}

// FilePicker collects a file path when interactive. Equivalent to
// [Context.Input] with [WithFilePicker]. Non-interactive use requires
// [WithDefault]; otherwise returns an error.
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
	applyInitial(&out, pc)
	f := buildFileField(&out, pc)
	if runErr := c.app.runPromptField(f); runErr != nil {
		return "", runErr
	}
	return out, nil
}

// Select asks for one choice from choices when interactive. E is inferred
// from the choices slice, enabling typed enum selects. When not interactive,
// Select returns defaultVal without prompting. Bad options yield
// [*ConfigErrors]; the prompt layer may also fail.
func (c *Context) Select[E comparable](prompt string, choices []E, defaultVal E, opts ...SelectOption) (E, error) {
	var zero E
	var pc promptConfig
	if err := applySelectOptions("select", opts, &pc); err != nil {
		return zero, err
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
		return zero, err
	}
	return out, nil
}

// MultiSelect asks for multiple choices when interactive. E is inferred from
// the choices slice. When not interactive, MultiSelect returns defaultVal
// without prompting. Bad options yield [*ConfigErrors]; the prompt layer may
// also fail.
func (c *Context) MultiSelect[E comparable](prompt string, choices, defaultVal []E, opts ...MultiSelectOption) ([]E, error) {
	var pc promptConfig
	if err := applySelectOptions("multi-select", asSelectOptions(opts), &pc); err != nil {
		return nil, err
	}
	if !c.interactive {
		return append([]E(nil), defaultVal...), nil
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
		return nil, err
	}
	return out, nil
}
