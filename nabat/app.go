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
	"strings"
	"sync"
	"sync/atomic"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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
	// without an extra union type. nil means no theme — finalize uses
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

	io *IOStreams

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
	return &config{
		theme:    t,
		io:       NewSystemIO(),
		help:     defaultHelpConfig(),
		rootSpec: &commandSpec{},
	}, nil
}

// finalize resolves the configured theme [Resolver] against the IO
// bundle's detected [theme.Capabilities] and stores the resulting
// [theme.ResolvedTheme] on the config. It runs once during [New],
// after option application, before any command or extension can
// observe Theme(). Calling finalize after construction is a no-op
// aside from re-detecting capabilities, which never change for the
// lifetime of an [App].
//
// Errors:
//   - Per-palette [theme.Palette.GlamourFor] callbacks may fail when
//     the inline manifest glamour block produces an invalid config
//     against the live capabilities. The error is surfaced through
//     [theme.Theme.ResolveErr] when the resolver is a [theme.Theme],
//     wrapped with the theme name. Bespoke [theme.Resolver] implementations
//     have no error channel; their Resolve method must self-handle.
func (c *config) finalize() error {
	caps := detectCapabilities(c.io)

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
			// silently — the user opted into a custom resolver and
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

// checkThemeRequirements runs the requirement validation declared by
// core consumers and by every installed [ExtensionWithRequirements].
// In strict mode the result is returned as a hard error so [New]
// fails closed; otherwise the diagnostic is rendered to
// [IOStreams.ErrOut] and construction continues.
//
// The split between warn and error matches the plan: opinionated
// minimal themes that intentionally drop slots should still install
// cleanly by default; tests and CLIs that want the regression catch
// opt in via [WithStrictThemeRequirements].
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
		if c.io != nil && c.io.ErrOut != nil {
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

// Option configures an [App] during construction with [New] or [MustNew].
//
// Three categories of values satisfy Option:
//
//   - App-level options like [WithTheme], [WithIO], [WithLogger],
//     [WithExtension], [WithVersion], the With(out)Help* family.
//   - [RootOption] values, which configure the root command directly
//     ([WithDescription], [WithRun], [WithFlag], [WithArg], etc.).
//   - [WithCommand], which registers a subcommand under the root.
//
// Passing a [CommandOption] that is not a [RootOption] (such as [WithGroup],
// [WithHidden], [WithAliases], [WithTypoHints]) to [New] is a build error:
// those options have no meaningful effect on the root command. They remain
// valid inside [WithCommand] for nested subcommands.
//
// Passing a nil [Option] to [New] returns [ErrNilOption].
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
// order.
//
// Errors:
//   - [ErrNilOption]: a nil entry appears in opts
//   - errors from individual options
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

// WithTheme installs the named built-in theme from the embedded catalog
// (see [nabat.dev/theme] for the available names). Use the untyped
// string constants in [theme] for IDE autocomplete and a build-time
// guarantee that the spelling matches an embedded manifest:
//
//	New("myctl", WithTheme(theme.Dracula))
//
// Plain strings work too — pass a flag value, env var, or user-supplied
// configuration directly:
//
//	name := os.Getenv("MYCTL_THEME")
//	if name == "" { name = theme.Default }
//	New("myctl", WithTheme(name))
//
// [WithTheme] and [WithCustomTheme] compose: the last one wins.
// Multiple calls do not error; the override slot
// [WithThemeOverride] handles per-token tweaks without forcing the
// caller to construct a derived theme by hand.
//
// Errors:
//   - The theme name is not in the embedded catalog. The error joins
//     into the [*ConfigErrors] returned by [New] and lists every
//     available name so the caller (or end user) can correct the
//     typo.
//
// For a programmatic theme that does not live as a manifest, use
// [WithCustomTheme].
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

// WithCustomTheme installs a programmatically-defined [theme.Resolver].
// Use this when a theme cannot be expressed as a JSON manifest — for
// example, when it ships an owned [chroma.Style] or a closure-based
// [huh.Theme] that depends on values not available at manifest-load
// time. The common case (a fully-declared [theme.Theme] struct) just
// passes the value directly because [theme.Theme] satisfies
// [theme.Resolver]:
//
//	acme := theme.Theme{
//	    Name:    "acme",
//	    Default: theme.VariantDark,
//	    Variants: map[theme.Variant]theme.Palette{
//	        theme.VariantDark: {
//	            Tokens:  acmeTokens,
//	            Chroma:  acmeChromaStyle,
//	            Glamour: acmeGlamourStyle,
//	            Huh:     acmeHuhTheme,
//	        },
//	    },
//	}
//	app, _ := nabat.New("myctl", nabat.WithCustomTheme(acme))
//
// The rare case (a Resolver whose palette choice depends on runtime
// [theme.Capabilities] in a way one Variant per Palette cannot
// express) implements [theme.Resolver] directly and passes that.
//
// [WithCustomTheme] composes with [WithTheme]: the last call wins.
// Per-token tweaks should reach for [WithThemeOverride] instead of
// constructing a derived [theme.Theme] for one slot.
//
// Errors:
//   - [ErrNilOption]: r is nil.
func WithCustomTheme(r theme.Resolver) Option {
	return optionFn(func(c *config) error {
		if r == nil {
			return fmt.Errorf("%w: WithCustomTheme resolver cannot be nil", ErrNilOption)
		}
		c.theme = r
		return nil
	})
}

// WithThemeOverride registers a per-token style override applied on
// top of the active theme at finalize time. Multiple calls compose:
// later overrides for the same token win, overrides on different
// tokens stack.
//
// The common use case is "use the bundled theme, but with my brand
// color":
//
//	app, _ := nabat.New("myctl",
//	    nabat.WithTheme(theme.Dracula),
//	    nabat.WithThemeOverride(theme.StatusError, magenta),
//	)
//
// Overrides apply to every variant of the underlying theme, so a
// multi-variant manifest stays multi-variant after the override —
// the same one-line tweak affects whichever variant the runtime
// capabilities pick.
//
// Overrides are silently ignored when the active theme is a bespoke
// [theme.Recipe] (anything other than a [theme.Theme] value); the
// recipe's Resolve method is opaque, so the framework cannot
// meaningfully apply per-Palette overrides into it.
func WithThemeOverride(t theme.Token, s lipgloss.Style) Option {
	return optionFn(func(c *config) error {
		c.themeOverrides = append(c.themeOverrides, theme.SetToken(t, s))
		return nil
	})
}

// WithThemeOverrides registers a batch of [theme.Override] mutations
// — useful for callers that want to apply more than one slot in one
// call (chroma swap plus a token tweak, for example) or that want
// to drop in an override produced by another helper such as
// [theme.SetAlias] or [theme.SetChroma].
//
// Overrides apply in slice order; the same composition rules
// described on [WithThemeOverride] apply.
//
// Errors:
//   - [ErrNilOption]: one of the overrides entries is nil.
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

// WithStrictThemeRequirements promotes the framework's
// theme-requirement warning into a hard error returned from [New].
// Without this option, the App writes a multi-line diagnostic to
// [IOStreams.ErrOut] when an installed theme misses
// tokens declared by core consumers or by an
// [ExtensionWithRequirements]; with it, the same diagnostic
// becomes a [*ConfigErrors] entry and construction fails.
//
// Use case: tests that assert "this theme covers everything we
// need" — flip the option and let the framework's check do the
// regression catching for you. Production CLIs typically leave it
// off so an opinionated minimal theme can still ship without
// breaking apps that picked it.
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

// WithLogger sets the structured logger returned by [Context.Logger] during
// command invocations. Pass any [*slog.Logger]; Nabat treats it as opaque.
//
// Use this when you want to bring your own logger. As an alternative, install
// the logging plugin via [WithExtension] for an opinionated styled charm logger
// with --verbose / --log-level flag wiring.
//
// When neither WithLogger nor the logging plugin is used, [Context.Logger]
// returns a discard logger that silently drops all records.
//
// Passing a nil [*slog.Logger] returns [ErrNilOption].
func WithLogger(l *slog.Logger) Option {
	return optionFn(func(c *config) error {
		if l == nil {
			return fmt.Errorf("%w: WithLogger logger cannot be nil", ErrNilOption)
		}
		c.logger = l
		return nil
	})
}

// App is the root CLI application for a single binary.
//
// It owns the underlying Cobra root command, global configuration, installed
// extensions (see [WithExtension]), and the cached color profile used for output.
//
// Construct an App with [New] or [MustNew]. Pass [Option] values to configure
// it: app-level options ([WithTheme], [WithIO], [WithLogger], [WithExtension],
// [WithVersion], the With(out)Help* family), [RootOption] values to configure
// the root command directly ([WithDescription], [WithFlag], [WithRun], etc.),
// and [WithCommand] to declare subcommands. The entire command tree can be
// declared in one call to [New]; every registration error aggregates into the
// returned error. For runtime/dynamic registration, use [App.Command] (returns
// `(*Command, error)`) or [App.MustCommand] (panics on failure). Run the
// program with [App.Run].
//
// An App is safe to use from multiple goroutines only AFTER registration is
// complete. All command and flag registration — including extensions calling
// [App.Command] or [App.MustCommand] from inside [Extension.Init] — must
// happen before [App.Run] (or [App.RunArgs]) is invoked, and before any
// goroutine forks. Concurrent registration is not supported. Calls to
// [App.Command] / [Command.Command] after [App.Run] return an error;
// [App.MustCommand] / [Command.MustCommand] panic with the same message.
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
	// (`app.IO().Out`, `app.IO().IsStdoutTTY()`, `app.IO().ColorEnabled()`) instead
	// of touching os.Stdin/Stdout/Stderr directly. Override at construction
	// with [WithIO].
	io *IOStreams

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

// Extension contributes subcommands, hooks, or a logger to an [App] during
// construction. Extensions are installed via [WithExtension] and their [Init]
// method runs inside [New] after the root command and core features (help,
// version, completion) are wired up.
//
// First-party extensions live under nabat.dev subpackages (manpage, logging)
// and are constructed by their package's New() function, which returns
// (Extension, error). Third-party extensions follow the same pattern.
//
// # Invariant
//
// Extensions install subcommands (via [AppSurface.Command]) and global hooks
// (via [AppSurface.OnPreRun]), and may set the logger (via
// [AppSurface.SetLogger]). They MUST NOT modify the root command or any
// command they did not create.
//
// [fmt.Stringer] is required so error messages can identify the extension by
// name.
type Extension interface {
	fmt.Stringer
	Init(AppSurface) error
}

// AsExtension returns an [Option] that registers an inline extension.
// fn runs in declaration order with other extensions, after help, version, and
// completion are registered on the root command.
//
// This is the inline alternative to implementing the [Extension] interface. For
// third-party extensions built as separate packages, use [WithExtension] with
// the package's New() constructor instead.
//
// Errors:
//   - [ErrNilOption]: fn is nil.
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

// ExtensionWithRequirements is an optional sub-interface for
// extensions that read tokens from [theme.ResolvedTheme]. The
// framework collects the [theme.Requirement] every implementing
// extension declares and validates it against the active resolved
// theme at App.finalize time.
//
// Extensions that don't read tokens (a man-page generator that only
// writes roff, for example) leave this method off. Extensions that
// do read tokens implement it so the App can surface "your theme is
// missing the slots my extension needs" diagnostics before the user
// hits the unstyled output:
//
//	func (e *Extension) ThemeRequires() theme.Requirement {
//	    return theme.Require("logging extension",
//	        theme.StatusInfo, theme.StatusWarning, theme.StatusError,
//	        theme.AccentPrimary, theme.TextPrimary,
//	    )
//	}
//
// The check is warn-by-default (writes to [IOStreams.ErrOut]
// at App construction time) so an opinionated minimal theme that
// intentionally omits some tokens does not break apps. Pass
// [WithStrictThemeRequirements] to promote the warning into a
// [*ConfigErrors] entry that blocks construction.
type ExtensionWithRequirements interface {
	Extension
	ThemeRequires() theme.Requirement
}

// WithExtension installs an [Extension]. Extensions run in declaration order
// inside [New], after the root command, help, version, and completion are
// configured. Later extensions can observe state set by earlier ones.
//
// WithExtension accepts the (Extension, error) pair returned by extension
// constructors (e.g. manpage.New(), logging.New()) so the construction error
// flows directly into New's [*ConfigErrors] without a separate error check:
//
//	New("myctl",
//	    WithExtension(manpage.New()),
//	    WithExtension(logging.New(logging.WithVerboseFlag("verbose"))),
//	)
//
// Errors:
//   - err is non-nil: the constructor error is forwarded as-is.
//   - ext is nil (and err is nil): [ErrNilOption].
//   - errors from [Extension.Init] cause [New] to return immediately.
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

// SetLogger installs the structured logger returned by [Context.Logger] during
// command invocations. This is the plugin-facing setter; user code should
// generally use [WithLogger] at construction time instead.
//
// Calling SetLogger after [WithLogger] replaces the previously installed
// logger.
// Passing nil resets to the discard logger returned by [Context.Logger] when no
// logger is configured.
//
// SetLogger is not safe for concurrent use and must be called before [App.Run].
// Extensions install their logger from [Extension.Init], which runs inside
// [New] before any command can execute.
func (a *App) SetLogger(l *slog.Logger) {
	a.cfg.logger = l
}

// UnsafeRoot returns the underlying [*cobra.Command] for the App's root.
// This is an escape hatch for operations that Nabat does not abstract
// (e.g. Cobra completion generators, command tree traversal for man page
// rendering). Prefer [App.Command], [App.MustCommand], and the App-level
// options ([WithVersion], the With(out)Help* family) for standard
// use.
func (a *App) UnsafeRoot() *cobra.Command {
	return a.root
}

// Theme returns the resolved theme installed via [WithTheme] or
// [WithCustomTheme] (or the built-in default if neither was used). The
// returned [theme.ResolvedTheme] is immutable; consumers query it by
// [theme.Token] for [lipgloss.Style] values, or by accessor for the
// chroma name, glamour name, [huh.Theme], list enumerator, and table
// border. Extensions read from this same value via the public method —
// they should never reach into [App.cfg].
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

// OnPreRun registers a global pre-run hook that fires before every command's
// handler, in registration order. Global hooks run before the command's own
// preRun hooks. The framework manages ordering; extensions never need to
// chain hooks manually.
//
// If the hook returns [ErrHandled], the command handler and remaining hooks
// are skipped and [App.Run] returns nil (success).
//
// Returns [ErrNilOption] if fn is nil.
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
//
// The name becomes the root command's primary word and the default environment
// variable prefix (uppercased with a trailing underscore) unless [WithEnvPrefix]
// overrides it.
//
// Construction order is fixed: options are applied to the config (including
// [RootOption] values that configure the root command's spec directly), the
// theme is resolved, the root command is built, help, version, and completion
// are installed, root flags are registered on Cobra, and finally [Extension.Init]
// runs for every [WithExtension] in declaration order. Errors from any phase
// short-circuit and are returned.
//
// Errors:
//   - [ErrNilOption]: a nil entry appears in opts
//   - wrapped errors from any [Option] or from root command configuration
//   - "nabat: app name cannot be empty", stdin/stdout/stderr cannot be nil, or
//     other messages aggregated in a *[ConfigErrors] value
//   - errors from registering flags or validating flag definitions
//   - "nabat: extension <name>: ..." when an [Extension.Init] returns an error
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
		var cErrs *ConfigErrors
		if errors.As(err, &cErrs) {
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
		cfg:  cfg,
		root: root,
		meta: make(map[*cobra.Command]*commandSpec),
		io:   cfg.io,
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

// renderMarkdown renders content as markdown using glamour in TTY mode,
// or returns the raw content when output is not a terminal.
//
// Style selection precedes glamour.NewTermRenderer:
//
//  1. Output is not a terminal -> "notty" preset (plain text).
//  2. [theme.ResolvedTheme.Glamour] returns a *ansi.StyleConfig
//     (already folded by [theme.Theme.Resolve] from the owned-style /
//     named-preset / capability-default cascade) -> WithStyles uses it
//     directly.
//  3. Otherwise -> the "dark" literal here is the last-resort branch
//     for code that constructs an empty ResolvedTheme directly. In a
//     normally-built App, [theme.Theme.Resolve] always populates the
//     glamour slot to at least the variant default, so this branch is
//     unreachable.
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

// Command creates a direct subcommand under the app's root command.
//
// Returns the new command and a non-nil error if registration fails (empty
// name, nil [CommandOption], invalid options, name collisions,
// flag-registration
// errors, etc.). On error, no command is registered and the returned [*Command]
// is nil — callers must handle the error before chaining further registrations.
//
// For aggregated registration errors (multiple commands surfaced together with
// option errors and validation errors), declare commands with [WithCommand]
// inside [New] instead. For panicking chains in main() or test setup, use
// [App.MustCommand].
//
// Errors:
//   - [ErrRegistrationFrozen]: called after [App.Run] / [App.RunArgs]
//   - [ErrNilOption]: a CommandOption in opts is nil
//   - "nabat: command name cannot be empty"
//   - [ErrArgFlagNameCollision]: an arg and a flag share a name
//   - errors from individual [CommandOption] application (wrapped, joined with
//     [ErrInvalidOption] when triggered by a nil option-list entry)
//   - errors from flag registration and command finalization
func (a *App) Command(name string, opts ...CommandOption) (*Command, error) {
	if a.registrationFrozen.Load() {
		return nil, ErrRegistrationFrozen
	}
	return a.newCommand(a.root, name, opts...)
}

// MustCommand is like [App.Command] but panics on registration failure.
//
// Use MustCommand in main() and test setup where chaining
// (`app.MustCommand("cluster").MustCommand("scale", ...)`) is more readable
// than scoped error checks. A registration error here is a programmer bug,
// not runtime input. Mirrors the relationship between [MustNew] and [New].
//
// Panics if [App.Command] returns an error. See its documentation for the
// full per-error list.
func (a *App) MustCommand(name string, opts ...CommandOption) *Command {
	c, err := a.Command(name, opts...)
	if err != nil {
		panic(fmt.Errorf("nabat: %w", err))
	}
	return c
}

// Run parses [os.Args] and executes the matching command using ctx as the base
// [context.Context] for each [Context] passed to [RunFunc].
//
// On failure (except when help or version output was handled internally), Run
// prints a styled error message and a usage hint to stderr before returning the
// error, unless [WithErrorHandler] replaced that behavior.
//
// Errors:
//   - errors returned by Cobra for unknown commands, usage, or flag parsing
//   - errors from resolving positional args or flags, validation hooks, or the
//     command handler
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

// RunArgs parses args as the command line instead of [os.Args], then executes
// the matching command using ctx as the base [context.Context].
//
// This is useful for tests, embedded CLIs, and programs that construct their
// own argument lists at runtime. In tests prefer the [nabattest] package which
// wraps
// this method with [testing.TB] helpers.
//
// Errors and stderr behavior match those described on [App.Run].
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
	// Strip the "nabat: " package prefix — it is useful for programmatic
	// callers but noise for end users reading stderr.
	msg := strings.TrimPrefix(err.Error(), "nabat: ")
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

// attachRunE wires cmd.RunE to Nabat's per-command handler pipeline (spec
// hooks, validation, run, post-run).
//
// Global pre-run hooks are NOT executed here; they fire exactly once per
// invocation in the root's [cobra.Command.PersistentPreRunE] (set in [New]),
// which also builds the [Context] this RunE consumes via the Cobra command's
// context. Consequences:
//
//   - attachRunE is called once at command registration time and never again;
//     extensions that add global hooks after some commands are registered are
//     still picked up because PersistentPreRunE reads [App.globalHooks] at
//     invocation time.
//   - When a global hook returns [ErrHandled], PersistentPreRunE marks the
//     run as handled and RunE short-circuits without invoking spec hooks.
//   - When the command has no command-level handler AND no command-level
//     hooks, RunE falls back to printing help — preserving the discovery UX
//     for subcommands that exist only to group children.
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
