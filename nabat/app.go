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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopherly.dev/termio"
	"gopherly.dev/termio/colorprofile"

	"nabat.dev/theme"
)

type config struct {
	name      string
	envPrefix string

	// theme is the recipe a [WithTheme] / [WithCustomTheme] option installed.
	// It is resolved once per App at finalize time against the detected
	// [theme.Capabilities]; the result lands in resolvedTheme. The recipe
	// itself stays around so tests and tooling can re-resolve under
	// different capabilities without re-applying every option.
	//
	// Stored as [theme.Resolver] (interface) so [theme.Theme] (the common
	// declarative form) and bespoke programmatic resolvers both fit
	// without an extra union type. nil means no theme; finalize uses
	// the zero ResolvedTheme in that case, which renders as terminal
	// defaults.
	//
	// Multiple [WithTheme] / [WithCustomTheme] calls compose: the
	// last one wins. There is no mutual-exclusion check; the override
	// machinery in [WithThemeOverride] handles the "tweak one slot of
	// an installed theme" case without requiring a custom theme.
	theme theme.Resolver

	// themeOverrides holds [theme.Override] mutations registered via
	// [WithThemeOverride] (and friends). They apply to the resolved
	// theme at finalize time, after the resolver runs but before the
	// resulting [theme.ResolvedTheme] is locked in. Overrides only
	// apply when c.theme is a concrete [theme.Theme] (not a bespoke
	// [theme.Resolver]); the latter does its own resolution and the
	// catalog overrides have nowhere safe to land.
	themeOverrides []theme.Override

	// strictThemeRequirements promotes the missing-token warning
	// into a hard error returned from [New]. Default is warn-only:
	// the App writes the diagnostic to [IOStreams.ErrOut]
	// but lets construction succeed so an opinionated minimal
	// theme can still ship.
	strictThemeRequirements bool

	// resolvedTheme is the immutable token / accessor surface every consumer
	// reads from. It is populated by [App.finalize] just after IOStreams is
	// known and before any command runs; reads after that are race-free
	// because the value is never mutated.
	resolvedTheme theme.ResolvedTheme

	// rootSpec accumulates RootOption values applied directly to nabat.New(...).
	// Every RootOption value satisfies Option via [rootOpt.applyToConfig], which
	// writes into rootSpec. The root's *cobra.Command is built from rootSpec
	// inside [New].
	rootSpec *commandSpec

	// pendingCommands holds top-level subcommands declared via [WithCommand]
	// at the New(...) level. They are registered after the root command, help,
	// version, and extensions are wired up. Errors aggregate into a single
	// [*ConfigErrors] returned from [New].
	pendingCommands []*commandReg

	io         *IOStreams
	outProfile colorprofile.Profile

	logger *slog.Logger

	// errorHandler, when non-nil, replaces default CLI error printing in [App.Run].
	errorHandler func(error)

	// help is a built-in core feature; see help_options.go for the public
	// option surface, the helpConfig struct, and [App.registerHelp] for how
	// these fields drive registration. The pointer is never nil after
	// defaultConfig(); the --help flag is on by default and the `help`
	// subcommand is opt-in via [WithHelpCommand].
	help *helpConfig

	// version, when non-nil, enables the built-in version feature; see
	// version_options.go for the public option surface and [App.registerVersion]
	// for how this drives registration.
	version *versionConfig

	// completion, when non-nil, enables the built-in completion subcommand;
	// see completion_options.go for the public option surface and
	// [App.registerCompletion] for how this drives registration.
	completion *completionConfig

	// extensions are installed by [WithExtension] and run inside [New] after
	// the root command and core features (help, version, completion) are
	// wired up.
	extensions []Extension
}

func defaultConfig() (*config, error) {
	t, err := theme.Get(theme.Default)
	if err != nil {
		return nil, fmt.Errorf("nabat: defaultConfig: built-in default theme: %w", err)
	}
	env := os.Environ()
	policy := colorprofile.Detect(os.Stdout, env)
	return &config{
		theme:      t,
		io:         termio.System(termio.WithColorPolicy(policy)),
		outProfile: policy.Profile(),
		help:       defaultHelpConfig(),
		rootSpec:   &commandSpec{},
	}, nil
}

// finalize resolves the theme against detected [theme.Capabilities]
// during [New]. For a [theme.Theme], errors come from [theme.Theme.ResolveErr];
// other [theme.Resolver] values have no error channel.
func (c *config) finalize() error {
	caps := detectCapabilities(c.io, c.outProfile)

	var resolved theme.ResolvedTheme
	if c.theme != nil {
		if t, ok := c.theme.(theme.Theme); ok {
			if len(c.themeOverrides) > 0 {
				t = t.With(c.themeOverrides...)
			}
			r, err := t.ResolveErr(caps)
			if err != nil {
				return fmt.Errorf("nabat: %w", err)
			}
			resolved = r
		} else {
			// Bespoke resolvers self-resolve; we cannot apply
			// per-Palette overrides into an opaque Resolver without
			// breaking its contract. Surface this as a no-op
			// silently; the user opted into a custom resolver and
			// the override path is for the declarative Theme case.
			resolved = c.theme.Resolve(caps)
		}
	}
	c.resolvedTheme = resolved

	if reqErr := c.checkThemeRequirements(); reqErr != nil {
		return reqErr
	}

	return nil
}

// checkThemeRequirements validates core and [ExtensionWithRequirements]
// token needs. Warns on [IOStreams.ErrOut] by default; fails [New] when
// [WithStrictThemeRequirements] is set.
func (c *config) checkThemeRequirements() error {
	reqs := append([]theme.Requirement(nil), theme.CoreRequirements()...)
	for _, ext := range c.extensions {
		if er, ok := ext.(ExtensionWithRequirements); ok {
			reqs = append(reqs, er.ThemeRequires())
		}
	}
	if err := c.resolvedTheme.CheckRequirements(reqs); err != nil {
		if c.strictThemeRequirements {
			return fmt.Errorf("nabat: %w", err)
		}
		if c.io != nil {
			//nolint:errcheck // Best-effort warning; a stderr write failure here cannot meaningfully be surfaced.
			fmt.Fprintf(c.io.ErrOut, "nabat: warning: %v\n", err)
		}
	}
	return nil
}

func (c *config) validate() error {
	var errs ConfigErrors
	if c.name == "" {
		errs.AddErr(errors.New("nabat: app name cannot be empty"))
	}
	if c.io == nil {
		errs.AddErr(errors.New("nabat: IOStreams cannot be nil"))
	}
	if c.help != nil {
		if err := c.help.validate(); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return &errs
	}
	return nil
}

// Option configures an [App] in [New] or [MustNew]. App-level options,
// [RootOption] values, and [WithCommand] all satisfy Option.
// A [CommandOption] that is not a [RootOption] (for example [WithHidden])
// is a build error at the root; use it inside [WithCommand].
// A nil Option yields [ErrNilOption].
type Option interface {
	applyToConfig(*config) error
}

// optionFn is the internal adapter that turns a func(*config) error into an
// [Option]. App-level option constructors return optionFn(...) so they can
// satisfy the Option interface without users having to declare a new struct
// type per option.
type optionFn func(*config) error

func (f optionFn) applyToConfig(c *config) error { return f(c) }

// AppOptions composes multiple [Option] values into one. Options apply in slice
// order. A nil entry returns [ErrNilOption]; other failures come from the
// individual options.
func AppOptions(opts ...Option) Option {
	return optionFn(func(c *config) error {
		for i, o := range opts {
			if o == nil {
				return fmt.Errorf("%w: AppOptions at index %d", ErrNilOption, i)
			}
			if err := o.applyToConfig(c); err != nil {
				return err
			}
		}
		return nil
	})
}

// WithTheme installs a built-in theme by name (see [nabat.dev/theme]).
// Prefer theme constants such as [theme.Dracula]. Plain strings from flags
// or env vars also work. Later [WithTheme] or [WithCustomTheme] calls win;
// use [WithThemeOverride] for per-token tweaks.
//
// Example:
//
//	New("myctl", WithTheme(theme.Dracula))
//
// An unknown name fails and is joined into [*ConfigErrors] from [New],
// with the list of available names.
func WithTheme(name string) Option {
	return optionFn(func(c *config) error {
		t, err := theme.Get(name)
		if err != nil {
			return err
		}
		c.theme = t
		return nil
	})
}

// WithCustomTheme installs a programmatic [theme.Resolver].
// A [theme.Theme] value works directly. Later [WithTheme] or
// [WithCustomTheme] calls win; prefer [WithThemeOverride] for one-off tokens.
//
// Example:
//
//	app, err := nabat.New("myctl", nabat.WithCustomTheme(acmeTheme))
//
// It returns [ErrNilOption] if r is nil.
func WithCustomTheme(r theme.Resolver) Option {
	return optionFn(func(c *config) error {
		if r == nil {
			return fmt.Errorf("%w: WithCustomTheme resolver cannot be nil", ErrNilOption)
		}
		c.theme = r
		return nil
	})
}

// WithThemeOverride applies a per-token style on top of the active theme.
// Later calls for the same token win. Overrides apply to every variant of a
// [theme.Theme]; they are ignored for opaque [theme.Recipe] resolvers.
//
// Example:
//
//	nabat.New("myctl",
//	    nabat.WithTheme(theme.Dracula),
//	    nabat.WithThemeOverride(theme.StatusError, magenta),
//	)
func WithThemeOverride(t theme.Token, s lipgloss.Style) Option {
	return optionFn(func(c *config) error {
		c.themeOverrides = append(c.themeOverrides, theme.SetToken(t, s))
		return nil
	})
}

// WithThemeOverrides applies several [theme.Override] values in order.
// Same composition rules as [WithThemeOverride]. A nil entry returns
// [ErrNilOption].
func WithThemeOverrides(overrides ...theme.Override) Option {
	return optionFn(func(c *config) error {
		for i, o := range overrides {
			if o == nil {
				return fmt.Errorf("%w: WithThemeOverrides[%d] is nil", ErrNilOption, i)
			}
		}
		c.themeOverrides = append(c.themeOverrides, overrides...)
		return nil
	})
}

// WithStrictThemeRequirements turns missing theme-token warnings into a
// hard [*ConfigErrors] failure from [New]. Without it, the App only prints
// a diagnostic to stderr. Useful in tests; leave off in production if a
// minimal theme is intentional.
func WithStrictThemeRequirements() Option {
	return optionFn(func(c *config) error {
		c.strictThemeRequirements = true
		return nil
	})
}

// WithEnvPrefix sets the prefix prepended to the primary key from [WithEnv] on
// each field.
// Fields without [WithEnv] do not read the environment.
// The value should include a trailing underscore when logical keys should read as
// separate words (for example "MYAPP_" plus key "token" becomes MYAPP_TOKEN).
//
// Example:
//
//	New("myctl", WithEnvPrefix("MYAPP_"))
func WithEnvPrefix(prefix string) Option {
	return optionFn(func(c *config) error {
		c.envPrefix = prefix
		return nil
	})
}

// WithIO replaces the [App]'s [IOStreams] bundle. Tests typically build one
// with [nabattest.NewIO] so they can capture output for assertions; production
// code rarely needs to override the default ([NewSystemIO]).
//
// Passing nil returns [ErrNilOption].
//
// Example:
//
//	io, _, out, _ := nabattest.NewIO()
//	app := MustNew("myctl", WithIO(io))
//	// ... run app ...
//	require.Contains(t, out.String(), "deployed")
func WithIO(s *IOStreams) Option {
	return optionFn(func(c *config) error {
		if s == nil {
			return fmt.Errorf("%w: WithIO IOStreams cannot be nil", ErrNilOption)
		}
		c.io = s
		c.outProfile = colorprofile.Detect(s.RawOut(), os.Environ()).Profile()
		return nil
	})
}

// WithErrorHandler replaces the default error rendering used by [App.Run] when
// execution fails (after Cobra parsing and Nabat resolution). The handler
// receives the same error returned by [App.Run]. When unset, Nabat prints a styled
// "error:"
// line and a styled "Run <command> --help for usage." hint to stderr.
//
// Passing a nil function returns [ErrNilOption].
func WithErrorHandler(fn func(error)) Option {
	return optionFn(func(c *config) error {
		if fn == nil {
			return fmt.Errorf("%w: WithErrorHandler handler cannot be nil", ErrNilOption)
		}
		c.errorHandler = fn
		return nil
	})
}

// WithLogger sets the [*slog.Logger] returned by [Context.Logger].
// Without this or the logging extension, Logger discards records.
// Nil returns [ErrNilOption].
func WithLogger(l *slog.Logger) Option {
	return optionFn(func(c *config) error {
		if l == nil {
			return fmt.Errorf("%w: WithLogger logger cannot be nil", ErrNilOption)
		}
		c.logger = l
		return nil
	})
}

// App is the root CLI for one binary.
// Construct it with [New] or [MustNew], declare commands with [WithCommand]
// or [App.Command], and run with [App.Run].
//
// Register all commands and flags before [App.Run] or [App.RunArgs]. After
// Run, further [App.Command] calls return [ErrRegistrationFrozen]. Concurrent
// registration is not supported; the App is safe for concurrent use only
// after registration finishes.
type App struct {
	cfg *config

	root *cobra.Command

	// mu guards meta writes (command registration) and globalPreRun appends.
	// Reads of both fields happen only after registrationFrozen is true, which
	// establishes a happens-before edge, but the mutex catches the edge case
	// where an extension goroutine outlives its Init call.
	mu           sync.RWMutex
	meta         map[*cobra.Command]*commandSpec
	globalPreRun []func(*Context) error

	// io is the bundled stdin/stdout/stderr plus terminal capability detection
	// shared by every [Context] this App produces. Read it via [App.IO]
	// (`app.IO().Out`, `app.IO().IsStdoutTTY()`) instead of touching
	// os.Stdin/Stdout/Stderr directly. Override at construction with [WithIO].
	io         *IOStreams
	outProfile colorprofile.Profile

	// registrationFrozen flips to true the first time [App.Run] / [App.RunArgs]
	// fires. Once set, [App.Command] and [Command.Command] reject further
	// registration with [ErrRegistrationFrozen]. The signal is one-shot and
	// never reset; an App that has run may not register more commands.
	registrationFrozen atomic.Bool
}

// addMeta stores the commandSpec for cmd. All registration paths go through
// this helper so the mutex is always held on writes.
func (a *App) addMeta(cmd *cobra.Command, spec *commandSpec) {
	a.mu.Lock()
	a.meta[cmd] = spec
	a.mu.Unlock()
}

// rollbackCommand undoes [parent.AddCommand] and [App.addMeta] when a
// [WithCommandInit] hook fails after the child was registered.
func (a *App) rollbackCommand(parent, cmd *cobra.Command) {
	parent.RemoveCommand(cmd)
	a.mu.Lock()
	delete(a.meta, cmd)
	a.mu.Unlock()
}

// globalHooks returns a snapshot of the global pre-run hooks. The snapshot
// is safe to iterate without holding the lock; new hooks appended after the
// snapshot is taken are not included, which is correct because
// registrationFrozen is set before any invocation begins.
func (a *App) globalHooks() []func(*Context) error {
	a.mu.RLock()
	out := append([]func(*Context) error(nil), a.globalPreRun...)
	a.mu.RUnlock()
	return out
}

// AppSurface is the interface extensions interact with during [Extension.Init].
// It is a strict subset of [*App]; extensions that reach outside it won't
// compile, making the extension boundary enforceable by the type system.
//
// The [*App] type satisfies AppSurface, so existing code that holds a [*App]
// can be passed wherever AppSurface is required.
type AppSurface interface {
	// Command registration
	Command(name string, opts ...CommandOption) (*Command, error)
	MustCommand(name string, opts ...CommandOption) *Command
	OnPreRun(fn func(*Context) error) error

	// Identity
	Name() string
	EnvPrefix() string

	// Output handles
	IO() *IOStreams
	Theme() theme.ResolvedTheme
	SetLogger(l *slog.Logger)

	// UnsafeRoot is the escape hatch for extensions that need to traverse or
	// inspect the command tree (for example, man-page generation). Prefer the
	// typed surface above for all standard use.
	UnsafeRoot() *cobra.Command
}

// Extension adds subcommands, hooks, or a logger during [New] via
// [WithExtension]. Init runs after help, version, and completion are wired.
// Extensions may use [AppSurface.Command], [AppSurface.OnPreRun], and
// [AppSurface.SetLogger], but must not modify the root or commands they
// did not create. [fmt.Stringer] names the extension in errors.
type Extension interface {
	fmt.Stringer
	Init(AppSurface) error
}

// AsExtension registers an inline extension that runs fn during [New].
// Prefer [WithExtension] for packaged extensions. Nil fn returns [ErrNilOption].
func AsExtension(name string, fn func(AppSurface) error) Option {
	return optionFn(func(c *config) error {
		if fn == nil {
			return fmt.Errorf("%w: AsExtension(%q): fn cannot be nil", ErrNilOption, name)
		}
		c.extensions = append(c.extensions, inlineExtension{name: name, fn: fn})
		return nil
	})
}

type inlineExtension struct {
	name string
	fn   func(AppSurface) error
}

func (e inlineExtension) String() string { return e.name }

func (e inlineExtension) Init(a AppSurface) error { return e.fn(a) }

// ExtensionWithRequirements is an [Extension] that declares theme tokens
// it reads. [New] warns (or fails with [WithStrictThemeRequirements]) when
// the active theme is missing those tokens.
type ExtensionWithRequirements interface {
	Extension
	ThemeRequires() theme.Requirement
}

// WithExtension installs an [Extension] during [New], in declaration order.
// Pass the (Extension, error) pair from constructors such as manpage.New.
//
// Example:
//
//	New("myctl",
//	    WithExtension(manpage.New()),
//	    WithExtension(logging.New()),
//	)
//
// A non-nil constructor err is returned as-is. A nil ext returns
// [ErrNilOption]. [Extension.Init] failures surface from [New].
func WithExtension(ext Extension, err error) Option {
	return optionFn(func(c *config) error {
		if err != nil {
			return err
		}
		if ext == nil {
			return fmt.Errorf("%w: WithExtension: extension cannot be nil", ErrNilOption)
		}
		c.extensions = append(c.extensions, ext)
		return nil
	})
}

// SetLogger sets the logger returned by [Context.Logger]. Prefer [WithLogger]
// at construction. Nil resets to discard. Not safe for concurrent use; call
// before [App.Run] (extensions typically set it from [Extension.Init]).
func (a *App) SetLogger(l *slog.Logger) {
	a.cfg.logger = l
}

// UnsafeRoot returns the root [*cobra.Command] for escape-hatch use
// (completion generators, man-page traversal). Prefer [App.Command] and
// App-level options for standard work.
func (a *App) UnsafeRoot() *cobra.Command {
	return a.root
}

// Theme returns the immutable [theme.ResolvedTheme] from [WithTheme],
// [WithCustomTheme], or the built-in default. Query by [theme.Token] or
// accessors; do not read [App] internals.
func (a *App) Theme() theme.ResolvedTheme {
	return a.cfg.resolvedTheme
}

// Name returns the app's name (the value passed to [New]).
func (a *App) Name() string {
	return a.cfg.name
}

// EnvPrefix returns the App's configured env-variable prefix (set by
// [WithEnvPrefix]; defaults to UPPER(name)+"_").
func (a *App) EnvPrefix() string {
	return a.cfg.envPrefix
}

// IO returns the stdin/stdout/stderr bundle and terminal capability detection
// for this app. The result is never nil for a successfully constructed [App].
func (a *App) IO() *IOStreams {
	return a.io
}

// NewBareContext returns a [*Context] with the app's [IOStreams] and theme,
// but no resolved args, flags, or command. Use for themed helpers outside a
// [RunFunc]. In tests prefer [nabattest.Context].
func (a *App) NewBareContext() *Context {
	if a == nil {
		return nil
	}
	ctx := &Context{
		ctx:         context.Background(),
		app:         a,
		io:          a.io,
		values:      map[string]any{},
		set:         map[string]bool{},
		interactive: false,
	}
	if a.io != nil {
		ctx.interactive = a.io.IsInteractive()
	}
	if a.cfg.logger != nil {
		ctx.logger = a.cfg.logger
	}
	return ctx
}

// OnPreRun registers a global pre-run hook before every command handler
// (before command-level preRun). [ErrHandled] skips the handler and remaining
// hooks; [App.Run] returns nil. It returns [ErrNilOption] if fn is nil.
func (a *App) OnPreRun(fn func(*Context) error) error {
	if fn == nil {
		return fmt.Errorf("%w: OnPreRun: fn cannot be nil", ErrNilOption)
	}
	a.mu.Lock()
	a.globalPreRun = append(a.globalPreRun, fn)
	a.mu.Unlock()
	return nil
}

// New constructs an [App] with the given CLI name and options.
// The name is the root command word and the default env prefix
// (uppercased + "_") unless [WithEnvPrefix] overrides it.
//
// Example:
//
//	app, err := nabat.New("myctl",
//	    nabat.WithTheme(theme.Dracula),
//	    nabat.WithCommand("deploy", nabat.WithRun(deploy)),
//	)
//
// It may return [ErrNilOption], [*ConfigErrors] (empty name, nil IO, and
// similar), option/root config failures, flag registration errors, or
// extension Init errors.
func New(name string, opts ...Option) (*App, error) {
	cfg, cfgErr := defaultConfig()
	if cfgErr != nil {
		return nil, cfgErr
	}
	cfg.name = name
	cfg.envPrefix = strings.ToUpper(name) + "_"

	var configErrs ConfigErrors
	for i, opt := range opts {
		if opt == nil {
			configErrs.AddErr(fmt.Errorf("%w at index %d", ErrNilOption, i))
			continue
		}
		if err := opt.applyToConfig(cfg); err != nil {
			configErrs.AddErr(fmt.Errorf("nabat: option at index %d: %w", i, err))
		}
	}
	if err := cfg.validate(); err != nil {
		if cErrs, ok := errors.AsType[*ConfigErrors](err); ok {
			for _, issue := range cErrs.Unwrap() {
				configErrs.AddErr(issue)
			}
		} else {
			configErrs.AddErr(err)
		}
	}
	if configErrs.HasIssues() {
		return nil, &configErrs
	}

	// Resolve the theme recipe against the IO bundle's capabilities.
	// finalize MUST run after every option has been applied (so the right
	// resolver is in cfg.theme and the right IO is in cfg.io) and BEFORE any
	// command, extension, or pre-run hook can read Theme(). Storing the
	// result on cfg means every later reader (including tests that swap
	// IOStreams via WithIO at construction) sees the same immutable view.
	if err := cfg.finalize(); err != nil {
		return nil, err
	}

	rootSpec := cfg.rootSpec
	if err := validateUniqueNames(rootSpec, "root command"); err != nil {
		return nil, err
	}

	root := &cobra.Command{
		Use:           buildUseString(cfg.name, rootSpec.args, rootSpec.passthrough),
		Short:         rootSpec.description,
		Long:          rootSpec.longDescription,
		Aliases:       rootSpec.aliases,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	app := &App{
		cfg:        cfg,
		root:       root,
		meta:       make(map[*cobra.Command]*commandSpec),
		io:         cfg.io,
		outProfile: cfg.outProfile,
	}
	app.addMeta(root, rootSpec)

	root.SetOut(app.io.Out)
	root.SetErr(app.io.ErrOut)
	root.SetIn(app.io.In)

	// PersistentPreRunE on the root runs once for every invoked command and
	// is the single execution point for global pre-run hooks. It builds the
	// per-invocation [Context] and propagates it (along with any short-circuit
	// signal from a hook returning [ErrHandled]) via the Cobra command's context
	// so [App.attachRunE] can retrieve it without a separate sync.Map.
	root.PersistentPreRunE = func(cobraCmd *cobra.Command, args []string) error {
		nctx, err := app.newContext(cobraCmd, args)
		if err != nil {
			return err
		}
		state := &runState{ctx: nctx}
		for _, fn := range app.globalHooks() {
			if hookErr := fn(nctx); hookErr != nil {
				if errors.Is(hookErr, ErrHandled) {
					state.handled = true
					cobraCmd.SetContext(context.WithValue(cobraCmd.Context(), runStateKey{}, state))
					return nil
				}
				return hookErr
			}
		}
		cobraCmd.SetContext(context.WithValue(cobraCmd.Context(), runStateKey{}, state))
		return nil
	}

	if err := app.registerHelp(); err != nil {
		return nil, err
	}
	if err := app.registerVersion(); err != nil {
		return nil, err
	}
	if err := app.registerCompletion(); err != nil {
		return nil, err
	}

	for i := range rootSpec.flags {
		if regErr := registerFlagOnCommand(root, rootSpec.flags[i], cfg.envPrefix); regErr != nil {
			return nil, regErr
		}
	}
	if err := app.finalizeCommand(root, rootSpec); err != nil {
		return nil, err
	}

	rootWrapper := &Command{app: app, cobra: root, spec: rootSpec}
	for i, hook := range rootSpec.commandInitHooks {
		if hook == nil {
			return nil, fmt.Errorf("%w: root command: WithCommandInit/WithRootInit at index %d is nil", ErrNilOption, i)
		}
		if hookErr := hook(rootWrapper); hookErr != nil {
			return nil, fmt.Errorf("nabat: root command: WithCommandInit: %w", hookErr)
		}
	}

	for _, ext := range cfg.extensions {
		if err := ext.Init(app); err != nil {
			return nil, fmt.Errorf("nabat: extension %s: %w", ext, err)
		}
	}

	// Register declarative subcommands. Both the root spec's children (any
	// WithCommand passed inside another RootOption position) and pending
	// top-level commands (WithCommand passed directly to New) flatten under
	// the root. Errors aggregate into one *ConfigErrors so users see every
	// problem at once.
	if err := app.registerPending(rootSpec.children); err != nil {
		return nil, err
	}
	if err := app.registerPending(cfg.pendingCommands); err != nil {
		return nil, err
	}

	if err := app.validate(); err != nil {
		return nil, err
	}

	return app, nil
}

// registerPending walks a list of pending subcommand registrations, registers
// each under the App's root, and recursively registers their nested children.
// All registration errors aggregate into one [*ConfigErrors] return value.
func (a *App) registerPending(regs []*commandReg) error {
	if len(regs) == 0 {
		return nil
	}
	var errs ConfigErrors
	a.registerPendingUnder(a.root, regs, &errs)
	if errs.HasIssues() {
		return &errs
	}
	return nil
}

// registerPendingUnder is the recursive worker for [App.registerPending]. It
// registers each entry of regs as a child of parent and recurses into the
// new command's own children. Aggregates failures into errs without
// short-circuiting so callers see every problem in one report.
func (a *App) registerPendingUnder(parent *cobra.Command, regs []*commandReg, errs *ConfigErrors) {
	for _, reg := range regs {
		cmd, err := a.newCommand(parent, reg.name, reg.opts...)
		if err != nil {
			errs.AddErr(err)
			continue
		}
		a.registerPendingUnder(cmd.cobra, cmd.spec.children, errs)
	}
}

func (a *App) validate() error {
	var errs ConfigErrors
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		children := cmd.Commands()
		if len(children) == 0 {
			return
		}
		seenNames := map[string]bool{}
		for _, child := range children {
			if seenNames[child.Name()] {
				errs.AddErr(fmt.Errorf("nabat: command %q: duplicate child command name %q", cmd.Name(), child.Name()))
			}
			seenNames[child.Name()] = true
		}

		shortSeen := map[string]string{}
		cmd.PersistentFlags().VisitAll(func(fl *pflag.Flag) {
			if fl.Shorthand == "" {
				return
			}
			if prev, ok := shortSeen[fl.Shorthand]; ok && prev != fl.Name {
				errs.AddErr(fmt.Errorf("nabat: command %q: persistent shorthand -%s conflicts between --%s and --%s", cmd.Name(), fl.Shorthand, prev, fl.Name))
			} else {
				shortSeen[fl.Shorthand] = fl.Name
			}
		})

		for _, child := range children {
			walk(child)
		}
	}
	walk(a.root)

	if errs.HasIssues() {
		return &errs
	}
	return nil
}

// validateUniqueNames reports an error when args and flags share a name in the
// given spec. commandPath is used in the error message ("root command" or
// `command "name"`).
func validateUniqueNames(spec *commandSpec, commandPath string) error {
	names := make(map[string]string, len(spec.args)+len(spec.flags))
	for _, in := range spec.args {
		names[in.name] = "arg"
	}
	for _, fl := range spec.flags {
		if prev, exists := names[fl.name]; exists {
			return fmt.Errorf("nabat: %s: name %q is used by both %s and flag: %w", commandPath, fl.name, prev, ErrArgFlagNameCollision)
		}
		names[fl.name] = "flag"
	}
	return nil
}

// renderMarkdown renders markdown with glamour on a TTY, or returns
// content unchanged otherwise. Style order: notty, [theme.ResolvedTheme.Glamour],
// then the "dark" preset as a last resort for an empty ResolvedTheme.
func (a *App) renderMarkdown(content string) string {
	if content == "" {
		return ""
	}

	rt := a.cfg.resolvedTheme

	var styleOpt glamour.TermRendererOption
	switch {
	case !a.io.IsStdoutTTY():
		styleOpt = glamour.WithStylePath("notty")
	default:
		if cfg := rt.Glamour(); cfg != nil {
			styleOpt = glamour.WithStyles(*cfg)
		} else {
			styleOpt = glamour.WithStylePath("dark")
		}
	}

	r, err := glamour.NewTermRenderer(
		styleOpt,
		glamour.WithWordWrap(a.io.TerminalWidth()),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return out
}

// MustNew is like [New] but panics on construction failure.
// Use [New] when errors should propagate (tests and libraries).
//
// Panics if name is empty, any [Option] is nil, or any [WithCommand] /
// extension registration aggregated by [New] fails.
func MustNew(name string, opts ...Option) *App {
	a, err := New(name, opts...)
	if err != nil {
		panic(fmt.Errorf("nabat initialization failed: %w", err))
	}
	return a
}

// Command registers a subcommand under the app root.
// On error, nothing is registered and the returned [*Command] is nil.
// Prefer [WithCommand] inside [New] to aggregate many registration errors;
// use [App.MustCommand] for panicking chains in main or tests.
// It may return [ErrRegistrationFrozen], [ErrNilOption],
// [ErrArgFlagNameCollision], or an error for an empty name or failed
// option/flag finalization.
func (a *App) Command(name string, opts ...CommandOption) (*Command, error) {
	if a.registrationFrozen.Load() {
		return nil, ErrRegistrationFrozen
	}
	return a.newCommand(a.root, name, opts...)
}

// MustCommand is like [App.Command] but panics on failure.
// Use in main or tests for chaining; a failure is a programmer bug.
func (a *App) MustCommand(name string, opts ...CommandOption) *Command {
	c, err := a.Command(name, opts...)
	if err != nil {
		panic(fmt.Errorf("nabat: %w", err))
	}
	return c
}

// Run parses [os.Args] and executes the matching command. ctx is the base
// [context.Context] for each [RunFunc]. On failure it prints a styled error
// and usage hint to stderr unless [WithErrorHandler] replaced that.
// Failures include Cobra parse/usage errors and errors from arg/flag
// resolution, validation, or the command handler.
func (a *App) Run(ctx context.Context) error {
	a.registrationFrozen.Store(true)
	cmd, err := a.root.ExecuteContextC(ctx)
	if err != nil {
		if a.cfg.errorHandler != nil {
			a.cfg.errorHandler(err)
		} else {
			printCmd := cmd
			if printCmd == nil {
				printCmd = a.root
			}
			a.printError(printCmd, err)
		}
	}
	return err
}

// RunArgs is like [App.Run] but takes args instead of [os.Args].
// In tests prefer [nabattest]. Errors and stderr behavior match [App.Run].
func (a *App) RunArgs(ctx context.Context, args ...string) error {
	a.root.SetArgs(args)
	defer a.root.SetArgs(nil)
	return a.Run(ctx)
}

func (a *App) printError(cmd *cobra.Command, err error) {
	if err == nil {
		return
	}
	out := &writer{w: a.io.ErrOut}
	errStyle := a.cfg.resolvedTheme.Style(theme.StatusError)
	msg := formatUserError(err)
	rt := a.cfg.resolvedTheme
	muted := rt.Style(theme.TextMuted)
	accent := rt.Style(theme.AccentPrimary)
	out.printf("%s %s\n", errStyle.Render("error:"), msg)
	out.printf("%s %s %s\n",
		muted.Render("Run"),
		accent.Render(cmd.CommandPath()+" --help"),
		muted.Render("for usage."),
	)
}

// formatUserError renders err for stderr. It strips a leading "nabat: "
// package prefix from the outermost message when present, without walking
// wrapped causes (those stay in the formatted string for ConfigErrors and
// similar multi-issue values).
func formatUserError(err error) string {
	msg := err.Error()
	const prefix = "nabat: "
	if after, ok := strings.CutPrefix(msg, prefix); ok {
		return after
	}
	return msg
}

// runStateKey is the context key for the per-invocation [runState].
type runStateKey struct{}

// runState carries the per-invocation [Context] and the "global hook handled
// the call" flag from the root [cobra.Command.PersistentPreRunE] down to
// every command's [cobra.Command.RunE]. It travels via the Cobra command's
// own [context.Context] so there is no separate map and no cleanup step.
type runState struct {
	ctx     *Context
	handled bool
}

// attachRunE wires cmd.RunE to the per-command pipeline (hooks, validation,
// run, post-run). Global pre-run runs in root PersistentPreRunE, which also
// builds the [Context]. [ErrHandled] from a global hook skips this pipeline;
// commands with no handler or hooks print help.
func (a *App) attachRunE(cmd *cobra.Command, spec *commandSpec) {
	hasSpecHooks := len(spec.preRun) > 0 || len(spec.validations) > 0 || len(spec.postRun) > 0
	cmd.RunE = func(cobraCmd *cobra.Command, args []string) error {
		state, ok := cobraCmd.Context().Value(runStateKey{}).(*runState)
		if !ok || state == nil {
			// Defensive: unreachable in a normally-built App because every
			// command tree is rooted at a.root whose PersistentPreRunE always
			// fires first and sets the state.
			ctx, err := a.newContext(cobraCmd, args)
			if err != nil {
				return err
			}
			state = &runState{ctx: ctx}
		}
		if state.handled {
			return nil
		}
		ctx := state.ctx

		// Discovery UX: command with no run and no spec hooks falls back to
		// help once global hooks have had their chance to short-circuit.
		if spec.run == nil && !hasSpecHooks {
			return cobraCmd.Help()
		}
		for _, fn := range spec.preRun {
			if hookErr := fn(ctx); hookErr != nil {
				if errors.Is(hookErr, ErrHandled) {
					return nil
				}
				return hookErr
			}
		}
		for _, fn := range spec.validations {
			if validateErr := fn(ctx); validateErr != nil {
				return validateErr
			}
		}
		var runErr error
		if spec.run != nil {
			runErr = spec.run(ctx)
		}
		for _, fn := range spec.postRun {
			if postErr := fn(ctx); postErr != nil && runErr == nil {
				runErr = postErr
			}
		}
		return runErr
	}
}
