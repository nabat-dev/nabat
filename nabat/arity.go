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

// ArityOption configures positional argument count validation for Cobra's
// [cobra.Command.Args].
// Use with [WithArgArity]. Nabat maps these to Cobra validators internally.
type ArityOption func(*arityConfig)

type arityConfig struct {
	exact *int
	minN  *int
	maxN  *int
}

// WithArgArity applies nested arity rules. [WithArgArity] may only be used
// once per command.
// Passing a nil [ArityOption] returns [ErrNilOption].
//
// Example:
//
//	app.MustCommand("copy",
//	    WithArg("src", "", WithRequired()),
//	    WithArg("dst", "", WithRequired()),
//	    WithArgArity(WithExactArgCount(2)),
//	    WithRun(func(c *Context) error { return nil }),
//	)
func WithArgArity(opts ...ArityOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		if len(c.arityOpts) > 0 {
			return fmt.Errorf("nabat: WithArgArity may only be used once per command")
		}
		if len(opts) == 0 {
			return fmt.Errorf("nabat: WithArgArity requires at least one ArityOption")
		}
		for i, o := range opts {
			if o == nil {
				return fmt.Errorf("%w at arity option index %d", ErrNilOption, i)
			}
			c.arityOpts = append(c.arityOpts, o)
		}
		return nil
	}}
}

// WithExactArgCount requires exactly n positional arguments before any "--"
// separator.
//
// Example:
//
//	WithArgArity(WithExactArgCount(2))
func WithExactArgCount(n int) ArityOption {
	return func(a *arityConfig) {
		v := n
		a.exact = &v
	}
}

// WithMinArgCount requires at least n positional arguments.
//
// Example:
//
//	WithArgArity(WithMinArgCount(1), WithMaxArgCount(3))
func WithMinArgCount(n int) ArityOption {
	return func(a *arityConfig) {
		v := n
		a.minN = &v
	}
}

// WithMaxArgCount allows at most n positional arguments.
//
// Example:
//
//	WithArgArity(WithMaxArgCount(1))
func WithMaxArgCount(n int) ArityOption {
	return func(a *arityConfig) {
		v := n
		a.maxN = &v
	}
}

func buildArityValidator(opts []ArityOption) (cobra.PositionalArgs, error) {
	var ac arityConfig
	for _, o := range opts {
		o(&ac)
	}
	if ac.exact != nil {
		if ac.minN != nil || ac.maxN != nil {
			return nil, fmt.Errorf("nabat: WithExactArgCount cannot be combined with WithMinArgCount or WithMaxArgCount")
		}
		return cobra.ExactArgs(*ac.exact), nil
	}
	var parts []cobra.PositionalArgs
	if ac.minN != nil {
		parts = append(parts, cobra.MinimumNArgs(*ac.minN))
	}
	if ac.maxN != nil {
		parts = append(parts, cobra.MaximumNArgs(*ac.maxN))
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("nabat: WithArgArity requires WithExactArgCount and/or WithMinArgCount/WithMaxArgCount")
	}
	if len(parts) == 1 {
		return parts[0], nil
	}
	return cobra.MatchAll(parts...), nil
}
