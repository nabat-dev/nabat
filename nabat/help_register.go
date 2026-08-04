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

	"github.com/spf13/cobra"
)

// registerHelp wires built-in help onto the root from helpConfig (see
// help_options.go). Called from [New] before root flags register.
// WithoutHelp leaves Cobra defaults; otherwise it sets the custom renderer,
// optional help subcommand, persistent help flag (plus a hidden --help alias
// when renamed via [WithHelpFlagName]), and a pre-run short-circuit for the
// help flag.
func (a *App) registerHelp() error {
	h := a.cfg.help
	if h.disabled {
		return nil
	}

	a.root.SetHelpFunc(a.renderHelp)

	if h.command != nil {
		root := a.root
		a.root.SetHelpCommand(&cobra.Command{
			Use:                   h.command.name + " [command]",
			Short:                 "Help about any command",
			DisableFlagsInUseLine: true,
			RunE: func(_ *cobra.Command, args []string) error {
				cmd, _, err := root.Find(args)
				if cmd == nil || err != nil {
					cmd = root
				}
				return cmd.Help()
			},
		})
	} else {
		// Suppress Cobra's auto-injected `help` subcommand by replacing it
		// with a hidden no-op. The Nabat default is "no help subcommand";
		// developers opt in via [WithHelpCommand] (or define their own
		// `help` command via [App.Command]).
		a.root.SetHelpCommand(&cobra.Command{
			Use:    "no-help",
			Hidden: true,
			Run:    func(*cobra.Command, []string) {},
		})
	}

	if h.flag.name != "" {
		// Primary flag is visible so users discover --help (and its
		// shorthand) the same way they do in cobra, kubectl, gh, etc.
		if err := a.appendHelpFlagToSpec(h.flag.name, h.flag.shorthand, false); err != nil {
			return err
		}
		// When the user renamed the primary flag, register a hidden
		// alias --help so we preempt Cobra's auto-injected --help and
		// avoid two help flags coexisting. The alias stays hidden so it
		// does not duplicate the primary flag in rendered output.
		if h.flag.name != "help" {
			if err := a.appendHelpFlagToSpec("help", 0, true); err != nil {
				return err
			}
		}
	}

	primary := h.flag.name
	if err := a.OnPreRun(func(c *Context) error {
		if primary == "" {
			return nil
		}
		if v, ok := c.values[primary].(bool); ok && c.set[primary] && v {
			if err := c.cmd.Help(); err != nil {
				return err
			}
			return ErrHandled
		}
		if primary != "help" {
			if v, ok := c.values["help"].(bool); ok && c.set["help"] && v {
				if err := c.cmd.Help(); err != nil {
					return err
				}
				return ErrHandled
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// appendHelpFlagToSpec adds a persistent boolean help flag to the root spec so
// the standard root-flag-registration loop in [New] picks it up. The shorthand
// is omitted when zero. Setting hidden suppresses the flag from rendered help
// output; this is used for the redundant --help alias when the primary flag
// has been renamed via [WithHelpFlagName]. Returns a Nabat-styled error when
// the name or shorthand conflicts with a user-defined root flag rather than
// letting pflag panic on duplicate registration.
func (a *App) appendHelpFlagToSpec(name string, shorthand rune, hidden bool) error {
	rootSpec := a.meta[a.root]
	if rootSpec == nil {
		return fmt.Errorf("nabat: registerHelp: root spec missing")
	}
	for _, fl := range rootSpec.flags {
		if fl.name == name {
			return fmt.Errorf("nabat: registerHelp: flag --%s collides with a user-defined root flag (use WithoutHelpFlag or rename via WithHelpFlagName to resolve)", name)
		}
		if shorthand != 0 && fl.config.hasShort && fl.config.short == shorthand {
			return fmt.Errorf("nabat: registerHelp: shorthand -%c collides with user-defined root flag --%s (use WithoutHelpShorthand or pick a different WithHelpShorthand)", shorthand, fl.name)
		}
	}
	for _, in := range rootSpec.args {
		if in.name == name {
			return fmt.Errorf("nabat: registerHelp: name %q collides with a user-defined root arg (rename via WithHelpFlagName)", name)
		}
	}
	flagOpts := []FlagOption{
		WithPersistent(),
		WithUsage("show help for this command"),
	}
	if hidden {
		flagOpts = append(flagOpts, WithHidden())
	}
	if shorthand != 0 {
		flagOpts = append(flagOpts, WithShort(shorthand))
	}
	opt := WithFlag(name, false, flagOpts...)
	if err := opt.applyToCommand(rootSpec); err != nil {
		return fmt.Errorf("nabat: registerHelp: %w", err)
	}
	return nil
}
