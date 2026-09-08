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
// on a [Context]. Return nil on success; a non-nil error propagates to the
// caller of [App.Run] (or [Run] in tests). Do not retain or use the [Context]
// after the function returns.
type RunFunc func(*Context) error

// Command is a Nabat view of one node in the command tree backed by Cobra.
// Create instances with [App.Command] and [Command.Command].
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

// CommandOption configures a subcommand for [App.Command] or [Command.Command].
// [RootOption] is also valid on the root (and as [Option] for [New]);
// [WithGroup], [WithHidden], [WithAliases], and [WithTypoHints] are
// CommandOption-only. A nil option returns an error (or panics in Must*).
type CommandOption interface {
	applyToCommand(*commandSpec) error
}

// RootOption is the subset of [CommandOption] also valid on the root and as
// [Option] for [New]. Constructors that return only [CommandOption]
// ([WithGroup], [WithHidden], [WithAliases], [WithTypoHints]) error when
// passed to [New].
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
// [Option], so the same value can be passed directly to [New] to configure
// the root command's spec without an intermediate wrapper.
type rootOpt struct {
	fn func(*commandSpec) error
}

func (o rootOpt) applyToCommand(c *commandSpec) error { return o.fn(c) }
func (rootOpt) rootCommandOnly()                      {}
func (o rootOpt) applyToConfig(c *config) error       { return o.fn(c.rootSpec) }

// commandReg is a pending subcommand registration from [WithCommand]. It
// satisfies [Option], [RootOption], and [CommandOption] so the same value
// registers under root ([New]) or as a nested child ([WithCommand]).
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

// WithCommand declares a subcommand for [New] (or nested under another
// [WithCommand]). Registration errors join the [*ConfigErrors] from [New].
// For per-call errors or dynamic registration, use [App.Command].
//
// Example:
//
//	New("myctl",
//	    WithCommand("cluster",
//	        WithCommand("scale", WithRun(scale)),
//	    ),
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

// WithAliases sets command aliases for parent lookup and suggestions (see
// [cobra.Command.Aliases]). Returns a [CommandOption], not a [RootOption];
// passing it to [New] is a build error.
func WithAliases(aliases ...string) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		c.aliases = append([]string(nil), aliases...)
		return nil
	}}
}

// WithGroup sets a help-listing group identifier. Returns a [CommandOption],
// not a [RootOption]; passing it to [New] is a build error.
func WithGroup(name string) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		c.group = name
		return nil
	}}
}

// hiddenOpt is the internal carrier for [WithHidden]. It satisfies both
// [CommandOption] and [FlagOption] so a single value works on subcommands
// AND named flags. Positional args are intentionally excluded: hiding a
// declared positional is meaningless because the slot still consumes the
// next CLI token regardless of help visibility.
type hiddenOpt struct{}

func (hiddenOpt) applyToCommand(c *commandSpec) error { c.hidden = true; return nil }
func (hiddenOpt) applyToFlag(s *flagSpec) error       { s.field.hidden = true; return nil }

// WithHidden hides a subcommand or flag from help listings while keeping it
// invokable. The return type satisfies both [CommandOption] and [FlagOption].
// It is not a [RootOption]; passing it to [New] is a build error.
//
// Example:
//
//	app.MustCommand("internal", WithHidden(), WithRun(handler))
//	WithFlag("secret", false, WithHidden())
func WithHidden() interface {
	CommandOption
	FlagOption
} {
	return hiddenOpt{}
}

// WithTypoHints sets suggested spellings for mistyped subcommand names (see
// [cobra.Command.SuggestFor]). Returns a [CommandOption], not a [RootOption];
// passing it to [New] is a build error.
func WithTypoHints(aliases ...string) CommandOption {
	return cmdOpt{fn: func(c *commandSpec) error {
		c.typoHints = append(c.typoHints, aliases...)
		return nil
	}}
}

// WithAnnotation sets a key/value entry on [cobra.Command.Annotations]. Repeat
// calls accumulate; the same key set later overwrites earlier values.
// key must be non-empty.
//
// Example:
//
//	WithAnnotation("kubectl.kubernetes.io/default_container", "app")
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
// fn must be non-nil. Distinct from [WithCompletion] (the `completion`
// subcommand). When omitted, completions come from a single [WithSelectArg]
// or [WithArg] with [WithStringSuggestions].
//
// Example:
//
//	WithPositionalCompleter(func(args []string, toComplete string) ([]string, CompletionDirective) {
//	    return []string{"staging", "prod"}, CompletionNoFileComp
//	})
func WithPositionalCompleter(fn CompletionFunc) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: WithPositionalCompleter function cannot be nil")
		}
		c.argCompletions = adaptCompletion(fn)
		return nil
	}}
}

// WithRun sets the command handler. fn must be non-nil.
//
// Example:
//
//	WithRun(func(c *Context) error {
//	    c.Success("done")
//	    return nil
//	})
func WithRun(fn RunFunc) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: command run cannot be nil")
		}
		c.run = fn
		return nil
	}}
}

// WithExample sets the example block shown in help. Write plain shell text;
// comments (#), the program name, flags, and quotes are styled in help.
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

// WithValidation adds a cross-field check after args and flags resolve and
// before the handler. Multiple calls accumulate; all must pass. fn must be
// non-nil.
func WithValidation(fn func(*Context) error) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: validation function cannot be nil")
		}
		c.validations = append(c.validations, fn)
		return nil
	}}
}

// WithPreRun adds a hook after arg resolution and before the handler.
// Multiple calls accumulate in order; a non-nil error aborts the run.
// fn must be non-nil.
func WithPreRun(fn func(*Context) error) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if fn == nil {
			return fmt.Errorf("nabat: pre-run function cannot be nil")
		}
		c.preRun = append(c.preRun, fn)
		return nil
	}}
}

// WithPassthrough declares arguments after `--`. label appears in usage as
// `[-- label]` and must be non-empty; optional desc appears under Arguments
// in help. Read tokens with [Context.Passthrough]; detect `--` with
// [Context.HasPassthrough].
//
// Example:
//
//	WithPassthrough("command [args...]", "command to run once ready")
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

// WithPostRun adds a hook after the handler returns, success or failure.
// Multiple calls accumulate in order. Post-run errors are returned only when
// the handler itself succeeded. fn must be non-nil.
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

// WithArg defines one positional argument with adaptive resolution. T is
// inferred from defaultVal. Use typed zeros (uint(0), int64(0)) when an int
// default is wrong.
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

// WithSelectArg defines one positional select argument. defaultVal must be in
// choices (or empty with [WithRequired]). Add [WithPrompt] for TTY prompting.
// Read the value with [Context.Bind] or [BindAs].
//
// Example:
//
//	WithSelectArg("env", "", []string{"staging", "production"},
//	    WithRequired(), WithPrompt("Target environment", ""))
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

// WithMultiSelectArg defines one positional multi-select argument. defaultVal
// items must be in choices (or nil with [WithRequired]). Add [WithPrompt] for
// TTY prompting. Read the value with [Context.Bind] or [BindAs].
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

// WithFlag defines a named flag. T is inferred from defaultVal. Use typed zeros
// when an int default is wrong. For count flags (-vvv), pass an int default
// (typically 0) and [WithCount].
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

// WithSelectFlag defines a named flag whose value must be one of choices.
// defaultVal must be in choices (or empty with [WithRequired]).
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

// WithMultiSelectFlag defines a named flag accepting multiple values from
// choices. defaultVal items must be in choices (or empty with [WithRequired]).
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
// in slice order. A nil entry returns [ErrNilOption]; other failures come from
// the individual options.
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
// slice order. A nil entry returns [ErrNilOption]; other failures come from
// the individual options.
func RootOptions(opts ...RootOption) RootOption {
	return &rootOptionsOpt{opts: opts}
}

// WithCommandInit is a [CommandOption] that runs after the command is built,
// receiving the live [*Command] (for example to call [Command.OnPreRun]).
// It returns [ErrNilOption] if fn is nil.
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
// [WithCommand]. It returns [ErrNilOption] if fn is nil.
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

// newCommand registers a child command under parent. On error, no command is
// added and the returned [*Command] is nil (empty name, invalid option, name
// collision, flag-registration failure, or finalization error).
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
			a.rollbackCommand(parent, cmd)
			return nil, fmt.Errorf("%w: command %q: WithCommandInit at index %d is nil", ErrNilOption, name, i)
		}
		if hookErr := hook(wrapper); hookErr != nil {
			a.rollbackCommand(parent, cmd)
			return nil, fmt.Errorf("nabat: command %q: WithCommandInit: %w", name, hookErr)
		}
	}
	return wrapper, nil
}

// Command creates a child command under this command. On error, no child is
// added and the returned [*Command] is nil. Same error contract as [App.Command]
// ([ErrRegistrationFrozen], [ErrNilOption], empty name,
// [ErrArgFlagNameCollision], and option or finalization failures).
func (c *Command) Command(name string, opts ...CommandOption) (*Command, error) {
	if c.app.registrationFrozen.Load() {
		return nil, ErrRegistrationFrozen
	}
	return c.app.newCommand(c.cobra, name, opts...)
}

// MustCommand is the panicking variant of [Command.Command], mirroring
// [App.MustCommand]. Prefer it in main() and tests for chaining. Panics if
// [Command.Command] returns an error.
func (c *Command) MustCommand(name string, opts ...CommandOption) *Command {
	child, err := c.Command(name, opts...)
	if err != nil {
		panic(fmt.Errorf("nabat: %w", err))
	}
	return child
}

// UnsafeCobra returns the underlying [*cobra.Command] for escape-hatch use.
// Mutating the tree after construction may bypass Nabat invariants; prefer
// [Command.Command] and CommandOption helpers for standard use.
//
// Example:
//
//	raw := parent.UnsafeCobra()
//	raw.Annotations = map[string]string{"x": "y"}
func (c *Command) UnsafeCobra() *cobra.Command {
	return c.cobra
}

// OnPreRun registers a command-level pre-run hook. Only valid before [App.Run].
// It returns [ErrNilOption] if fn is nil and [ErrRegistrationFrozen] after Run.
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

// OnValidate registers a validation hook. Only valid before [App.Run].
// It returns [ErrNilOption] if fn is nil and [ErrRegistrationFrozen] after Run.
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

// OnPostRun registers a post-run hook. Only valid before [App.Run].
// It returns [ErrNilOption] if fn is nil and [ErrRegistrationFrozen] after Run.
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
