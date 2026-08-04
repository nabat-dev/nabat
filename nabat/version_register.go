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

	"nabat.dev/theme"
)

// registerVersion wires built-in version onto the root from versionConfig
// (see version_options.go). Called from [New] after [App.registerHelp].
// When [WithVersion] was passed, it resolves build info, registers the
// optional subcommand and hidden flag, and installs a pre-run short-circuit.
func (a *App) registerVersion() error {
	vc := a.cfg.version
	if vc == nil {
		return nil
	}

	bi := resolveBuildInfo(vc.version, vc.commit, vc.date, vc.dateTimeFormat)
	v := versioner{name: a.cfg.name, bi: bi, infoStyle: a.Theme().Style(theme.StatusInfo)}
	v.cachedLine = v.line()

	if !vc.commandDisabled {
		if _, err := a.newCommand(a.root, vc.commandName,
			WithDescription("Print version information"),
			WithFlag("format", "text", WithUsage(`output format: "text", "short", or "json"`)),
			WithRun(func(c *Context) error {
				format, err := c.BindAs[string]("format")
				if err != nil {
					return err
				}
				return v.print(a.io.Out, format)
			}),
		); err != nil {
			return err
		}
	}

	if !vc.flagDisabled {
		if err := a.appendVersionFlagToSpec(vc.flagName, vc.shorthand); err != nil {
			return err
		}
		flagName := vc.flagName
		if err := a.OnPreRun(func(c *Context) error {
			if wantVer, ok := c.values[flagName].(bool); ok && c.set[flagName] && wantVer {
				if err := v.print(a.io.Out, "text"); err != nil {
					return err
				}
				return ErrHandled
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// appendVersionFlagToSpec adds a hidden boolean version flag to the root
// spec so the standard root-flag-registration loop in [New] picks it up.
// The shorthand is omitted when zero. Returns a Nabat-styled error when the
// name or shorthand conflicts with a user-defined root flag rather than
// letting pflag panic on duplicate registration.
func (a *App) appendVersionFlagToSpec(name string, shorthand rune) error {
	rootSpec := a.meta[a.root]
	if rootSpec == nil {
		return fmt.Errorf("nabat: registerVersion: root spec missing")
	}
	for _, fl := range rootSpec.flags {
		if fl.name == name {
			return fmt.Errorf("nabat: registerVersion: flag --%s collides with a user-defined root flag (use WithoutVersionFlag or rename via WithVersionFlagName to resolve)", name)
		}
		if shorthand != 0 && fl.config.hasShort && fl.config.short == shorthand {
			return fmt.Errorf("nabat: registerVersion: shorthand -%c collides with user-defined root flag --%s (use WithoutVersionShorthand or pick a different WithVersionShorthand)", shorthand, fl.name)
		}
	}
	for _, in := range rootSpec.args {
		if in.name == name {
			return fmt.Errorf("nabat: registerVersion: name %q collides with a user-defined root arg (rename via WithVersionFlagName)", name)
		}
	}
	flagOpts := []FlagOption{
		WithHidden(),
		WithUsage("version for " + a.cfg.name),
	}
	if shorthand != 0 {
		flagOpts = append(flagOpts, WithShort(shorthand))
	}
	opt := WithFlag(name, false, flagOpts...)
	if err := opt.applyToCommand(rootSpec); err != nil {
		return fmt.Errorf("nabat: registerVersion: %w", err)
	}
	return nil
}
