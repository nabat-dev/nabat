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
	"strings"
)

// DeprecationOption refines the message from [WithDeprecated] and
// [WithDeprecatedShorthand]. Sub-options ([WithDeprecatedSince],
// [WithDeprecatedReplacement]) compose into the single string cobra/pflag
// prints and Nabat shows in --help.
type DeprecationOption interface {
	applyToDeprecation(*deprecationSpec) error
}

type deprecationOptionFunc func(*deprecationSpec) error

func (f deprecationOptionFunc) applyToDeprecation(s *deprecationSpec) error { return f(s) }

// deprecationSpec is the internal accumulator filled by [WithDeprecated] and
// its sub-options before the final message string is composed.
type deprecationSpec struct {
	message     string
	since       string
	replacement string
}

// compose collapses the spec into the single deprecation string consumed by
// cobra and pflag. Format keeps the user-supplied message first so the most
// actionable hint stays at the start of the printed line.
func (s deprecationSpec) compose() string {
	out := strings.TrimSpace(s.message)
	if s.since != "" {
		out += " (since " + s.since + ")"
	}
	if s.replacement != "" {
		out += "; use " + s.replacement + " instead"
	}
	return out
}

// buildDeprecation runs every sub-option against a fresh [deprecationSpec]
// seeded with the primary message. Returns an error if message is empty
// (cobra/pflag refuse empty deprecation strings) or any sub-option fails.
func buildDeprecation(message string, sub []DeprecationOption) (deprecationSpec, error) {
	if strings.TrimSpace(message) == "" {
		return deprecationSpec{}, errors.New("nabat: WithDeprecated requires a non-empty message")
	}
	spec := deprecationSpec{message: message}
	var errs ConfigErrors
	for i, opt := range sub {
		if opt == nil {
			errs.AddErr(fmtErrInvalidOption("deprecation option", i))
			continue
		}
		if err := opt.applyToDeprecation(&spec); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return deprecationSpec{}, &errs
	}
	return spec, nil
}

// deprecatedOpt carries a composed deprecation message for a command or flag.
// It satisfies [CommandOption] and [FlagOption]. Positionals are excluded
// because cobra/pflag have no deprecation hook for them.
type deprecatedOpt struct {
	spec deprecationSpec
}

func (o deprecatedOpt) applyToCommand(c *commandSpec) error {
	c.deprecatedCommand = o.spec.compose()
	return nil
}

func (o deprecatedOpt) applyToFlag(s *flagSpec) error {
	s.field.deprecated = true
	s.field.deprecatedMsg = o.spec.compose()
	return nil
}

// WithDeprecated marks a subcommand or flag as deprecated. cobra prints the
// composed message on use; Nabat also shows it in --help. Positionals are
// unsupported (compile-time error via the return type). Sub-options compose
// into the same message.
//
// Example:
//
//	WithDeprecated("use `new-cmd` instead", WithDeprecatedSince("v0.7.0"))
//	WithFlag("config", "", WithDeprecated("use --settings instead",
//	    WithDeprecatedReplacement("--settings")))
func WithDeprecated(message string, sub ...DeprecationOption) interface {
	CommandOption
	FlagOption
} {
	spec, err := buildDeprecation(message, sub)
	if err != nil {
		return deprecatedOptErr{err: err}
	}
	return deprecatedOpt{spec: spec}
}

// deprecatedOptErr defers a build error from [WithDeprecated] so it surfaces
// alongside other option errors during command registration.
type deprecatedOptErr struct {
	err error
}

func (o deprecatedOptErr) applyToCommand(*commandSpec) error { return o.err }
func (o deprecatedOptErr) applyToFlag(*flagSpec) error       { return o.err }

// WithDeprecatedShorthand marks a flag's shorthand as deprecated while the
// long form remains current. Requires [WithShort] on the same flag.
//
// Example:
//
//	WithFlag("config", "", WithShort('c'),
//	    WithDeprecatedShorthand("use --config instead of -c"))
func WithDeprecatedShorthand(message string, sub ...DeprecationOption) FlagOption {
	spec, err := buildDeprecation(message, sub)
	if err != nil {
		return flagOptionFn(func(*flagSpec) error { return err })
	}
	return flagOptionFn(func(s *flagSpec) error {
		s.field.shorthandDeprecated = true
		s.field.shorthandDeprecatedMsg = spec.compose()
		return nil
	})
}

// WithDeprecatedSince appends " (since <version>)" to the deprecation message.
//
// Example:
//
//	WithDeprecated("use --output instead", WithDeprecatedSince("v0.7.0"))
func WithDeprecatedSince(version string) DeprecationOption {
	return deprecationOptionFunc(func(s *deprecationSpec) error {
		s.since = strings.TrimSpace(version)
		return nil
	})
}

// WithDeprecatedReplacement appends "; use <symbol> instead" to the
// deprecation message.
//
// Example:
//
//	WithDeprecated("legacy", WithDeprecatedReplacement("--output"))
func WithDeprecatedReplacement(symbol string) DeprecationOption {
	return deprecationOptionFunc(func(s *deprecationSpec) error {
		s.replacement = strings.TrimSpace(symbol)
		return nil
	})
}
