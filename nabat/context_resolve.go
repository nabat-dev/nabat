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
	"cmp"
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
)

func (a *App) resolveArgs(c *Context) error {
	meta := a.meta[c.cmd]
	if meta == nil || len(meta.args) == 0 {
		return nil
	}

	for idx, in := range meta.args {
		if idx < len(c.args) {
			v, parseErr := parseStringToType(c.args[idx], in.valueType)
			if parseErr != nil {
				return fmt.Errorf("nabat: arg %q from positional argument: %w", in.name, parseErr)
			}
			c.values[in.name] = v
			c.set[in.name] = true
			continue
		}

		if v, ok, err := lookupEnvForField(in.config, a.cfg.envPrefix, in.valueType); err != nil {
			return fmt.Errorf("nabat: arg %q: %w", in.name, err)
		} else if ok {
			c.values[in.name] = v
			c.set[in.name] = true
			continue
		}

		// Declarative args use the typed defaultVal passed to WithArg as
		// their non-interactive fallback (see prompt.go for the rationale);
		// hasFallback only applies to ad-hoc Context.* prompts.
		if c.interactive && in.prompt.text != "" {
			v, promptErr := a.promptArg(in)
			if promptErr != nil {
				return fmt.Errorf("nabat: prompt for arg %q failed: %w", in.name, promptErr)
			}
			c.values[in.name] = v
			c.set[in.name] = true
			continue
		}

		if in.config.required {
			return fmt.Errorf("nabat: required arg %q is missing (args/env/prompt)", in.name)
		}

		if in.config.hasDefault {
			c.values[in.name] = cloneDefault(in.config.defaultValue)
			continue
		}
	}

	// Single post-resolution choice membership pass. Keeping this in one place
	// means each new source (arg, env, prompt) does not have to remember to
	// call [validateChoice] on its own.
	for _, in := range meta.args {
		v, ok := c.values[in.name]
		if !ok {
			continue
		}
		if err := validateChoice(in.valueType, v); err != nil {
			return fmt.Errorf("nabat: arg %q: %w", in.name, err)
		}
	}
	return nil
}

func (a *App) resolveFlags(c *Context) error {
	flagDefs := a.collectFlagDefsForCommand(c.cmd)
	for _, fl := range flagDefs {
		value, resolved, explicit, err := resolveOneFlag(c.cmd, fl, a.cfg.envPrefix)
		if err != nil {
			return err
		}
		if resolved {
			c.values[fl.name] = value
			if explicit {
				c.set[fl.name] = true
			}
			continue
		}
		if fl.config.required {
			return fmt.Errorf("nabat: required flag --%s is missing (flag/env/default)", fl.name)
		}
	}

	for _, fl := range flagDefs {
		v, ok := c.values[fl.name]
		if !ok {
			continue
		}
		if err := validateChoice(fl.valueType, v); err != nil {
			return fmt.Errorf("nabat: flag %q: %w", fl.name, err)
		}
	}
	return nil
}

func (a *App) collectFlagDefsForCommand(cmd *cobra.Command) []flagDef {
	// Walk leaf→root then reverse for root→leaf precedence (leaf wins on conflict).
	var lineage []*cobra.Command
	for cur := cmd; cur != nil; {
		lineage = append(lineage, cur)
		meta := a.meta[cur]
		if meta == nil {
			break
		}
		cur = meta.parent
	}
	slices.Reverse(lineage)

	byName := make(map[string]flagDef, 8)
	for _, node := range lineage {
		meta := a.meta[node]
		if meta == nil {
			continue
		}
		for _, fl := range meta.flags {
			if node == cmd || fl.config.persistent {
				byName[fl.name] = fl
			}
		}
	}

	out := make([]flagDef, 0, len(byName))
	for _, fl := range byName {
		out = append(out, fl)
	}
	slices.SortStableFunc(out, func(a, b flagDef) int {
		return cmp.Compare(a.name, b.name)
	})
	return out
}

// resolveOneFlag resolves a single flag value. Returns (value, resolved,
// explicit, err).
// explicit is true when the value came from a CLI flag or env var (not from a
// default).
func resolveOneFlag(cmd *cobra.Command, fl flagDef, envPrefix string) (any, bool, bool, error) {
	flags := cmd.Flags()
	if f := flags.Lookup(fl.name); f != nil && f.Changed {
		v, ok, err := readFlagTypedValue(cmd, fl)
		return v, ok, true, err
	}

	if v, ok, err := lookupEnvForField(fl.config, envPrefix, fl.valueType); err != nil {
		return nil, false, false, fmt.Errorf("nabat: flag --%s: %w", fl.name, err)
	} else if ok {
		return v, true, true, nil
	}

	if fl.config.hasDefault && !fl.config.required {
		return cloneDefault(fl.config.defaultValue), true, false, nil
	}
	return nil, false, false, nil
}

// lookupEnvForField walks the env candidate list (prefix-prepended names from
// [WithEnv] first, then literal names from [WithEnvAlias]) and returns the
// first non-empty match parsed into the appropriate type. When no env source
// is configured, no lookup is performed. Choice membership is enforced in
// the single post-resolution pass, not here.
func lookupEnvForField(cfg fieldConfig, envPrefix string, vt valueType) (any, bool, error) {
	if !cfg.envEnabled {
		return nil, false, nil
	}

	candidates := make([]string, 0, len(cfg.envPrefixed)+len(cfg.envLiteral))
	for _, n := range cfg.envPrefixed {
		candidates = append(candidates, envPrefix+normalizeEnvName(n))
	}
	candidates = append(candidates, cfg.envLiteral...)

	for _, key := range candidates {
		if raw, ok := os.LookupEnv(key); ok && raw != "" {
			v, parseErr := parseStringToType(raw, vt)
			if parseErr != nil {
				return nil, false, fmt.Errorf("from env %s: %w", key, parseErr)
			}
			return v, true, nil
		}
	}
	return nil, false, nil
}

func readFlagTypedValue(cmd *cobra.Command, fl flagDef) (any, bool, error) {
	a := adapterFor(fl.valueType.kind)
	if a == nil {
		return nil, false, fmt.Errorf("%w for --%s", ErrInvalidValueType, fl.name)
	}
	v, err := a.readFlag(cmd.Flags(), fl.name)
	return v, err == nil, wrapFlagErr(fl.name, err)
}

func wrapFlagErr(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("nabat: read flag --%s: %w", name, err)
}
