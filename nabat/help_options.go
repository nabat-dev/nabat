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

// HelpCommandOption configures the opt-in `help <subcmd>` surface inside
// [WithHelpCommand]. The persistent `--help`/`-h` flag is on by default;
// the subcommand is opt-in. Use [WithoutHelp] to disable the entire feature.
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

// WithHelpCommand installs the `help <subcmd>` subcommand with Nabat's themed
// renderer. Pass [HelpCommandOption] values such as [WithHelpCommandName] to
// customize. A nil option returns [ErrNilOption].
//
// Example:
//
//	MustNew("ctl", WithHelpCommand(WithHelpCommandName("aide")))
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
func WithHelpCommandName(name string) HelpCommandOption {
	return func(cc *helpCommandConfig) error {
		if name == "" {
			return fmt.Errorf("%w: WithHelpCommandName: name cannot be empty", ErrInvalidOption)
		}
		cc.name = name
		return nil
	}
}

// WithHelpFlagName overrides the help flag name (default "help"). Empty
// string returns [ErrInvalidOption]; use [WithoutHelpFlag] to disable. When
// the name is not "help", a hidden `--help` alias preempts Cobra's auto flag.
//
// Example:
//
//	MustNew("myctl", WithHelpFlagName("info"), WithHelpShorthand('i'))
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
// subcommand (via [WithHelpCommand]) keeps working. Use [WithoutHelp] for
// Cobra's stock defaults.
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

// WithoutHelp disables Nabat help entirely (no custom renderer, flag, or
// subcommand); Cobra defaults take over. Mixing with other With(out)Help*
// options returns [ErrInvalidOption].
func WithoutHelp() Option {
	return optionFn(func(c *config) error {
		c.help.disabled = true
		c.help.disabledTouched = true
		return nil
	})
}
