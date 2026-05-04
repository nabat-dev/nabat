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

// widgetMode selects which interactive widget is rendered for a string prompt.
// The default (zero value) is a single-line input widget.
type widgetMode int

const (
	widgetInput      widgetMode = iota // default: single-line text input
	widgetMultiline                    // multi-line text area (and optional external editor)
	widgetFilePicker                   // file-system browser
)

// promptConfig is the mutation target for every [FieldOption] helper. Each
// field maps 1:1 to a Huh widget option. The typed-seal mechanism on each
// [FieldOption] helper ensures only kind-appropriate fields are set at compile
// time; cross-kind misuse (e.g. [WithEditor] on a bool field) is a build error.
type promptConfig struct {
	// text is the title rendered above the prompt body. Set automatically
	// from the prompt's first positional argument by the ad-hoc prompts
	// and by [WithPrompt] for declarative args.
	text string

	// description is rendered as the small help line beneath the title.
	// Set via the second positional argument of [WithPrompt] / [WithFormField].
	description string

	// mode selects which widget is rendered for string fields.
	// Non-string fields ignore this field.
	mode widgetMode

	// placeholder is the greyed-out hint inside text-style inputs.
	// Populated by [WithHint]; takes a typed value rendered with fmt.Sprint.
	placeholder string

	// charLimit caps the number of characters accepted by string widgets.
	charLimit int

	// inline asks Huh to render the prompt as a single line where supported.
	inline bool

	// password switches Input widgets to masked echo.
	password bool

	// suggestions populates the autocomplete dropdown of Input widgets and
	// doubles as the choice list for select-style widgets driven by args.
	suggestions []string

	// affirmative / negative replace the default "Yes"/"No" labels on Confirm.
	affirmative string
	negative    string

	// filtering enables incremental filtering on Select / MultiSelect widgets.
	filtering bool

	// height bounds the visible row count of Select/MultiSelect lists.
	height int

	// limit caps how many items a MultiSelect lets the user pick.
	limit int

	// editor and friends configure the external editor for multiline mode.
	editor          bool
	editorCmd       string
	editorExtension string

	// allowedTypes / dirAllowed / currentDir configure the file-picker widget.
	allowedTypes []string
	dirAllowed   bool
	currentDir   string

	// showHidden / showSize / showPermissions are file-picker display flags.
	// Set by [WithShowHidden], [WithShowSize], [WithShowPermissions].
	showHidden      bool
	showSize        bool
	showPermissions bool

	// validate is a typed validation callback set by [WithValidate]. It
	// wraps the user's func(T) error via a type assertion closure.
	validate func(any) error

	// fallback / hasFallback supply a default to use when a prompt is
	// skipped because the program is non-interactive. Set by [WithDefault].
	fallback    any
	hasFallback bool

	// optionsFunc / optionsFuncBinds support dynamic select choices.
	// Set by [WithOptionsFunc]. Stored as any; the build closure casts back.
	optionsFunc      any
	optionsFuncBinds any
}
