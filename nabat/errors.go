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
	"slices"
	"strings"
)

var (
	// ErrNilOption indicates a nil option was passed to a constructor.
	ErrNilOption = errors.New("nabat: option cannot be nil")
	// ErrInvalidOption indicates one or more options were invalid.
	ErrInvalidOption = errors.New("nabat: invalid option")
	// ErrInvalidValueType indicates an unrecognized ValueKind.
	ErrInvalidValueType = errors.New("nabat: invalid value type")

	// ErrHandled is a sentinel returned by global pre-run hooks ([App.OnPreRun])
	// or per-command pre-run hooks ([WithPreRun]) to signal that the hook has
	// fully handled the invocation (e.g. printed the version string or help
	// text). The framework treats it as a successful short-circuit: remaining
	// hooks and the command handler are skipped, and [App.Run] returns nil.
	ErrHandled = errors.New("nabat: handled")

	// ErrRegistrationFrozen is returned by App.Command and Command.Command
	// when called after App.Run / App.RunArgs has been invoked. Register
	// every command before running the app.
	ErrRegistrationFrozen = errors.New("nabat: command registration is frozen after App.Run")

	// ErrArgFlagNameCollision is returned (wrapped, with command and field
	// names) when a command declares a positional arg and a flag that share
	// a name. Match it with [errors.Is] in tests instead of substring
	// matching the formatted message.
	ErrArgFlagNameCollision = errors.New("nabat: name used by both arg and flag")
)

// ConfigErrors collects configuration validation failures into one error value.
//
// [New] returns a *ConfigErrors when app-level validation fails after options are
// applied.
// Use [errors.Is] and [errors.As] with [ConfigErrors.Unwrap] to inspect individual
// wrapped errors, or call [ConfigErrors.Error] for a human-readable summary.
type ConfigErrors struct {
	issues []error
}

// AddErr appends a pre-formed error, preserving its wrapping chain so [errors.Is]
// and [errors.As] can match individual issues after [ConfigErrors.Unwrap].
func (e *ConfigErrors) AddErr(err error) {
	if err == nil {
		return
	}
	e.issues = append(e.issues, err)
}

// HasIssues reports whether one or more validation issues exist.
func (e *ConfigErrors) HasIssues() bool {
	return len(e.issues) > 0
}

// Error implements the error interface.
func (e *ConfigErrors) Error() string {
	switch len(e.issues) {
	case 0:
		return "nabat: no configuration errors"
	case 1:
		return e.issues[0].Error()
	default:
		buf := fmt.Appendf(nil, "%d configuration errors:\n", len(e.issues))
		for _, err := range e.issues {
			buf = fmt.Appendf(buf, "  - %s\n", err.Error())
		}
		return strings.TrimRight(string(buf), "\n")
	}
}

// Unwrap returns each underlying issue so [errors.Is] and [errors.As] can inspect
// a [ConfigErrors] value as a multi-error.
func (e *ConfigErrors) Unwrap() []error {
	return slices.Clone(e.issues)
}

// fmtErrInvalidOption formats a "%w at index %d: %w" error joining
// [ErrInvalidOption] with [ErrNilOption] for the given option-list label.
func fmtErrInvalidOption(label string, idx int) error {
	return fmt.Errorf("%w (%s at index %d): %w", ErrInvalidOption, label, idx, ErrNilOption)
}
