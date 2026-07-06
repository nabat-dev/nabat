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
