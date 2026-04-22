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

import "fmt"

// Help is a built-in core feature with a two-axis design:
//
//   - The persistent `--help` (`-h`) flag is on by default for every [App]
//     constructed with [New] or [MustNew]. This matches the GNU/POSIX
//     convention that `--help` should always be available, and parallels
//     other CLI frameworks (cobra, urfave/cli, clap). Configure or disable
//     the flag with [WithHelpFlagName], [WithHelpShorthand],
//     [WithoutHelpFlag], and [WithoutHelpShorthand].
//
//   - The `help <subcmd>` subcommand is opt-in via [WithHelpCommand],
//     mirroring how [WithVersion] opts into the version feature. Omit
//     [WithHelpCommand] and no subcommand is installed; pass it to install
//     the Nabat-themed `help` subcommand and customize its name with
//     [WithHelpCommandName].
//
// Use [WithoutHelp] to disable the entire feature (no flag, no subcommand,
// no custom renderer); Cobra's stock defaults take over.
//
// Defaults: flag "--help", shorthand "-h", subcommand absent. A bare
// `nabat.MustNew("foo")` accepts `foo --help` and `foo -h` but not
// `foo help <subcmd>` until [WithHelpCommand] is added.

// HelpCommandOption configures the opt-in `help <subcmd>` surface inside
// [WithHelpCommand].
type HelpCommandOption func(*helpCommandConfig) error

type helpConfig struct {
	flag    helpFlagState
	command *helpCommandConfig

	disabled        bool
	disabledTouched bool
	otherTouched    bool
}

type helpFlagState struct {
	name              string
	shorthand         rune
	shorthandSet      bool
	shorthandDisabled bool
}

type helpCommandConfig struct {
	name string
}

func defaultHelpConfig() *helpConfig {
	return &helpConfig{
		flag: helpFlagState{name: "help", shorthand: 'h'},
	}
}

func (hc *helpConfig) validate() error {
	if hc.disabledTouched && hc.otherTouched {
		return fmt.Errorf("%w: WithoutHelp cannot be combined with any other With(out)Help* option", ErrInvalidOption)
	}
	if hc.flag.shorthandDisabled && hc.flag.shorthandSet {
		return fmt.Errorf("%w: WithoutHelpShorthand cannot be combined with WithHelpShorthand", ErrInvalidOption)
	}
	return nil
}

// WithHelpCommand installs the `help <subcmd>` subcommand with Nabat's
// themed renderer. The subcommand is off by default; pass this option to
// turn it on, mirroring how [WithVersion] opts into the version feature.
//
// Pass [HelpCommandOption] values such as [WithHelpCommandName] to
// customize the subcommand. A nil option in the slice returns
// [ErrNilOption].
//
// Example:
//
//	MustNew("ctl", WithHelpCommand())
//	// `ctl help run` prints help for the `run` subcommand.
//
//	MustNew("ctl", WithHelpCommand(
//	    WithHelpCommandName("aide"),
//	))
//	// `ctl aide run` prints help for the `run` subcommand.
func WithHelpCommand(opts ...HelpCommandOption) Option {
	return optionFn(func(c *config) error {
		cmdCfg := &helpCommandConfig{name: "help"}
		for i, opt := range opts {
			if opt == nil {
				return fmt.Errorf("%w at WithHelpCommand option index %d", ErrNilOption, i)
			}
			if err := opt(cmdCfg); err != nil {
				return fmt.Errorf("nabat: WithHelpCommand: %w", err)
			}
		}
		c.help.command = cmdCfg
		c.help.otherTouched = true
		return nil
	})
}

// WithHelpCommandName overrides the help subcommand name (default "help").
// Use inside [WithHelpCommand]; empty strings return [ErrInvalidOption].
//
// Example:
//
//	app := MustNew("myctl", WithHelpCommand(
//	    WithHelpCommandName("aide"),
//	))
//	// `myctl aide deploy` prints help for the deploy subcommand.
func WithHelpCommandName(name string) HelpCommandOption {
	return func(cc *helpCommandConfig) error {
		if name == "" {
			return fmt.Errorf("%w: WithHelpCommandName: name cannot be empty", ErrInvalidOption)
		}
		cc.name = name
		return nil
	}
}

// WithHelpFlagName overrides the help flag name (default "help"). Combine
// with [WithHelpShorthand] to also change the shorthand.
//
// Empty string returns [ErrInvalidOption]; use [WithoutHelpFlag] to disable
// the flag instead.
//
// When the flag name is not "help", a hidden alias --help is also registered
// to preempt Cobra's auto-injected --help and avoid two help flags coexisting.
//
// Example:
//
//	app := MustNew("myctl",
//	    WithHelpFlagName("info"),
//	    WithHelpShorthand('i'),
//	)
//	// `myctl --info` and `myctl -i` show help.
func WithHelpFlagName(name string) Option {
	return optionFn(func(c *config) error {
		if name == "" {
			return fmt.Errorf("%w: WithHelpFlagName: name cannot be empty (use WithoutHelpFlag to disable)", ErrInvalidOption)
		}
		c.help.flag.name = name
		c.help.otherTouched = true
		return nil
	})
}

// WithoutHelpFlag disables the persistent --help flag. The opt-in help
// subcommand (when enabled via [WithHelpCommand]) keeps working.
//
// Cobra's auto-injected --help is suppressed by registering a hidden no-op
// flag in its place; if the user explicitly wants Cobra's default behavior,
// use [WithoutHelp] instead.
func WithoutHelpFlag() Option {
	return optionFn(func(c *config) error {
		c.help.flag.name = ""
		c.help.flag.shorthand = 0
		c.help.otherTouched = true
		return nil
	})
}

// WithHelpShorthand sets the one-character shorthand for the help flag
// (default 'h').
func WithHelpShorthand(r rune) Option {
	return optionFn(func(c *config) error {
		c.help.flag.shorthand = r
		c.help.flag.shorthandSet = true
		c.help.otherTouched = true
		return nil
	})
}

// WithoutHelpShorthand disables the help flag's shorthand. The long form
// (default --help) keeps working.
func WithoutHelpShorthand() Option {
	return optionFn(func(c *config) error {
		c.help.flag.shorthand = 0
		c.help.flag.shorthandDisabled = true
		c.help.otherTouched = true
		return nil
	})
}

// WithoutHelp disables the built-in help feature entirely: no Nabat custom
// renderer is installed, no opt-in `help` subcommand, no persistent
// `--help` flag. Cobra's defaults take over (auto-injected --help with the
// stock template).
//
// Mixing WithoutHelp with any other With(out)Help* option returns
// [ErrInvalidOption]. The check runs in [New] after all options have been
// applied so order does not matter.
func WithoutHelp() Option {
	return optionFn(func(c *config) error {
		c.help.disabled = true
		c.help.disabledTouched = true
		return nil
	})
}
