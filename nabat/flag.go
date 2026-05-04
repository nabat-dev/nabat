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
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

type flagDef struct {
	name      string
	valueType valueType
	config    fieldConfig
}

func (f flagDef) validate() error {
	if f.name == "" {
		return fmt.Errorf("nabat: flag name cannot be empty")
	}
	if f.valueType.kind == 0 {
		return fmt.Errorf("%w for flag %q", ErrInvalidValueType, f.name)
	}
	if f.config.hasShort && f.config.short == 0 {
		return fmt.Errorf("nabat: flag %q has invalid short option", f.name)
	}
	if (f.valueType.kind == valueSelect || f.valueType.kind == valueMultiSelect) && len(f.valueType.choices) == 0 {
		return fmt.Errorf("nabat: flag %q: Select/MultiSelect requires at least one choice", f.name)
	}
	// Empty deprecation messages are rejected up-front in
	// [buildDeprecation]; only the structural constraint that the shorthand
	// must exist before it can be deprecated lives here.
	if f.config.shorthandDeprecated && !f.config.hasShort {
		return fmt.Errorf("nabat: flag %q: WithDeprecatedShorthand requires WithShort", f.name)
	}
	return validateDefaultType("flag", f.name, f.valueType, f.config)
}

func registerFlagOnCommand(cmd *cobra.Command, f flagDef, envPrefix string) error {
	target := cmd.Flags()
	if f.config.persistent {
		target = cmd.PersistentFlags()
	}

	usage := f.config.usage
	if frag := f.config.envUsageFragment(envPrefix); frag != "" {
		if usage != "" {
			usage += " "
		}
		usage += frag
	}

	short := ""
	if f.config.hasShort {
		short = string(f.config.short)
	}

	a := adapterFor(f.valueType.kind)
	if a == nil {
		return fmt.Errorf("%w for flag %q", ErrInvalidValueType, f.name)
	}
	var def any
	if f.config.hasDefault {
		def = f.config.defaultValue
	}
	if err := a.registerFlag(target, f.name, short, usage, def); err != nil {
		return fmt.Errorf("nabat: flag %q: %w", f.name, err)
	}
	if f.config.deprecated {
		if err := target.MarkDeprecated(f.name, strings.TrimSpace(f.config.deprecatedMsg)); err != nil {
			return fmt.Errorf("nabat: flag %q: %w", f.name, err)
		}
		// pflag marks deprecated flags Hidden so they disappear from default templates;
		// Nabat custom help still lists the flag with a deprecation suffix, so keep it visible.
		if fl := target.Lookup(f.name); fl != nil {
			fl.Hidden = false
		}
	}
	if f.config.shorthandDeprecated {
		if err := target.MarkShorthandDeprecated(f.name, strings.TrimSpace(f.config.shorthandDeprecatedMsg)); err != nil {
			return fmt.Errorf("nabat: flag %q: %w", f.name, err)
		}
	}
	if f.config.hidden {
		if fl := target.Lookup(f.name); fl != nil {
			fl.Hidden = true
		}
	}
	return nil
}

func validateDefaultType(kind, name string, vt valueType, cfg fieldConfig) error {
	if !cfg.hasDefault {
		return nil
	}
	a := adapterFor(vt.kind)
	if a == nil {
		return fmt.Errorf("%w for %s %q", ErrInvalidValueType, kind, name)
	}
	if err := a.checkDefault(cfg.defaultValue); err != nil {
		return fmt.Errorf("nabat: %s %q %s", kind, name, err.Error())
	}
	// Choice membership is enforced for non-required select/multi-select defaults.
	switch vt.kind {
	case valueSelect:
		if !cfg.required {
			v := defaultAs[string](cfg.defaultValue)
			if len(vt.choices) > 0 && !slices.Contains(vt.choices, v) {
				return fmt.Errorf("nabat: %s %q default must be one of %v", kind, name, vt.choices)
			}
		}
	case valueMultiSelect:
		if !cfg.required {
			s := defaultAs[[]string](cfg.defaultValue)
			for _, item := range s {
				if !slices.Contains(vt.choices, item) {
					return fmt.Errorf("nabat: %s %q default value %q is not in choices %v", kind, name, item, vt.choices)
				}
			}
		}
	}
	return nil
}
