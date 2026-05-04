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
	"strconv"
	"time"

	"charm.land/huh/v2"
)

// FormFieldValue is the set of non-select value types accepted by
// [WithFormField]. Select types use [WithSelectField] or
// [WithMultiSelectField] instead.
//
// String fields use [WithMultiline] or [WithFilePicker] to switch widget mode.
// The old nabat.Text and nabat.File types have been removed; use plain string
// with the appropriate mode option.
type FormFieldValue interface {
	string | bool | int | int64 | uint | float64 | time.Duration
}

// FormOption configures a form built with [Context.Form]. It is satisfied by
// field constructors ([WithFormField], [WithSelectField],
// [WithMultiSelectField], [WithFormNote]) and chrome helpers ([WithFormTitle],
// [WithFormDescription], [WithFormSubmit], [WithFormCancel],
// [WithFormAccessible], [WithFormKeyMap], [WithFormTimeout]).
type FormOption interface {
	formOption()
	apply(*formConfig) error
}

// GroupOption configures a single page (group) inside a multi-page form.
// Field constructors satisfy both [FormOption] and [GroupOption] so they slot
// into either [WithFormGroup] or directly into [Context.Form].
type GroupOption interface {
	groupOption()
	applyToGroup(*formGroup) error
}

// formOpt adapts a function to [FormOption].
type formOpt func(*formConfig) error

func (f formOpt) formOption()                {}
func (f formOpt) apply(fc *formConfig) error { return f(fc) }

// groupOpt adapts a function to [GroupOption].
type groupOpt func(*formGroup) error

func (f groupOpt) groupOption()                     {}
func (f groupOpt) applyToGroup(fg *formGroup) error { return f(fg) }

// formFieldDual implements both [FormOption] and [GroupOption]. Used by all
// field constructors so they can slot into either site.
type formFieldDual struct {
	formApply  func(*formConfig) error
	groupApply func(*formGroup) error
}

func (f formFieldDual) formOption()                      {}
func (f formFieldDual) apply(fc *formConfig) error       { return f.formApply(fc) }
func (f formFieldDual) groupOption()                     {}
func (f formFieldDual) applyToGroup(fg *formGroup) error { return f.groupApply(fg) }

// formConfig collects form-level settings and field definitions.
type formConfig struct {
	title       string
	description string
	submit      string
	cancel      string
	accessible  bool
	keymap      *huh.KeyMap
	timeout     time.Duration
	groups      []formGroup
}

// formGroup is one page (huh.Group) within the form. Fields added directly
// via [WithFormField] etc. land in the default first group.
type formGroup struct {
	title       string
	description string
	fields      []fieldEntry
}

// fieldEntry is the existential interface that lets formGroup hold typed
// fields without fixing T at the group level.
type fieldEntry interface {
	buildHuh() (huh.Field, error)
	applyFallback() error
	fieldTitle() string
}

// typedFormField is a form field whose target pointer and fallback value keep
// their Go type T end-to-end. applyFallback never performs an any assertion;
// the type check happens once at field construction time in WithFormField /
// WithSelectField / WithMultiSelectField.
type typedFormField[T any] struct {
	title    string
	target   *T
	fallback T
	hasFB    bool
	build    func() (huh.Field, error)
}

func (f typedFormField[T]) buildHuh() (huh.Field, error) { return f.build() }

func (f typedFormField[T]) applyFallback() error {
	if !f.hasFB {
		return fmt.Errorf("nabat: form field %q has no default for non-interactive mode", f.title)
	}
	*f.target = f.fallback
	return nil
}

func (f typedFormField[T]) fieldTitle() string { return f.title }

// noteEntry is a display-only field that applies a no-op fallback so
// formFallbackWalk can treat every entry uniformly.
type noteEntry struct {
	title string
	body  string
}

func (e noteEntry) buildHuh() (huh.Field, error) {
	n := huh.NewNote().Title(e.title)
	if e.body != "" {
		n = n.Description(e.body)
	}
	return n, nil
}

func (noteEntry) applyFallback() error { return nil }
func (e noteEntry) fieldTitle() string { return e.title }

// ensureDefaultGroup creates groups[0] if the group slice is empty.
func (fc *formConfig) ensureDefaultGroup() {
	if len(fc.groups) == 0 {
		fc.groups = append(fc.groups, formGroup{})
	}
}

// WithFormTitle sets the form-level title. It is applied to the first group
// when that group has no [WithGroupTitle] set (chrome precedence rule).
func WithFormTitle(s string) FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.title = s
		return nil
	})
}

// WithFormDescription sets the form-level description. It is applied to the
// first group when that group has no [WithGroupDescription] set.
func WithFormDescription(s string) FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.description = s
		return nil
	})
}

// WithFormSubmit sets the submit button label.
func WithFormSubmit(label string) FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.submit = label
		return nil
	})
}

// WithFormCancel sets the cancel button label.
func WithFormCancel(label string) FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.cancel = label
		return nil
	})
}

// WithFormAccessible enables huh's screen-reader-friendly accessible mode.
// Recommended pattern: drive from an environment variable:
//
//	nabat.WithFormAccessible(), // always on; or:
//	// conditionally: use c.Form opts builder with os.Getenv("ACCESSIBLE") != ""
func WithFormAccessible() FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.accessible = true
		return nil
	})
}

// WithFormKeyMap installs a custom [huh.KeyMap]. Use [huh.NewDefaultKeyMap]
// and modify before passing. Has no effect in non-interactive mode.
func WithFormKeyMap(km *huh.KeyMap) FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.keymap = km
		return nil
	})
}

// WithFormTimeout aborts the form after d if the user has not submitted.
// Has no effect in non-interactive mode.
func WithFormTimeout(d time.Duration) FormOption {
	return formOpt(func(fc *formConfig) error {
		fc.timeout = d
		return nil
	})
}

// WithGroupTitle sets the title for a form group (page). Only valid inside
// [WithFormGroup].
func WithGroupTitle(s string) GroupOption {
	return groupOpt(func(fg *formGroup) error {
		fg.title = s
		return nil
	})
}

// WithGroupDescription sets the description for a form group (page). Only
// valid inside [WithFormGroup].
func WithGroupDescription(s string) GroupOption {
	return groupOpt(func(fg *formGroup) error {
		fg.description = s
		return nil
	})
}

// WithFormGroup wraps fields and group chrome into a new page within the form.
// Fields used outside any [WithFormGroup] land in the default first group.
//
// Example:
//
//	c.Form(
//	    nabat.WithFormGroup(
//	        nabat.WithGroupTitle("Identity"),
//	        nabat.WithFormField(&name, "Name", ""),
//	    ),
//	    nabat.WithFormGroup(
//	        nabat.WithGroupTitle("Deployment"),
//	        nabat.WithSelectField(&env, "Environment", "", choices, "staging"),
//	    ),
//	)
func WithFormGroup(opts ...GroupOption) FormOption {
	return formOpt(func(fc *formConfig) error {
		var fg formGroup
		var errs ConfigErrors
		for i, o := range opts {
			if o == nil {
				errs.AddErr(fmtErrInvalidOption("group option", i))
				continue
			}
			if err := o.applyToGroup(&fg); err != nil {
				errs.AddErr(err)
			}
		}
		if errs.HasIssues() {
			return &errs
		}
		fc.groups = append(fc.groups, fg)
		return nil
	})
}

// WithFormField adds a typed field to a form or group. T determines the widget
// kind:
//   - string renders a single-line input by default; use [WithMultiline] for
//     multi-line or [WithFilePicker] for a file browser.
//   - bool renders a confirm widget.
//   - Numeric types render a text input with runtime parsing.
//
// The description is rendered as a subtitle beneath the field title; pass ""
// to omit it.
//
// Use [WithSelectField] or [WithMultiSelectField] for select-style widgets.
func WithFormField[T FormFieldValue](
	target *T, title, description string, opts ...FieldOption[T],
) interface {
	FormOption
	GroupOption
} {
	build := func() (fieldEntry, error) {
		if target == nil {
			return nil, fmt.Errorf("nabat: form field %q: target cannot be nil", title)
		}
		var pc promptConfig
		pc.text = title
		pc.description = description
		if err := applyFieldOptions("field", opts, &pc); err != nil {
			return nil, fmt.Errorf("nabat: form field %q: %w", title, err)
		}
		// Extract the typed fallback at construction time so applyFallback
		// never needs an any assertion.
		var fallback T
		if pc.hasFallback {
			v, ok := pc.fallback.(T)
			if !ok {
				return nil, fmt.Errorf("nabat: form field %q: default type mismatch", title)
			}
			fallback = v
		}
		return typedFormField[T]{
			title:    title,
			target:   target,
			fallback: fallback,
			hasFB:    pc.hasFallback,
			build:    buildFormField(target, pc),
		}, nil
	}
	return formFieldDual{
		formApply: func(fc *formConfig) error {
			fe, err := build()
			if err != nil {
				return err
			}
			fc.ensureDefaultGroup()
			fc.groups[0].fields = append(fc.groups[0].fields, fe)
			return nil
		},
		groupApply: func(fg *formGroup) error {
			fe, err := build()
			if err != nil {
				return err
			}
			fg.fields = append(fg.fields, fe)
			return nil
		},
	}
}

// WithSelectField adds a single-select field to a form or group. E is inferred
// from the choices slice and target pointer. The defaultVal value is used in
// non-interactive mode. The description is rendered as a subtitle; pass "" to
// omit it.
func WithSelectField[E comparable](
	target *E, title, description string,
	choices []E, defaultVal E,
	opts ...SelectOption,
) interface {
	FormOption
	GroupOption
} {
	build := func() (fieldEntry, error) {
		if target == nil {
			return nil, fmt.Errorf("nabat: form field %q: target cannot be nil", title)
		}
		var pc promptConfig
		pc.text = title
		pc.description = description
		if err := applySelectOptions("field", opts, &pc); err != nil {
			return nil, fmt.Errorf("nabat: form field %q: %w", title, err)
		}
		return typedFormField[E]{
			title:    title,
			target:   target,
			fallback: defaultVal,
			hasFB:    true,
			build: func() (huh.Field, error) {
				options := make([]huh.Option[E], 0, len(choices))
				for _, c := range choices {
					options = append(options, huh.NewOption(fmt.Sprint(c), c))
				}
				f := huh.NewSelect[E]().Title(title).Options(options...).Value(target)
				if pc.description != "" {
					f = f.Description(pc.description)
				}
				f = f.Filtering(pc.filtering)
				if pc.height > 0 {
					f = f.Height(pc.height)
				}
				if pc.optionsFunc != nil {
					opFn, ok := pc.optionsFunc.(func() []E)
					if !ok {
						return nil, fmt.Errorf("nabat: WithOptionsFunc: internal type mismatch for field %q", title)
					}
					dynOptions := func() []huh.Option[E] {
						raw := opFn()
						out := make([]huh.Option[E], 0, len(raw))
						for _, item := range raw {
							out = append(out, huh.NewOption(fmt.Sprint(item), item))
						}
						return out
					}
					f = f.OptionsFunc(dynOptions, pc.optionsFuncBinds)
				}
				if pc.validate != nil {
					fn := pc.validate
					f = f.Validate(func(v E) error { return fn(v) })
				}
				return f, nil
			},
		}, nil
	}
	return formFieldDual{
		formApply: func(fc *formConfig) error {
			fe, err := build()
			if err != nil {
				return err
			}
			fc.ensureDefaultGroup()
			fc.groups[0].fields = append(fc.groups[0].fields, fe)
			return nil
		},
		groupApply: func(fg *formGroup) error {
			fe, err := build()
			if err != nil {
				return err
			}
			fg.fields = append(fg.fields, fe)
			return nil
		},
	}
}

// WithMultiSelectField adds a multi-select field to a form or group. E is
// inferred from the choices slice. The defaultVal value is used in
// non-interactive mode. The description is rendered as a subtitle; pass "" to
// omit it.
func WithMultiSelectField[E comparable](
	target *[]E, title, description string,
	choices, defaultVal []E,
	opts ...MultiSelectOption,
) interface {
	FormOption
	GroupOption
} {
	build := func() (fieldEntry, error) {
		if target == nil {
			return nil, fmt.Errorf("nabat: form field %q: target cannot be nil", title)
		}
		var pc promptConfig
		pc.text = title
		pc.description = description
		if err := applySelectOptions("field", asSelectOptions(opts), &pc); err != nil {
			return nil, fmt.Errorf("nabat: form field %q: %w", title, err)
		}
		// Defensive copy of defaultVal stored as the typed fallback.
		dv := append([]E(nil), defaultVal...)
		return typedFormField[[]E]{
			title:    title,
			target:   target,
			fallback: dv,
			hasFB:    true,
			build: func() (huh.Field, error) {
				options := make([]huh.Option[E], 0, len(choices))
				for _, c := range choices {
					options = append(options, huh.NewOption(fmt.Sprint(c), c))
				}
				f := huh.NewMultiSelect[E]().Title(title).Options(options...).Value(target)
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
				return f, nil
			},
		}, nil
	}
	return formFieldDual{
		formApply: func(fc *formConfig) error {
			fe, err := build()
			if err != nil {
				return err
			}
			fc.ensureDefaultGroup()
			fc.groups[0].fields = append(fc.groups[0].fields, fe)
			return nil
		},
		groupApply: func(fg *formGroup) error {
			fe, err := build()
			if err != nil {
				return err
			}
			fg.fields = append(fg.fields, fe)
			return nil
		},
	}
}

// WithFormNote adds a display-only note (heading + markdown body) to a form
// or group. Notes are interactive-only: in non-interactive mode they are
// silently skipped.
func WithFormNote(title, body string) interface {
	FormOption
	GroupOption
} {
	e := noteEntry{title: title, body: body}
	return formFieldDual{
		formApply: func(fc *formConfig) error {
			fc.ensureDefaultGroup()
			fc.groups[0].fields = append(fc.groups[0].fields, e)
			return nil
		},
		groupApply: func(fg *formGroup) error {
			fg.fields = append(fg.fields, e)
			return nil
		},
	}
}

// asSelectOptions converts []MultiSelectOption to []SelectOption for
// applySelectOptions.
func asSelectOptions(opts []MultiSelectOption) []SelectOption {
	out := make([]SelectOption, 0, len(opts))
	for _, o := range opts {
		out = append(out, o)
	}
	return out
}

// buildFormField dispatches to the right Huh widget based on T and mode.

func buildFormField[T FormFieldValue](target *T, pc promptConfig) func() (huh.Field, error) {
	return func() (huh.Field, error) {
		var zero T
		switch any(zero).(type) {
		case string:
			s, ok := any(target).(*string)
			if !ok {
				return nil, fmt.Errorf("nabat: internal: expected *string target, got %T", target)
			}
			switch pc.mode {
			case widgetMultiline:
				return buildTextField(s, pc), nil
			case widgetFilePicker:
				return buildFileField(s, pc), nil
			default:
				return buildInputField(s, pc), nil
			}
		case bool:
			b, ok := any(target).(*bool)
			if !ok {
				return nil, fmt.Errorf("nabat: internal: expected *bool target, got %T", target)
			}
			return buildConfirmField(b, pc), nil
		case int, int64, uint, float64, time.Duration:
			return buildNumericInputField(target, pc), nil
		default:
			return nil, fmt.Errorf("nabat: unsupported form field type %T", zero)
		}
	}
}

func buildInputField(target *string, pc promptConfig) huh.Field {
	f := huh.NewInput().Title(pc.text).Value(target)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	if pc.placeholder != "" {
		f = f.Placeholder(pc.placeholder)
	}
	if pc.password {
		f = f.EchoMode(huh.EchoModePassword)
	}
	if pc.charLimit > 0 {
		f = f.CharLimit(pc.charLimit)
	}
	if len(pc.suggestions) > 0 {
		f = f.Suggestions(pc.suggestions)
	}
	if pc.inline {
		f = f.Inline(true)
	}
	if pc.validate != nil {
		fn := pc.validate
		f = f.Validate(func(s string) error { return fn(s) })
	}
	return f
}

func buildConfirmField(target *bool, pc promptConfig) huh.Field {
	f := huh.NewConfirm().Title(pc.text).Value(target)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	if pc.affirmative != "" {
		f = f.Affirmative(pc.affirmative)
	}
	if pc.negative != "" {
		f = f.Negative(pc.negative)
	}
	if pc.inline {
		f = f.Inline(true)
	}
	if pc.validate != nil {
		fn := pc.validate
		f = f.Validate(func(b bool) error { return fn(b) })
	}
	return f
}

func buildTextField(target *string, pc promptConfig) huh.Field {
	f := huh.NewText().Title(pc.text).Value(target)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	if pc.placeholder != "" {
		f = f.Placeholder(pc.placeholder)
	}
	if pc.charLimit > 0 {
		f = f.CharLimit(pc.charLimit)
	}
	if pc.editor {
		f = f.ExternalEditor(true)
		if pc.editorCmd != "" {
			f = f.Editor(pc.editorCmd)
		}
		if pc.editorExtension != "" {
			f = f.EditorExtension(pc.editorExtension)
		}
	}
	if pc.validate != nil {
		fn := pc.validate
		f = f.Validate(func(s string) error { return fn(s) })
	}
	return f
}

func buildFileField(target *string, pc promptConfig) huh.Field {
	f := huh.NewFilePicker().Title(pc.text).Value(target)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	if len(pc.allowedTypes) > 0 {
		f = f.AllowedTypes(pc.allowedTypes)
	}
	if pc.dirAllowed {
		f = f.DirAllowed(true)
	}
	if pc.currentDir != "" {
		f = f.CurrentDirectory(pc.currentDir)
	}
	if pc.showHidden {
		f = f.ShowHidden(true)
	}
	if pc.showSize {
		f = f.ShowSize(true)
	}
	if pc.showPermissions {
		f = f.ShowPermissions(true)
	}
	if pc.validate != nil {
		fn := pc.validate
		f = f.Validate(func(s string) error { return fn(s) })
	}
	return f
}

// buildNumericInputField renders numeric types as a string input.
// huh's Validate fires when the user leaves the field or submits the form;
// that is where we parse the raw string and commit the typed value to *target.
// Pre-populating raw formats *target with [fmt.Sprint] to show the current or
// default value.
func buildNumericInputField[T FormFieldValue](target *T, pc promptConfig) huh.Field {
	raw := fmt.Sprint(*target)
	f := huh.NewInput().Title(pc.text).Value(&raw)
	if pc.description != "" {
		f = f.Description(pc.description)
	}
	if pc.placeholder != "" {
		f = f.Placeholder(pc.placeholder)
	}
	if pc.charLimit > 0 {
		f = f.CharLimit(pc.charLimit)
	}
	if pc.inline {
		f = f.Inline(true)
	}
	userValidate := pc.validate
	f = f.Validate(func(s string) error {
		v, err := parseFormFieldValue[T](s)
		if err != nil {
			return err
		}
		if userValidate != nil {
			if vErr := userValidate(any(v)); vErr != nil {
				return vErr
			}
		}
		*target = v
		return nil
	})
	return f
}

// parseFormFieldValue parses a raw string into the FormFieldValue type T.
func parseFormFieldValue[T FormFieldValue](s string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case int:
		v, err := strconv.Atoi(s)
		if err != nil {
			return zero, fmt.Errorf("expected int, got %q", s)
		}
		result, ok := any(v).(T)
		if !ok {
			return zero, fmt.Errorf("nabat: type mismatch: int -> %T", zero)
		}
		return result, nil
	case int64:
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("expected int64, got %q", s)
		}
		result, ok := any(v).(T)
		if !ok {
			return zero, fmt.Errorf("nabat: type mismatch: int64 -> %T", zero)
		}
		return result, nil
	case uint:
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("expected uint, got %q", s)
		}
		result, ok := any(uint(v)).(T)
		if !ok {
			return zero, fmt.Errorf("nabat: type mismatch: uint -> %T", zero)
		}
		return result, nil
	case float64:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return zero, fmt.Errorf("expected float64, got %q", s)
		}
		result, ok := any(v).(T)
		if !ok {
			return zero, fmt.Errorf("nabat: type mismatch: float64 -> %T", zero)
		}
		return result, nil
	case time.Duration:
		v, err := time.ParseDuration(s)
		if err != nil {
			return zero, fmt.Errorf("expected duration (e.g. 30s, 5m), got %q", s)
		}
		result, ok := any(v).(T)
		if !ok {
			return zero, fmt.Errorf("nabat: type mismatch: duration -> %T", zero)
		}
		return result, nil
	default:
		return zero, fmt.Errorf("nabat: unsupported numeric type %T", zero)
	}
}

// Form runs a typed form built from [FormOption] values. In interactive mode,
// Huh widgets are built and presented to the user. In non-interactive mode,
// every field must have a default (set with [WithDefault]); missing defaults
// are aggregated into a [*ConfigErrors].
// Use [Context.UnsafeForm] when the declarative API cannot express the layout.
//
// Errors:
//   - [*ConfigErrors] when option application or fallback walk fails
//   - errors from the underlying Huh form in interactive mode
func (c *Context) Form(opts ...FormOption) error {
	var fc formConfig
	var errs ConfigErrors
	for i, o := range opts {
		if any(o) == nil {
			errs.AddErr(fmtErrInvalidOption("form option", i))
			continue
		}
		if err := o.apply(&fc); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return &errs
	}

	if len(fc.groups) == 0 {
		return nil
	}

	if !c.interactive {
		return formFallbackWalk(fc.groups)
	}

	huhGroups := make([]*huh.Group, 0, len(fc.groups))
	for i, g := range fc.groups {
		huhFields := make([]huh.Field, 0, len(g.fields))
		for _, fe := range g.fields {
			hf, err := fe.buildHuh()
			if err != nil {
				return fmt.Errorf("nabat: form field %q: %w", fe.fieldTitle(), err)
			}
			huhFields = append(huhFields, hf)
		}
		hg := huh.NewGroup(huhFields...)
		// Chrome precedence: form-level title/description fall through to the
		// first group only when that group has no explicit WithGroupTitle /
		// WithGroupDescription.
		groupTitle := g.title
		groupDesc := g.description
		if i == 0 {
			if groupTitle == "" {
				groupTitle = fc.title
			}
			if groupDesc == "" {
				groupDesc = fc.description
			}
		}
		if groupTitle != "" {
			hg = hg.Title(groupTitle)
		}
		if groupDesc != "" {
			hg = hg.Description(groupDesc)
		}
		huhGroups = append(huhGroups, hg)
	}

	form := huh.NewForm(huhGroups...)
	if fc.accessible {
		form = form.WithAccessible(true)
	}
	if fc.keymap != nil {
		form = form.WithKeyMap(fc.keymap)
	}
	if fc.timeout > 0 {
		form = form.WithTimeout(fc.timeout)
	}
	c.app.applyHuhTheme(form)
	return form.Run()
}

// UnsafeForm runs a Huh form built from raw [huh.Field] values. Use this
// escape hatch when the typed form API cannot express a specific layout.
//
// Errors:
//   - "nabat: form requires interactive terminal" when not interactive
//   - errors from the underlying Huh form
func (c *Context) UnsafeForm(fields ...huh.Field) error {
	if !c.interactive {
		return fmt.Errorf("nabat: form requires interactive terminal")
	}
	form := huh.NewForm(huh.NewGroup(fields...))
	c.app.applyHuhTheme(form)
	return form.Run()
}

// formFallbackWalk writes fallback values into every field's target pointer
// across all groups. Fields without fallbacks are collected into a
// [*ConfigErrors].
func formFallbackWalk(groups []formGroup) error {
	var errs ConfigErrors
	for _, g := range groups {
		for _, fe := range g.fields {
			if err := fe.applyFallback(); err != nil {
				errs.AddErr(err)
			}
		}
	}
	if errs.HasIssues() {
		return &errs
	}
	return nil
}
