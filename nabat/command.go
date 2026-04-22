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
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// RunFunc is the handler invoked after positional args and flags are resolved
// on a [Context].
//
// Return nil to indicate success. Return a non-nil error to report failure; the
// error is propagated to the caller of [App.Run] (or [Run] in tests).
//
// RunFunc implementations must not retain the [Context] or use it after the
// function returns.
type RunFunc func(*Context) error

// Command is a Nabat view of one node in the command tree backed by Cobra.
//
// Use [App.Command] and [Command.Command] to create instances. The wrapped Cobra
// command is used internally for parsing and execution.
type Command struct {
	app   *App
	cobra *cobra.Command
	spec  *commandSpec
}

type passthroughDef struct {
	label string
	desc  string
}

type annotationKV struct {
	key, value string
}

// commandSpec is the single source of truth for a command's declarative
// configuration AND its runtime state. Options mutate this directly; the
// [App.meta] map keeps a pointer per [*cobra.Command] so resolution and help
// rendering can read it back.
type commandSpec struct {
	parent *cobra.Command

	description     string
	longDescription string
	aliases         []string
	group           string
	example         string
	run             RunFunc
	flags           []flagDef
	args            []argDef
	validations     []func(*Context) error
	preRun          []func(*Context) error
	postRun         []func(*Context) error
	passthrough     *passthroughDef

	hidden            bool
	deprecatedCommand string
	typoHints         []string
	annotations       []annotationKV
	argCompletions    cobra.CompletionFunc
	parseOpts         []ParseOption
	arityOpts         []ArityOption

	// children holds nested subcommands declared via [WithCommand] inside this
	// command's options. They are registered recursively under this command
	// when the parent command is built; errors aggregate into the same
	// *ConfigErrors as the parent's options.
	children []*commandReg

	// commandInitHooks run once after a [*Command] value exists; see [WithCommandInit].
	commandInitHooks []func(*Command) error
}

// CommandOption configures a subcommand when passed to [App.Command] or
// [Command.Command]. Every value returned by a `With*` constructor in this
// package satisfies CommandOption.
//
// CommandOption is the wider of nabat's two command-option types:
//
//   - [RootOption] is the strict subset of CommandOption that is also valid
//     on the root command. Every RootOption also satisfies [Option] and so
//     can be passed directly to [New] to configure the root.
//   - CommandOption (this type) accepts every option, including those that
//     have no meaningful effect on the root command (such as [WithGroup],
//     [WithHidden], [WithAliases], [WithTypoHints]).
//
// Use the `With*` functions in this package to build a list of options.
//
// Passing a nil CommandOption to [App.Command] or [Command.Command] returns
// an inline error (or panics, in [App.MustCommand] / [Command.MustCommand]).
type CommandOption interface {
	applyToCommand(*commandSpec) error
}

// RootOption is the strict subset of [CommandOption] that is also valid on
// the root command. RootOption values are passed directly to [nabat.New] to
// configure the root command (every RootOption also satisfies [Option]); they
// can also be used inside [WithCommand], [App.Command], and [Command.Command]
// because every RootOption is a [CommandOption].
//
// Constructors that return only [CommandOption] (because cobra ignores them
// on the root command — [WithGroup], [WithHidden], [WithAliases],
// [WithTypoHints]) are build errors when passed directly to [New], but
// remain valid inside [WithCommand] for nested subcommands.
type RootOption interface {
	CommandOption
	Option
	rootCommandOnly()
}

// cmdOpt is the internal adapter for options that are not valid on the root
// command. It satisfies [CommandOption] only.
type cmdOpt struct {
	fn func(*commandSpec) error
}

func (o cmdOpt) applyToCommand(c *commandSpec) error { return o.fn(c) }

// rootOpt is the internal adapter for options that are valid on every command
// (including the root). It satisfies [CommandOption], [RootOption], and
// [Option] — the last so the same value can be passed directly to [New] to
// configure the root command's spec without an intermediate wrapper.
type rootOpt struct {
	fn func(*commandSpec) error
}

func (o rootOpt) applyToCommand(c *commandSpec) error { return o.fn(c) }
func (rootOpt) rootCommandOnly()                      {}
func (o rootOpt) applyToConfig(c *config) error       { return o.fn(c.rootSpec) }

// commandReg is a pending subcommand registration produced by [WithCommand].
// It satisfies all three option interfaces:
//
//   - [Option] (via [commandReg.applyToConfig]): a top-level WithCommand
//     passed to [New] appends to the App's pending list.
//   - [RootOption] (via [commandReg.rootCommandOnly]): WithCommand can be
//     used wherever a RootOption is accepted.
//   - [CommandOption] (via [commandReg.applyToCommand]): a nested WithCommand
//     inside another WithCommand appends as a child of that command.
//
// The same value declares a top-level subcommand at the New site or a nested
// subcommand inside another WithCommand — one name, one nesting primitive.
type commandReg struct {
	name string
	opts []CommandOption
}

func (r *commandReg) applyToConfig(c *config) error {
	c.pendingCommands = append(c.pendingCommands, r)
	return nil
}

func (r *commandReg) applyToCommand(spec *commandSpec) error {
	spec.children = append(spec.children, r)
	return nil
}

func (*commandReg) rootCommandOnly() {}

// WithCommand declares a subcommand. The same value works at every nesting
// level:
//
//   - Passed to [nabat.New], it registers a top-level subcommand under root.
//   - Nested inside another [WithCommand], it registers a child under that
//     command.
//
// Errors aggregate into the [*ConfigErrors] returned by [New], alongside
// option errors and validation errors. For inline (per-call) error handling
// or runtime/dynamic registration, use [App.Command] / [Command.Command]
// instead.
//
// Errors:
//   - registration errors (nil option, empty name, name collisions,
//     flag-registration failures) are surfaced by [New] in the returned
//     [*ConfigErrors]. See [App.Command] for the full per-error list.
//
// Example (declarative tree):
//
//	app, err := New("myctl",
//	    WithDescription("My CLI"),
//	    WithCommand("cluster",
//	        WithDescription("Cluster management"),
//	        WithCommand("scale", WithRun(scaleHandler)),
//	        WithCommand("status", WithRun(statusHandler)),
//	    ),
//	    WithCommand("deploy", WithRun(deployHandler)),
//	)
func WithCommand(name string, opts ...CommandOption) RootOption {
	return &commandReg{name: name, opts: opts}
}

// WithDescription sets the one-line description shown in parent command help
// listings.
func WithDescription(text string) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		c.description = text
		return nil
	}}
}

// WithLongDescription sets a detailed command description shown in full help
// output.
func WithLongDescription(desc string) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		c.longDescription = desc
		return nil
	}}
}

// WithAliases sets command aliases used by the parent command's lookup and
// suggestion logic (see [cobra.Command.Aliases]).
//
// Returns a [CommandOption], not a [RootOption]; passing this directly to
// [New] is a build error because the binary's invocation name is set by the
// OS executable name, so cobra cannot honor aliases on the root command.
func WithAliases(aliases ...string) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		c.aliases = append([]string(nil), aliases...)
		return nil
	}}
}

// WithGroup sets a command group identifier used to visually group commands in
// help output.
//
// Returns a [CommandOption], not a [RootOption]; passing this directly to
// [New] is a build error because the root command has no parent listing to
// be grouped under.
func WithGroup(name string) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		c.group = name
		return nil
	}}
}

// hiddenOpt is the internal carrier for [WithHidden]. It satisfies both
// [CommandOption] and [FlagOption] so a single value works on subcommands
// AND named flags. Positional args are intentionally excluded — hiding a
// declared positional is meaningless because the slot still consumes the
// next CLI token regardless of help visibility.
type hiddenOpt struct{}

func (hiddenOpt) applyToCommand(c *commandSpec) error { c.hidden = true; return nil }
func (hiddenOpt) applyToFlag(s *flagSpec) error       { s.field.hidden = true; return nil }

// WithHidden hides a subcommand or a flag from help listings. The
// command/flag remains invokable; it simply does not appear in --help.
//
// Returns an interface that satisfies [CommandOption] AND [FlagOption], so
// one helper covers both call sites:
//
//	app.MustCommand("internal", WithHidden(), WithRun(func(c *Context) error {
//	    return nil
//	}))
//
//	WithFlag("secret", false, WithHidden())
//
// For conditional hiding, use a plain Go `if` instead of a dedicated helper:
//
//	opts := []CommandOption{WithRun(handler)}
//	if cfg.hidden {
//	    opts = append(opts, WithHidden())
//	}
//	app.MustCommand("internal", opts...)
//
// Returns the union interface (not a [RootOption]); passing this directly to
// [New] is a build error because the root command has no parent listing to
// be hidden from.
func WithHidden() interface {
	CommandOption
	FlagOption
} {
	return hiddenOpt{}
}

// WithTypoHints sets suggested spellings when the user mistypes the subcommand
// name
// (see [cobra.Command.SuggestFor]).
//
// Returns a [CommandOption], not a [RootOption]; passing this directly to
// [New] is a build error because suggestion lookup iterates a command's
// siblings and the root command has none.
func WithTypoHints(aliases ...string) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		c.typoHints = append(c.typoHints, aliases...)
		return nil
	}}
}

// WithAnnotation sets a key/value entry on [cobra.Command.Annotations]. Repeat
// calls
// accumulate; the same key set later overwrites earlier values.
//
// Example:
//
//	app.MustCommand("pods",
//	    WithAnnotation("kubectl.kubernetes.io/default_container", "app"),
//	    WithRun(func(c *Context) error { return nil }),
//	)
//
// Errors:
//   - "nabat: WithAnnotation key cannot be empty": key is "".
func WithAnnotation(key, value string) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if key == "" {
			return fmt.Errorf("nabat: WithAnnotation key cannot be empty")
		}
		c.annotations = append(c.annotations, annotationKV{key: key, value: value})
		return nil
	}}
}

// WithPositionalCompleter overrides automatic positional shell completions.
// Use this option only with a non-nil [CompletionFunc]; automatic
// completions are derived from a single [WithSelectArg] or [WithArg] with
// [WithStringSuggestions] when this option is omitted.
//
// "Completer" names the per-positional callback to keep it distinct from
// [WithCompletion], the App-level switch that installs the `completion`
// subcommand.
//
// Example:
//
//	app.MustCommand("env",
//	    WithPositionalCompleter(func(args []string, toComplete string) ([]string, CompletionDirective) {
//	        return []string{"staging", "prod"}, CompletionNoFileComp
//	    }),
//	    WithRun(func(c *Context) error { return nil }),
//	)
func WithPositionalCompleter(fn CompletionFunc) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: WithPositionalCompleter function cannot be nil")
		}
		c.argCompletions = adaptCompletion(fn)
		return nil
	}}
}

// WithRun sets the command handler.
//
// Example:
//
//	app.MustCommand("deploy",
//	    WithFlag("env", "staging"),
//	    WithRun(func(c *Context) error {
//	        c.Success("done", "env", args.Env) // after c.Bind(&args)
//	        return nil
//	    }),
//	)
//
// Errors:
//   - "nabat: command run cannot be nil": fn is nil.
func WithRun(fn RunFunc) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: command run cannot be nil")
		}
		c.run = fn
		return nil
	}}
}

// WithExample sets the example block shown in the command's help output.
// Write it as plain shell text: comment lines starting with # are dimmed,
// the program name on each line is highlighted, flags (--flag, -f) are
// accented, quoted strings and shell operators are styled accordingly.
//
//	WithExample(`
//	# Deploy to production:
//	myapp deploy production --replicas 3
//	`)
func WithExample(md string) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		c.example = md
		return nil
	}}
}

// WithValidation adds a cross-field validation function that runs after all
// positional args and flags are resolved but before the command handler.
// Multiple calls accumulate; all validations must pass.
//
//	WithValidation(func(c *Context) error {
//	    if format == "json" && !c.Explicit("output") { // use BindAs[string](c, "format"), etc.
//	        return errors.New("--output is required when --format=json")
//	    }
//	    return nil
//	})
//
// Errors:
//   - "nabat: validation function cannot be nil": fn is nil.
func WithValidation(fn func(*Context) error) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: validation function cannot be nil")
		}
		c.validations = append(c.validations, fn)
		return nil
	}}
}

// WithPreRun adds a hook that runs after arg resolution but before the command
// handler. Multiple calls accumulate in order. A non-nil error aborts the run.
//
// Use for auth checks, telemetry setup, or any pre-flight work.
//
// Errors:
//   - "nabat: pre-run function cannot be nil": fn is nil.
func WithPreRun(fn func(*Context) error) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: pre-run function cannot be nil")
		}
		c.preRun = append(c.preRun, fn)
		return nil
	}}
}

// WithPassthrough declares that this command accepts arguments after --.
// label is shown in the usage line as [-- label] (e.g. "command..." →
// [-- command...]).
// The optional desc is shown as a row under the Arguments section in help output.
// Access the passthrough args in the handler via [Context.Passthrough]; use
// [Context.HasPassthrough] to tell whether "--" appeared even with no tokens
// after it.
//
// Example:
//
//	app.MustCommand("exec",
//	    WithArg("service", "", WithRequired()),
//	    WithPassthrough("command [args...]", "command to run once the service is ready"),
//	    WithRun(func(c *Context) error {
//	        if c.HasPassthrough() {
//	            return exec(c.Passthrough())
//	        }
//	        return nil
//	    }),
//	)
//
// Errors:
//   - "nabat: WithPassthrough label cannot be empty": label is "".
func WithPassthrough(label string, desc ...string) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if label == "" {
			return fmt.Errorf("nabat: WithPassthrough label cannot be empty")
		}
		pt := &passthroughDef{label: label}
		if len(desc) > 0 {
			pt.desc = desc[0]
		}
		c.passthrough = pt
		return nil
	}}
}

// WithPostRun adds a hook that runs after the command handler returns, regardless
// of whether it succeeded. Multiple calls accumulate in order.
// Post-run errors are returned only when the command handler itself succeeded.
//
// Use for cleanup, audit logging, or telemetry flushing.
//
// Errors:
//   - "nabat: post-run function cannot be nil": fn is nil.
func WithPostRun(fn func(*Context) error) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: post-run function cannot be nil")
		}
		c.postRun = append(c.postRun, fn)
		return nil
	}}
}

func appendArg(c *commandSpec, name string, vt valueType, s argSpec) error {
	in := argDef{name: name, valueType: vt, config: s.field, prompt: s.prompt}
	if err := in.validate(); err != nil {
		return err
	}
	c.args = append(c.args, in)
	return nil
}

func appendFlag(c *commandSpec, name string, vt valueType, s flagSpec) error {
	fl := flagDef{name: name, valueType: vt, config: s.field}
	if err := fl.validate(); err != nil {
		return err
	}
	c.flags = append(c.flags, fl)
	return nil
}

// ArgValue is the set of value kinds accepted by [WithArg].
type ArgValue interface {
	string | bool | int | int64 | uint | float64 | time.Duration | []string
}

// FlagValue is the set of value kinds accepted by [WithFlag].
type FlagValue interface {
	string | bool | int | int64 | uint | float64 | time.Duration | []string | []bool
}

func argValueTypeFor[T ArgValue]() (valueType, error) {
	switch any(*new(T)).(type) {
	case string:
		return vtString(), nil
	case bool:
		return vtBool(), nil
	case int:
		return vtInt(), nil
	case int64:
		return vtInt64(), nil
	case uint:
		return vtUint(), nil
	case float64:
		return vtFloat(), nil
	case time.Duration:
		return vtDuration(), nil
	case []string:
		return vtStringSlice(), nil
	default:
		return valueType{}, fmt.Errorf("%w for arg value", ErrInvalidValueType)
	}
}

func flagValueTypeFor[T FlagValue]() (valueType, error) {
	switch any(*new(T)).(type) {
	case string:
		return vtString(), nil
	case bool:
		return vtBool(), nil
	case int:
		return vtInt(), nil
	case int64:
		return vtInt64(), nil
	case uint:
		return vtUint(), nil
	case float64:
		return vtFloat(), nil
	case time.Duration:
		return vtDuration(), nil
	case []string:
		return vtStringSlice(), nil
	case []bool:
		return vtBoolSlice(), nil
	default:
		return valueType{}, fmt.Errorf("%w for flag value", ErrInvalidValueType)
	}
}

func normalizeDefaultValue[T any](v T) any {
	switch t := any(v).(type) {
	case []string:
		return append([]string(nil), t...)
	case []bool:
		return append([]bool(nil), t...)
	default:
		return t
	}
}

// WithArg defines one positional argument with adaptive resolution.
// The type argument T is inferred from defaultVal and selects the stored value
// kind (for example int vs uint vs int64 vs [time.Duration]).
// Use typed zero values such as uint(0) or int64(0) rather than untyped 0 when
// the field must not be an int.
func WithArg[T ArgValue](name string, defaultVal T, opts ...ArgOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		s, err := applyArgOptions(opts)
		if err != nil {
			return fmt.Errorf("nabat: arg %q: %w", name, err)
		}
		vt, err := argValueTypeFor[T]()
		if err != nil {
			return fmt.Errorf("nabat: arg %q: %w", name, err)
		}
		s.field.defaultValue = normalizeDefaultValue(defaultVal)
		s.field.hasDefault = true
		return appendArg(c, name, vt, s)
	}}
}

// WithSelectArg defines one positional select argument with adaptive resolution.
// defaultVal must be one of choices (or empty when [WithRequired] is set).
// Add [WithPrompt] to enable interactive prompting when the terminal is a TTY.
// Handlers read the value with [Context.Bind] or [BindAs].
//
// Example:
//
//	app.MustCommand("deploy",
//	    WithSelectArg("env", "", []string{"staging", "production"},
//	        WithRequired(),
//	        WithPrompt("Target environment", ""),
//	    ),
//	    WithRun(handler),
//	)
func WithSelectArg(name, defaultVal string, choices []string, opts ...ArgOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		s, err := applyArgOptions(opts)
		if err != nil {
			return fmt.Errorf("nabat: arg %q: %w", name, err)
		}
		s.field.defaultValue = defaultVal
		s.field.hasDefault = true
		return appendArg(c, name, vtSelect(choices...), s)
	}}
}

// WithMultiSelectArg defines one positional multi-select argument with adaptive
// resolution.
// defaultVal are the pre-selected items; each must be in choices (or pass nil
// when [WithRequired] is set).
// Add [WithPrompt] to enable interactive prompting when the terminal is a TTY.
// Handlers read the value with [Context.Bind] or [BindAs].
func WithMultiSelectArg(name string, defaultVal, choices []string, opts ...ArgOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		s, err := applyArgOptions(opts)
		if err != nil {
			return fmt.Errorf("nabat: arg %q: %w", name, err)
		}
		s.field.defaultValue = defaultVal
		s.field.hasDefault = true
		return appendArg(c, name, vtMultiSelect(choices...), s)
	}}
}

// WithFlag defines a named flag on a command.
// The type argument T is inferred from defaultVal and selects the flag's value
// kind.
// Use typed zero values when an int default is wrong; choose kinds such as
// [uint] or [time.Duration] explicitly. For count flags (for example -vvv), pass
// an int default (typically 0) and add [WithCount].
func WithFlag[T FlagValue](name string, defaultVal T, opts ...FlagOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		s, err := applyFlagOptions(opts)
		if err != nil {
			return fmt.Errorf("nabat: flag %q: %w", name, err)
		}
		vt, err := flagValueTypeFor[T]()
		if err != nil {
			return fmt.Errorf("nabat: flag %q: %w", name, err)
		}
		if s.isCount {
			if vt.kind != valueInt {
				return fmt.Errorf("nabat: flag %q: WithCount() requires an int flag default", name)
			}
			vt = vtCount()
		}
		if vt.kind != valueCount {
			s.field.defaultValue = normalizeDefaultValue(defaultVal)
			s.field.hasDefault = true
		}
		return appendFlag(c, name, vt, s)
	}}
}

// WithSelectFlag defines a named flag whose value must be one of the given
// choices.
// defaultVal must be one of choices (or empty string if [WithRequired] is also
// set).
func WithSelectFlag(name, defaultVal string, choices []string, opts ...FlagOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		s, err := applyFlagOptions(opts)
		if err != nil {
			return fmt.Errorf("nabat: flag %q: %w", name, err)
		}
		s.field.defaultValue = defaultVal
		s.field.hasDefault = true
		return appendFlag(c, name, vtSelect(choices...), s)
	}}
}

// WithMultiSelectFlag defines a named flag accepting multiple values from choices.
// defaultVal items must each be one of choices (or empty slice if [WithRequired]
// is also set).
func WithMultiSelectFlag(name string, defaultVal, choices []string, opts ...FlagOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		s, err := applyFlagOptions(opts)
		if err != nil {
			return fmt.Errorf("nabat: flag %q: %w", name, err)
		}
		s.field.defaultValue = append([]string(nil), defaultVal...)
		s.field.hasDefault = true
		return appendFlag(c, name, vtMultiSelect(choices...), s)
	}}
}

// rootOptionsOpt bundles multiple [RootOption] values into one.
type rootOptionsOpt struct {
	opts []RootOption
}

func (b *rootOptionsOpt) applyToConfig(c *config) error {
	for i, o := range b.opts {
		if o == nil {
			return fmt.Errorf("%w: RootOptions at index %d", ErrNilOption, i)
		}
		if err := o.applyToConfig(c); err != nil {
			return err
		}
	}
	return nil
}

func (b *rootOptionsOpt) applyToCommand(s *commandSpec) error {
	for i, o := range b.opts {
		if o == nil {
			return fmt.Errorf("%w: RootOptions at index %d", ErrNilOption, i)
		}
		if err := o.applyToCommand(s); err != nil {
			return err
		}
	}
	return nil
}

func (*rootOptionsOpt) rootCommandOnly() {}

// CommandOptions composes multiple [CommandOption] values into one. Options apply
// in slice order.
//
// Errors:
//   - [ErrNilOption]: a nil entry appears in opts
//   - errors from individual options
func CommandOptions(opts ...CommandOption) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		for i, o := range opts {
			if o == nil {
				return fmt.Errorf("%w: CommandOptions at index %d", ErrNilOption, i)
			}
			if err := o.applyToCommand(c); err != nil {
				return err
			}
		}
		return nil
	}}
}

// RootOptions composes multiple [RootOption] values into one. Options apply in
// slice order.
//
// Errors:
//   - [ErrNilOption]: a nil entry appears in opts
//   - errors from individual options
func RootOptions(opts ...RootOption) RootOption {
	return &rootOptionsOpt{opts: opts}
}

// WithCommandInit is a [CommandOption] that runs after the command is built,
// receiving the live [*Command] (for example to call [Command.OnPreRun]).
func WithCommandInit(fn func(*Command) error) CommandOption {
	return commandInitOpt(fn)
}

type commandInitOpt func(*Command) error

func (f commandInitOpt) applyToCommand(spec *commandSpec) error {
	if f == nil {
		return fmt.Errorf("%w: WithCommandInit: fn cannot be nil", ErrNilOption)
	}
	spec.commandInitHooks = append(spec.commandInitHooks, f)
	return nil
}

// WithRootInit is like [WithCommandInit] but satisfies [RootOption], so it can be
// passed directly to [New]. For non-root commands, use [WithCommandInit] inside
// [WithCommand].
func WithRootInit(fn func(*Command) error) RootOption {
	return rootOpt{fn: func(spec *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("%w: WithRootInit: fn cannot be nil", ErrNilOption)
		}
		spec.commandInitHooks = append(spec.commandInitHooks, fn)
		return nil
	}}
}

func buildUseString(name string, args []argDef, pt *passthroughDef) string {
	if len(args) == 0 && pt == nil {
		return name
	}
	parts := []string{name}
	for _, in := range args {
		if in.config.required {
			parts = append(parts, "<"+in.name+">")
		} else {
			parts = append(parts, "["+in.name+"]")
		}
	}
	if pt != nil {
		parts = append(parts, "[-- "+pt.label+"]")
	}
	return strings.Join(parts, " ")
}

func applyCommandOptions(opts []CommandOption) (*commandSpec, error) {
	spec := &commandSpec{}
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("%w at index %d", ErrNilOption, i)
		}
		if err := opt.applyToCommand(spec); err != nil {
			return nil, err
		}
	}
	return spec, nil
}

// newCommand registers a child command under parent. Returns the new
// [*Command] and a non-nil error if registration fails (empty name, invalid
// option, name collision, flag-registration failure, finalization error). On
// error, no command is added to parent and the returned [*Command] is nil —
// callers must handle the error before chaining further registrations.
func (a *App) newCommand(parent *cobra.Command, name string, opts ...CommandOption) (*Command, error) {
	if name == "" {
		return nil, fmt.Errorf("nabat: command name cannot be empty")
	}

	spec, err := applyCommandOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("nabat: command %q: %w", name, err)
	}
	spec.parent = parent

	cmd := &cobra.Command{
		Use:     buildUseString(name, spec.args, spec.passthrough),
		Short:   spec.description,
		Long:    spec.longDescription,
		Aliases: spec.aliases,
	}
	if spec.group != "" {
		existing := false
		for _, g := range parent.Groups() {
			if g.ID == spec.group {
				existing = true
				break
			}
		}
		if !existing {
			parent.AddGroup(&cobra.Group{ID: spec.group, Title: spec.group})
		}
		cmd.GroupID = spec.group
	}

	names := make(map[string]string, len(spec.args)+len(spec.flags))
	for _, in := range spec.args {
		names[in.name] = "arg"
	}
	for _, fl := range spec.flags {
		if prev, exists := names[fl.name]; exists {
			return nil, fmt.Errorf("nabat: command %q: name %q is used by both %s and flag: %w", name, fl.name, prev, ErrArgFlagNameCollision)
		}
		names[fl.name] = "flag"
	}

	for i := range spec.flags {
		if regErr := registerFlagOnCommand(cmd, spec.flags[i], a.cfg.envPrefix); regErr != nil {
			return nil, fmt.Errorf("nabat: command %q: %w", name, regErr)
		}
	}

	if finErr := a.finalizeCommand(cmd, spec); finErr != nil {
		return nil, fmt.Errorf("nabat: command %q: %w", name, finErr)
	}

	parent.AddCommand(cmd)
	a.addMeta(cmd, spec)
	wrapper := &Command{app: a, cobra: cmd, spec: spec}
	for i, hook := range spec.commandInitHooks {
		if hook == nil {
			return nil, fmt.Errorf("%w: command %q: WithCommandInit at index %d is nil", ErrNilOption, name, i)
		}
		if hookErr := hook(wrapper); hookErr != nil {
			return nil, fmt.Errorf("nabat: command %q: WithCommandInit: %w", name, hookErr)
		}
	}
	return wrapper, nil
}

// Command creates a child command under this command. Same error contract as
// [App.Command]: returns the new command and a non-nil error if registration
// fails. On error, no child is added and the returned [*Command] is nil.
//
// Errors:
//   - [ErrRegistrationFrozen]: called after [App.Run] / [App.RunArgs]
//   - [ErrNilOption]: a CommandOption in opts is nil
//   - "nabat: command name cannot be empty"
//   - [ErrArgFlagNameCollision]: an arg and a flag share a name
//   - errors from individual [CommandOption] application (wrapped, joined with
//     [ErrInvalidOption] when triggered by a nil option-list entry)
//   - errors from flag registration and command finalization
func (c *Command) Command(name string, opts ...CommandOption) (*Command, error) {
	if c.app.registrationFrozen.Load() {
		return nil, ErrRegistrationFrozen
	}
	return c.app.newCommand(c.cobra, name, opts...)
}

// MustCommand is the panicking variant of [Command.Command], mirroring
// [App.MustCommand]. Use in main() and test setup where chaining
// (`cluster.MustCommand("scale").MustCommand("up", ...)`) is more readable
// than scoped error checks.
//
// Panics if [Command.Command] returns an error. See its documentation for
// the full per-error list.
func (c *Command) MustCommand(name string, opts ...CommandOption) *Command {
	child, err := c.Command(name, opts...)
	if err != nil {
		panic(fmt.Errorf("nabat: %w", err))
	}
	return child
}

// UnsafeCobra returns the underlying [*cobra.Command] for operations that
// Nabat does not abstract (for example C-only Cobra hooks or dynamic
// registration). Mutating the command tree after construction may bypass
// Nabat invariants — prefer [Command.Command] / [Command.MustCommand] and the
// CommandOption family (e.g. [WithRun], [WithArg], [WithFlag]) for standard
// use.
//
// Example:
//
//	parent := app.MustCommand("exec", WithRun(func(c *Context) error { return nil }))
//	raw := parent.UnsafeCobra()
//	raw.Annotations = map[string]string{"x": "y"}
func (c *Command) UnsafeCobra() *cobra.Command {
	return c.cobra
}

// OnPreRun registers a command-level pre-run hook. It is only valid before
// [App.Run].
func (c *Command) OnPreRun(fn func(*Context) error) error {
	if fn == nil {
		return fmt.Errorf("%w: OnPreRun: fn cannot be nil", ErrNilOption)
	}
	if c.app.registrationFrozen.Load() {
		return ErrRegistrationFrozen
	}
	c.spec.preRun = append(c.spec.preRun, fn)
	return nil
}

// OnValidate registers a validation hook. It is only valid before [App.Run].
func (c *Command) OnValidate(fn func(*Context) error) error {
	if fn == nil {
		return fmt.Errorf("%w: OnValidate: fn cannot be nil", ErrNilOption)
	}
	if c.app.registrationFrozen.Load() {
		return ErrRegistrationFrozen
	}
	c.spec.validations = append(c.spec.validations, fn)
	return nil
}

// OnPostRun registers a post-run hook. It is only valid before [App.Run].
func (c *Command) OnPostRun(fn func(*Context) error) error {
	if fn == nil {
		return fmt.Errorf("%w: OnPostRun: fn cannot be nil", ErrNilOption)
	}
	if c.app.registrationFrozen.Load() {
		return ErrRegistrationFrozen
	}
	c.spec.postRun = append(c.spec.postRun, fn)
	return nil
}
