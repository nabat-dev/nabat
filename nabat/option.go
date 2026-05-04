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
)

// normalizeEnvName converts an env-key fragment to the conventional
// uppercase ENV_VAR style. Hyphens and dots become underscores, surrounding
// whitespace is trimmed, and the result is uppercased.
func normalizeEnvName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ToUpper(s)
}

// fieldConfig holds resolution metadata shared by positional args and named flags
// (default value, env wiring, usage text, deprecation, persistence). Both
// [argSpec] and [flagSpec] embed it; constructors mutate it directly.
type fieldConfig struct {
	defaultValue           any
	hasDefault             bool
	required               bool
	persistent             bool
	hidden                 bool
	short                  rune
	hasShort               bool
	envEnabled             bool
	envPrefixed            []string // names that get the [WithEnvPrefix] prefix prepended
	envLiteral             []string // verbatim env var names
	usage                  string
	deprecated             bool
	deprecatedMsg          string
	shorthandDeprecated    bool
	shorthandDeprecatedMsg string
	shellComplete          completionFunc
}

// argSpec is the mutation target for [ArgOption]. Each declarative arg
// constructor (WithArg, WithSelectArg, WithMultiSelectArg) builds one.
type argSpec struct {
	field  fieldConfig
	prompt promptConfig
}

// flagSpec is the mutation target for [FlagOption]. Each flag constructor
// (WithFlag, WithSelectFlag, WithMultiSelectFlag) builds one.
type flagSpec struct {
	field   fieldConfig
	isCount bool
}

// ArgOption configures one declarative positional argument. It is satisfied
// by every helper that targets [WithArg], [WithSelectArg], or
// [WithMultiSelectArg] — for example [WithRequired], [WithUsage], [WithEnv],
// and the generic [WithPrompt] helper.
//
// ArgOption is an interface (not a function alias) so misuse such as passing
// a flag-only option (e.g. [WithShort]) to [WithArg] is rejected by the Go
// compiler at the call site instead of failing later at registration time or
// being silently ignored at runtime.
//
// Options return an error so validation failures (invalid values, bad
// combinations) aggregate into a [ConfigErrors] at command build time.
type ArgOption interface {
	applyToArg(*argSpec) error
}

// FlagOption configures one named flag. It is satisfied by every helper that
// targets [WithFlag], [WithSelectFlag], or [WithMultiSelectFlag] — for
// example [WithRequired], [WithUsage], [WithEnv], [WithShort],
// [WithPersistent], [WithCompleter], and [WithDeprecated].
//
// FlagOption is an interface (not a function alias) so misuse such as
// passing an arg-only option to [WithFlag] is rejected by the Go compiler at
// the call site.
//
// Options return an error so validation failures aggregate into a
// [ConfigErrors] at command build time.
type FlagOption interface {
	applyToFlag(*flagSpec) error
}

// argOptionFn adapts a plain function to [ArgOption] for internal use.
type argOptionFn func(*argSpec) error

func (f argOptionFn) applyToArg(s *argSpec) error { return f(s) }

// flagOptionFn adapts a plain function to [FlagOption] for internal use.
type flagOptionFn func(*flagSpec) error

func (f flagOptionFn) applyToFlag(s *flagSpec) error { return f(s) }

// ArgOptions composes multiple [ArgOption] values into one. Options apply in
// slice order on the same arg slot.
//
// Errors:
//   - [ErrNilOption]: a nil entry appears in opts
//   - errors from individual options
func ArgOptions(opts ...ArgOption) ArgOption {
	return argOptionFn(func(s *argSpec) error {
		for i, o := range opts {
			if o == nil {
				return fmt.Errorf("%w: ArgOptions at index %d", ErrNilOption, i)
			}
			if err := o.applyToArg(s); err != nil {
				return err
			}
		}
		return nil
	})
}

// FlagOptions composes multiple [FlagOption] values into one. Options apply in
// slice order on the same flag slot.
//
// Errors:
//   - [ErrNilOption]: a nil entry appears in opts
//   - errors from individual options
func FlagOptions(opts ...FlagOption) FlagOption {
	return flagOptionFn(func(s *flagSpec) error {
		for i, o := range opts {
			if o == nil {
				return fmt.Errorf("%w: FlagOptions at index %d", ErrNilOption, i)
			}
			if err := o.applyToFlag(s); err != nil {
				return err
			}
		}
		return nil
	})
}

// sharedFieldOpt is the internal carrier for options that are valid on both
// positional args and named flags. It targets the embedded [fieldConfig] in
// [argSpec] and [flagSpec], so a single value satisfies [ArgOption] and
// [FlagOption] simultaneously without two parallel implementations.
type sharedFieldOpt struct {
	fn func(*fieldConfig) error
}

func (o sharedFieldOpt) applyToArg(s *argSpec) error   { return o.fn(&s.field) }
func (o sharedFieldOpt) applyToFlag(s *flagSpec) error { return o.fn(&s.field) }

// applyArgOptions runs every option against a fresh [argSpec] and collects any
// errors into a [ConfigErrors] value. Nil entries report their index.
func applyArgOptions(opts []ArgOption) (argSpec, error) {
	var s argSpec
	var errs ConfigErrors
	for i, opt := range opts {
		if opt == nil {
			errs.AddErr(fmtErrInvalidOption("arg option", i))
			continue
		}
		if err := opt.applyToArg(&s); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return argSpec{}, &errs
	}
	return s, nil
}

// applyFlagOptions runs every option against a fresh [flagSpec] and collects
// any errors into a [ConfigErrors] value. Nil entries report their index.
func applyFlagOptions(opts []FlagOption) (flagSpec, error) {
	var s flagSpec
	var errs ConfigErrors
	for i, opt := range opts {
		if opt == nil {
			errs.AddErr(fmtErrInvalidOption("flag option", i))
			continue
		}
		if err := opt.applyToFlag(&s); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return flagSpec{}, &errs
	}
	return s, nil
}

// The helpers below return an anonymous interface that embeds both
// [ArgOption] and [FlagOption], so the same value can be used inside
// [WithArg]/[WithSelectArg]/[WithMultiSelectArg] AND
// [WithFlag]/[WithSelectFlag]/[WithMultiSelectFlag]. Misuse on a slot that
// does not accept either interface (e.g. as a [CommandOption]) fails at
// compile time.

// WithRequired marks an arg or flag as required; resolution fails if no
// value is found from CLI, env, prompt, or default.
//
// Example (positional):
//
//	WithArg("name", "", WithRequired())
//
// Example (flag):
//
//	WithFlag("token", "", WithRequired(), WithEnv("token"))
func WithRequired() interface {
	ArgOption
	FlagOption
} {
	return sharedFieldOpt{fn: func(c *fieldConfig) error {
		c.required = true
		return nil
	}}
}

// WithUsage sets the usage text shown next to an arg or flag in --help.
func WithUsage(text string) interface {
	ArgOption
	FlagOption
} {
	return sharedFieldOpt{fn: func(c *fieldConfig) error {
		c.usage = text
		return nil
	}}
}

// WithEnv enables environment-variable fallback for an arg or flag. Each
// name is appended to the [WithEnvPrefix] prefix when resolving.
//
// Example:
//
//	WithArg("environment", "", WithEnv("environment"))
//	WithFlag("token", "", WithEnv("token"))
func WithEnv(names ...string) interface {
	ArgOption
	FlagOption
} {
	cleaned := cleanEnvNames(names)
	return sharedFieldOpt{fn: func(c *fieldConfig) error {
		c.envEnabled = true
		c.envPrefixed = append(c.envPrefixed, cleaned...)
		return nil
	}}
}

// WithEnvAlias adds verbatim env var names that must NOT receive the
// [WithEnvPrefix] prefix. Use for legacy/external env names like
// `GITHUB_TOKEN`.
func WithEnvAlias(names ...string) interface {
	ArgOption
	FlagOption
} {
	cleaned := cleanEnvNames(names)
	return sharedFieldOpt{fn: func(c *fieldConfig) error {
		c.envEnabled = true
		c.envLiteral = append(c.envLiteral, cleaned...)
		return nil
	}}
}

func cleanEnvNames(in []string) []string {
	out := make([]string, 0, len(in))
	for _, n := range in {
		if s := strings.TrimSpace(n); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// WithShort sets the one-character shorthand for a flag (for example `-c` for
// `--config`).
func WithShort(r rune) FlagOption {
	return flagOptionFn(func(s *flagSpec) error {
		s.field.short = r
		s.field.hasShort = true
		return nil
	})
}

// WithCount registers the flag as a counter (for example `-v`, `-vv`, `-vvv`).
// Each occurrence increments an [int] value. Combine with [WithFlag] and an int
// default, typically [0]:
//
//	WithFlag("verbose", 0, WithShort('v'), WithCount())
//
// WithCount is valid only when the flag's default type is int.
func WithCount() FlagOption {
	return flagOptionFn(func(s *flagSpec) error {
		s.isCount = true
		return nil
	})
}

// WithPersistent registers the flag as a Cobra persistent flag: it lives on
// the defining command and every descendant inherits it (unless shadowed).
func WithPersistent() FlagOption {
	return flagOptionFn(func(s *flagSpec) error {
		s.field.persistent = true
		return nil
	})
}

// WithCompleter registers a shell completion function for a flag's value.
// fn must not be nil. The returned candidates are shown by the user's shell
// when [WithCompletion] (or any equivalent install path) has been set up.
//
// "Completer" names the per-flag callback to keep it distinct from
// [WithCompletion], the App-level switch that installs the `completion`
// subcommand and its shell-script generators.
//
// Example:
//
//	WithFlag("cluster", "", WithCompleter(
//	    func(args []string, toComplete string) ([]string, CompletionDirective) {
//	        return []string{"eu-1", "us-1"}, CompletionNoFileComp
//	    },
//	))
//
// Errors:
//   - "nabat: WithCompleter function cannot be nil": fn is nil.
func WithCompleter(fn CompletionFunc) FlagOption {
	return flagOptionFn(func(s *flagSpec) error {
		if fn == nil {
			return errors.New("nabat: WithCompleter function cannot be nil")
		}
		s.field.shellComplete = adaptCompletion(fn)
		return nil
	})
}

// envUsageFragment returns a Cobra-style "(env: KEY, ...)" suffix for help
// text. The prefix is prepended to envPrefixed entries; envLiteral entries are
// included verbatim. Returns "" when no env source is configured.
func (cfg fieldConfig) envUsageFragment(envPrefix string) string {
	if !cfg.envEnabled {
		return ""
	}
	parts := make([]string, 0, len(cfg.envPrefixed)+len(cfg.envLiteral))
	for _, n := range cfg.envPrefixed {
		parts = append(parts, envPrefix+normalizeEnvName(n))
	}
	parts = append(parts, cfg.envLiteral...)
	if len(parts) == 0 {
		return ""
	}
	return "(env: " + strings.Join(parts, ", ") + ")"
}
