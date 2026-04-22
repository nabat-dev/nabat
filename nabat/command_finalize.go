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

func (a *App) finalizeCommand(cmd *cobra.Command, spec *commandSpec) error {
	cmd.Hidden = spec.hidden
	cmd.Deprecated = strings.TrimSpace(spec.deprecatedCommand)
	if len(spec.typoHints) > 0 {
		cmd.SuggestFor = append([]string(nil), spec.typoHints...)
	}
	if len(spec.annotations) > 0 {
		if cmd.Annotations == nil {
			cmd.Annotations = make(map[string]string)
		}
		for _, kv := range spec.annotations {
			cmd.Annotations[kv.key] = kv.value
		}
	}

	applyParseOptions(cmd, spec.parseOpts)

	if len(spec.arityOpts) > 0 {
		v, err := buildArityValidator(spec.arityOpts)
		if err != nil {
			return err
		}
		if arityErr := validateArityAgainstDeclaredArgs(spec, spec.arityOpts); arityErr != nil {
			return arityErr
		}
		cmd.Args = v
	}
	// Intentionally no default cmd.Args from declared Nabat args: required values
	// may come from env, prompts, or defaults after Cobra parsing. Cobra-side
	// MinimumNArgs would reject valid invocations such as `DEPLOYCTL_KEY=val cmd sub`.

	if spec.argCompletions != nil {
		cmd.ValidArgsFunction = spec.argCompletions
	} else if fn := autoValidArgsFunction(spec); fn != nil {
		cmd.ValidArgsFunction = fn
	}

	for _, fl := range spec.flags {
		if fl.config.shellComplete != nil {
			if err := cmd.RegisterFlagCompletionFunc(fl.name, fl.config.shellComplete); err != nil {
				return fmt.Errorf("nabat: command %q: flag %q: %w", cmd.Name(), fl.name, err)
			}
		}
	}

	a.attachRunE(cmd, spec)
	return nil
}

func validateArityAgainstDeclaredArgs(spec *commandSpec, opts []ArityOption) error {
	if spec.passthrough != nil {
		return nil
	}
	var ac arityConfig
	for _, o := range opts {
		o(&ac)
	}
	n := len(spec.args)
	if ac.exact != nil && *ac.exact != n {
		return fmt.Errorf("nabat: WithExactArgCount(%d) does not match %d declared positional args", *ac.exact, n)
	}
	if ac.minN != nil && *ac.minN > n {
		return fmt.Errorf("nabat: WithMinArgCount(%d) exceeds %d declared positional args", *ac.minN, n)
	}
	if ac.maxN != nil && *ac.maxN < n {
		return fmt.Errorf("nabat: WithMaxArgCount(%d) is less than %d declared positional args", *ac.maxN, n)
	}
	return nil
}

func autoValidArgsFunction(spec *commandSpec) cobra.CompletionFunc {
	if len(spec.args) != 1 {
		return nil
	}
	a0 := spec.args[0]
	switch a0.valueType.kind {
	case valueSelect:
		choices := slices.Clone(a0.valueType.choices)
		return cobra.FixedCompletions(choices, cobra.ShellCompDirectiveDefault)
	case valueString:
		if len(a0.prompt.suggestions) == 0 {
			return nil
		}
		choices := slices.Clone(a0.prompt.suggestions)
		return cobra.FixedCompletions(choices, cobra.ShellCompDirectiveDefault)
	default:
		return nil
	}
}
