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
)

// ErrConfirmationRequired is the sentinel wrapped by [ConfirmationError] when
// a confirm cannot proceed without an interactive terminal or an explicit
// bypass flag.
var ErrConfirmationRequired = errors.New("nabat: confirmation required")

// ConfirmationError is returned when [Context.Confirm] cannot proceed in a
// non-interactive environment (or when a type-to-confirm value does not match).
// Match it with [errors.Is] against [ErrConfirmationRequired], or [errors.As]
// to read [ConfirmationError.BypassHint].
type ConfirmationError struct {
	// Prompt is the confirm prompt text.
	Prompt string
	// BypassHint is an optional flag hint for the error message
	// (for example "--yes" or "--confirm=production").
	BypassHint string
}

// Error returns a short confirmation-required message. A nil receiver and an
// empty [ConfirmationError.BypassHint] yield [ErrConfirmationRequired]'s text;
// a non-empty BypassHint names the flag the caller should pass.
func (e *ConfirmationError) Error() string {
	if e == nil {
		return ErrConfirmationRequired.Error()
	}
	if e.BypassHint != "" {
		return fmt.Sprintf("nabat: confirmation required (pass %s to proceed)", e.BypassHint)
	}
	return ErrConfirmationRequired.Error()
}

// Unwrap returns [ErrConfirmationRequired].
func (e *ConfirmationError) Unwrap() error {
	return ErrConfirmationRequired
}

// WithYes bypasses a moderate [Context.Confirm] when yes is true (typically
// from an app `--yes` flag). When yes is false, Confirm proceeds normally.
//
// WithYes does not bypass type-to-confirm: when [WithConfirmValue] is set,
// the confirm input must still match. Prefer WithYes alone for moderate
// (y/N) actions, and [WithConfirmValue] / [WithConfirmInput] for severe ones.
//
// Example:
//
//	ok, err := c.Confirm("Delete release?", nabat.WithYes(flags.Yes), nabat.WithBypassHint("--yes"))
func WithYes(yes bool) FieldOption[bool] {
	return fieldOpt[bool]{fn: func(pc *promptConfig) error {
		pc.bypassYes = yes
		return nil
	}}
}

// WithConfirmValue enables type-to-confirm for severe destructive actions.
// Interactively, the user must type expected exactly. Non-interactively,
// [WithConfirmInput] must supply the same string (typically from a
// `--confirm` flag).
//
// When WithConfirmValue is set, [WithYes] and [WithDefault] do not satisfy
// the confirm: only a matching [WithConfirmInput] or an interactive match
// proceeds.
//
// Example:
//
//	ok, err := c.Confirm("This will delete production.",
//	    nabat.WithConfirmValue("production"),
//	    nabat.WithConfirmInput(flags.Confirm),
//	    nabat.WithBypassHint("--confirm=production"),
//	)
func WithConfirmValue(expected string) FieldOption[bool] {
	return fieldOpt[bool]{fn: func(pc *promptConfig) error {
		pc.confirmValue = expected
		return nil
	}}
}

// WithConfirmInput supplies the non-interactive bypass value for
// [WithConfirmValue] (typically the app's `--confirm` flag). When it matches
// the expected value, Confirm returns true without prompting.
func WithConfirmInput(input string) FieldOption[bool] {
	return fieldOpt[bool]{fn: func(pc *promptConfig) error {
		pc.confirmInput = input
		return nil
	}}
}

// WithBypassHint sets the flag hint included in [ConfirmationError] messages
// (for example "--yes" or "--confirm=production").
func WithBypassHint(hint string) FieldOption[bool] {
	return fieldOpt[bool]{fn: func(pc *promptConfig) error {
		pc.bypassHint = hint
		return nil
	}}
}

func confirmationError(prompt string, hint string) error {
	return &ConfirmationError{Prompt: prompt, BypassHint: hint}
}
