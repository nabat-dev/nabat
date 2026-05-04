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
	"errors"
	"fmt"
	"strings"
	"time"
)

// FieldOption is the kind-checked option family for interactive prompts,
// form fields, and declarative arg prompts. T is the bind type at the call
// site. Misuse (e.g. [WithEditor] on a bool field) is a build error.
//
// [FieldOption][T] is the kind-agnostic, annotation-free option family. The
// phantom method fieldOpt(T) is the compile-time type witness. Each helper's
// concrete type implements fieldOpt(T) only for the T values it is valid on.
// When the user writes:
//
//	nabat.WithFormField(&proceed, "Proceed?",
//	    nabat.WithAffirmative("Yes"),    // FieldOption[bool]
//	    nabat.WithDefault(false),        // FieldOption[bool]  (T from false)
//	)
//
// Go infers T=bool from the *bool target and verifies every option satisfies
// FieldOption[bool]. WithAffirmative returns FieldOption[bool]; passing it to
// a string field is a build error because its fieldOpt method only takes bool.
// Multi-kind helpers (WithHint, WithDefault, WithValidate) use generics so
// T is inferred from their own value argument — no annotation required.
// Single-kind helpers (WithAffirmative, WithEditor, etc.) use concrete struct
// types whose fieldOpt(T) is implemented only for the one valid kind.
//
// Most helpers produce a [FieldOption][T] with T inferred from their value
// argument, so no explicit type annotation is needed. The handful of
// zero-argument, single-kind helpers ([WithEditor], [WithMultiline],
// [WithPassword], [WithAffirmative], [WithFilePicker], …) return a concrete
// type that only satisfies [FieldOption] for the relevant kind.
//
// Usage contexts:
//   - Ad-hoc prompts: [Context.Input], [Context.Confirm], [nabat.Select], etc.
//   - Form fields: [WithFormField], [WithSelectField], [WithMultiSelectField]
//   - Declarative arg prompts: [WithPrompt]
type FieldOption[T any] interface {
	fieldOpt(T)
	apply(*promptConfig) error
}

// fieldOpt is the generic concrete implementation for multi-kind and T-bearing
// helpers (WithDefault, WithValidate, WithHint).
type fieldOpt[T any] struct {
	fn func(*promptConfig) error
}

func (f fieldOpt[T]) fieldOpt(T)                   {}
func (f fieldOpt[T]) apply(pc *promptConfig) error { return f.fn(pc) }

// TextInput is the set of Go types that are presented as single-line (or
// multi-line) text inputs. Bool and select types are excluded. It constrains
// [WithExample] and [WithMaxChars]: exact types (not ~T) so that bool,
// []string, and future custom types are excluded cleanly.
type TextInput interface {
	string | int | int64 | uint | float64 | time.Duration
}

// SelectOption configures select and multi-select fields. Its implementors
// satisfy neither [FieldOption][string] nor [FieldOption][bool], so passing
// them to a non-select field is a build error. [SelectOption] and
// [MultiSelectOption] are separate sealed interfaces for [WithFiltering],
// [WithHeight], and [WithLimit].
type SelectOption interface {
	applyToSelect(*promptConfig) error
	sealSelect()
}

// MultiSelectOption configures multi-select fields. Satisfies [SelectOption]
// so it can also be passed to a single-select field, but [WithLimit] is
// rejected there at registration time.
type MultiSelectOption interface {
	SelectOption
	sealMultiSelect()
}

// selectOpt is the concrete type for SelectOption helpers.
type selectOpt struct {
	fn func(*promptConfig) error
}

func (o selectOpt) applyToSelect(pc *promptConfig) error { return o.fn(pc) }
func (selectOpt) sealSelect()                            {}

// multiSelectOpt is the concrete type for MultiSelectOption helpers.
type multiSelectOpt struct {
	fn func(*promptConfig) error
}

func (o multiSelectOpt) applyToSelect(pc *promptConfig) error { return o.fn(pc) }
func (multiSelectOpt) sealSelect()                            {}
func (multiSelectOpt) sealMultiSelect()                       {}

// WithFiltering enables or disables incremental option filtering on select
// and multi-select fields.
func WithFiltering(enabled bool) SelectOption {
	return selectOpt{fn: func(pc *promptConfig) error {
		pc.filtering = enabled
		return nil
	}}
}

// WithHeight sets the visible row count for select and multi-select lists.
// n must be > 0.
func WithHeight(n int) SelectOption {
	return selectOpt{fn: func(pc *promptConfig) error {
		if n <= 0 {
			return errors.New("nabat: WithHeight: n must be > 0")
		}
		pc.height = n
		return nil
	}}
}

// WithLimit caps how many items a multi-select field lets the user pick.
// n must be > 0.
func WithLimit(n int) MultiSelectOption {
	return multiSelectOpt{fn: func(pc *promptConfig) error {
		if n <= 0 {
			return errors.New("nabat: WithLimit: n must be > 0")
		}
		pc.limit = n
		return nil
	}}
}

// WithOptionsFunc recomputes the choices for a select field when the binding
// changes. E is inferred from the function's return type:
//
//	nabat.WithOptionsFunc(func() []string { return states[country] }, &country)
//
// The binds argument matches [huh.Select.OptionsFunc]'s bindings parameter.
func WithOptionsFunc[E comparable](fn func() []E, binds any) SelectOption {
	return selectOpt{fn: func(pc *promptConfig) error {
		pc.optionsFunc = fn
		pc.optionsFuncBinds = binds
		return nil
	}}
}

type passwordOpt struct{}

func (passwordOpt) fieldOpt(string)              {}
func (passwordOpt) apply(pc *promptConfig) error { pc.password = true; return nil }

// WithPassword switches the string input to masked echo mode.
func WithPassword() FieldOption[string] { return passwordOpt{} }

type suggestionsOpt struct{ v []string }

func (suggestionsOpt) fieldOpt(string) {}
func (o suggestionsOpt) apply(pc *promptConfig) error {
	pc.suggestions = append([]string(nil), o.v...)
	return nil
}

// WithSuggestions sets tab-completion suggestions for string inputs.
func WithSuggestions(v ...string) FieldOption[string] { return suggestionsOpt{v: v} }

type maxCharsOpt struct{ n int }

func (maxCharsOpt) fieldOpt(string) {}
func (o maxCharsOpt) apply(pc *promptConfig) error {
	if o.n <= 0 {
		return errors.New("nabat: WithMaxChars: n must be > 0")
	}
	pc.charLimit = o.n
	return nil
}

// WithMaxChars caps the number of characters accepted by a string input.
// n must be > 0.
func WithMaxChars(n int) FieldOption[string] { return maxCharsOpt{n: n} }

type multilineOpt struct{}

func (multilineOpt) fieldOpt(string)              {}
func (multilineOpt) apply(pc *promptConfig) error { pc.mode = widgetMultiline; return nil }

// WithMultiline switches the string field widget to a multi-line text area.
// Mutually exclusive with [WithFilePicker]; combining them is a registration
// error.
func WithMultiline() FieldOption[string] { return multilineOpt{} }

type filePickerOpt struct{}

func (filePickerOpt) fieldOpt(string)              {}
func (filePickerOpt) apply(pc *promptConfig) error { pc.mode = widgetFilePicker; return nil }

// WithFilePicker switches the string field widget to a file-system browser.
// Mutually exclusive with [WithMultiline].
func WithFilePicker() FieldOption[string] { return filePickerOpt{} }

type editorOpt struct{}

func (editorOpt) fieldOpt(string) {}
func (editorOpt) apply(pc *promptConfig) error {
	pc.editor = true
	pc.mode = widgetMultiline
	return nil
}

// WithEditor enables the external editor for multi-line string inputs.
// Implies [WithMultiline].
func WithEditor() FieldOption[string] { return editorOpt{} }

type editorCmdOpt struct{ cmd string }

func (editorCmdOpt) fieldOpt(string) {}
func (o editorCmdOpt) apply(pc *promptConfig) error {
	if strings.TrimSpace(o.cmd) == "" {
		return errors.New("nabat: WithEditorCmd: cmd cannot be empty")
	}
	pc.editorCmd = o.cmd
	return nil
}

// WithEditorCmd sets the external editor binary. Only meaningful when
// [WithEditor] is also set; a registration error is reported otherwise.
func WithEditorCmd(cmd string) FieldOption[string] { return editorCmdOpt{cmd: cmd} }

type editorExtOpt struct{ ext string }

func (editorExtOpt) fieldOpt(string)                {}
func (o editorExtOpt) apply(pc *promptConfig) error { pc.editorExtension = o.ext; return nil }

// WithEditorExtension sets the temp-file extension used by the external
// editor. Only meaningful when [WithEditor] is also set.
func WithEditorExtension(ext string) FieldOption[string] { return editorExtOpt{ext: ext} }

type allowedTypesOpt struct{ types []string }

func (allowedTypesOpt) fieldOpt(string) {}
func (o allowedTypesOpt) apply(pc *promptConfig) error {
	pc.allowedTypes = append([]string(nil), o.types...)
	return nil
}

// WithAllowedTypes restricts file selection to the given extensions.
// Only meaningful when [WithFilePicker] is also set.
func WithAllowedTypes(types ...string) FieldOption[string] { return allowedTypesOpt{types: types} }

type dirAllowedOpt struct{}

func (dirAllowedOpt) fieldOpt(string)              {}
func (dirAllowedOpt) apply(pc *promptConfig) error { pc.dirAllowed = true; return nil }

// WithDirAllowed permits directory selection in the file picker.
// Only meaningful when [WithFilePicker] is also set.
func WithDirAllowed() FieldOption[string] { return dirAllowedOpt{} }

type currentDirOpt struct{ dir string }

func (currentDirOpt) fieldOpt(string)                {}
func (o currentDirOpt) apply(pc *promptConfig) error { pc.currentDir = o.dir; return nil }

// WithCurrentDir sets the starting directory for the file picker.
// Only meaningful when [WithFilePicker] is also set.
func WithCurrentDir(dir string) FieldOption[string] { return currentDirOpt{dir: dir} }

// Options below apply only to bool (confirm) prompts.

type affirmativeOpt struct{ label string }

func (affirmativeOpt) fieldOpt(bool)                  {}
func (o affirmativeOpt) apply(pc *promptConfig) error { pc.affirmative = o.label; return nil }

// WithAffirmative sets the "yes" label for confirm prompts.
func WithAffirmative(label string) FieldOption[bool] { return affirmativeOpt{label: label} }

type negativeOpt struct{ label string }

func (negativeOpt) fieldOpt(bool)                  {}
func (o negativeOpt) apply(pc *promptConfig) error { pc.negative = o.label; return nil }

// WithNegative sets the "no" label for confirm prompts.
func WithNegative(label string) FieldOption[bool] { return negativeOpt{label: label} }

// Inline layout options apply only to string and bool field types.

// inlineStringOpt is the string variant of WithInline.
type inlineStringOpt struct{}

func (inlineStringOpt) fieldOpt(string)              {}
func (inlineStringOpt) apply(pc *promptConfig) error { pc.inline = true; return nil }

// inlineBoolOpt is the bool variant of WithInline.
type inlineBoolOpt struct{}

func (inlineBoolOpt) fieldOpt(bool)                {}
func (inlineBoolOpt) apply(pc *promptConfig) error { pc.inline = true; return nil }

// WithInlineString renders the string prompt title and value on the same line.
func WithInlineString() FieldOption[string] { return inlineStringOpt{} }

// WithInlineBool renders the bool (confirm) prompt title and value on the same
// line.
func WithInlineBool() FieldOption[bool] { return inlineBoolOpt{} }

// WithHint sets the greyed-out example text inside text-style inputs.
// T is inferred from the value: WithHint("alice") ⇒ T=string,
// WithHint(3) ⇒ T=int. Only compiles for [TextInput] types.
//
// Unlike the old [WithPlaceholder], this takes a typed value so T is inferred
// naturally by Go without requiring an explicit type annotation.
func WithHint[T TextInput](v T) FieldOption[T] {
	return fieldOpt[T]{fn: func(pc *promptConfig) error {
		pc.placeholder = fmt.Sprint(v)
		return nil
	}}
}

// WithDefault sets the value returned when the terminal is non-interactive.
// T is inferred from the value: WithDefault("anon") ⇒ T=string.
func WithDefault[T any](v T) FieldOption[T] {
	return fieldOpt[T]{fn: func(pc *promptConfig) error {
		pc.fallback = v
		pc.hasFallback = true
		return nil
	}}
}

// WithValidate attaches a typed validation callback. fn must not be nil.
// T is inferred from fn: WithValidate(func(s string) error { ... }) ⇒ T=string.
func WithValidate[T any](fn func(T) error) FieldOption[T] {
	return fieldOpt[T]{fn: func(pc *promptConfig) error {
		if fn == nil {
			return errors.New("nabat: WithValidate: fn cannot be nil")
		}
		pc.validate = func(v any) error {
			vt, ok := v.(T)
			if !ok {
				return fmt.Errorf("nabat: validation value type mismatch")
			}
			return fn(vt)
		}
		return nil
	}}
}

type showHiddenOpt struct{}

func (showHiddenOpt) fieldOpt(string) {}
func (showHiddenOpt) apply(pc *promptConfig) error {
	if pc.mode != widgetFilePicker {
		return errors.New("nabat: WithShowHidden requires WithFilePicker")
	}
	pc.showHidden = true
	return nil
}

// WithShowHidden makes the file picker display hidden files. Requires
// [WithFilePicker]; a registration-time error is returned otherwise.
func WithShowHidden() FieldOption[string] { return showHiddenOpt{} }

type showSizeOpt struct{}

func (showSizeOpt) fieldOpt(string) {}
func (showSizeOpt) apply(pc *promptConfig) error {
	if pc.mode != widgetFilePicker {
		return errors.New("nabat: WithShowSize requires WithFilePicker")
	}
	pc.showSize = true
	return nil
}

// WithShowSize makes the file picker display file sizes. Requires
// [WithFilePicker]; a registration-time error is returned otherwise.
func WithShowSize() FieldOption[string] { return showSizeOpt{} }

type showPermissionsOpt struct{}

func (showPermissionsOpt) fieldOpt(string) {}
func (showPermissionsOpt) apply(pc *promptConfig) error {
	if pc.mode != widgetFilePicker {
		return errors.New("nabat: WithShowPermissions requires WithFilePicker")
	}
	pc.showPermissions = true
	return nil
}

// WithShowPermissions makes the file picker display file permissions. Requires
// [WithFilePicker]; a registration-time error is returned otherwise.
func WithShowPermissions() FieldOption[string] { return showPermissionsOpt{} }

// applyFieldOptions applies a slice of [FieldOption] to a [promptConfig],
// aggregating errors. Used by ad-hoc prompts, form fields, and
// declarative arg prompts.
func applyFieldOptions[T any](label string, opts []FieldOption[T], pc *promptConfig) error {
	var errs ConfigErrors
	for i, o := range opts {
		if any(o) == nil {
			errs.AddErr(fmtErrInvalidOption(label+" option", i))
			continue
		}
		if err := o.apply(pc); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return &errs
	}
	return nil
}

// applySelectOptions applies [SelectOption] values to a [promptConfig].
func applySelectOptions(label string, opts []SelectOption, pc *promptConfig) error {
	var errs ConfigErrors
	for i, o := range opts {
		if o == nil {
			errs.AddErr(fmtErrInvalidOption(label+" select option", i))
			continue
		}
		if err := o.applyToSelect(pc); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return &errs
	}
	return nil
}
