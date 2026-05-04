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

// ParseOption configures low-level Cobra parsing for a single command (including
// the root by passing options directly to [New]). Use with [WithParseOptions].
type ParseOption func(*parseConfig)

type parseConfig struct {
	allowUnknownFlags     bool
	traverseChildrenSet   bool
	traverseChildrenValue bool
	disableFlagParsing    bool
}

// WithParseOptions applies nested parsing options to the command.
// Passing a nil [ParseOption] returns [ErrNilOption]. An empty opts slice is a
// no-op.
//
// Example:
//
//	app.MustCommand("proxy",
//	    WithParseOptions(
//	        WithAllowUnknownFlags(),
//	        WithTraverseChildren(true),
//	    ),
//	    WithRun(func(c *Context) error { return nil }),
//	)
func WithParseOptions(opts ...ParseOption) RootOption {
	return rootOpt{fn: func(c *commandSpec) error {
		for i, o := range opts {
			if o == nil {
				return fmt.Errorf("%w at parse option index %d", ErrNilOption, i)
			}
			c.parseOpts = append(c.parseOpts, o)
		}
		return nil
	}}
}

// WithAllowUnknownFlags sets [cobra.Command.FParseErrWhitelist].UnknownFlags so
// unknown flags do not fail parsing. Useful for wrapper CLIs that forward argv.
//
// Example:
//
//	WithParseOptions(WithAllowUnknownFlags())
func WithAllowUnknownFlags() ParseOption {
	return func(p *parseConfig) {
		p.allowUnknownFlags = true
	}
}

// WithTraverseChildren sets [cobra.Command.TraverseChildren]. When true, the
// parent command parses local flags before dispatching to a subcommand.
//
// Example:
//
//	WithParseOptions(WithTraverseChildren(true))
func WithTraverseChildren(enabled bool) ParseOption {
	return func(p *parseConfig) {
		p.traverseChildrenSet = true
		p.traverseChildrenValue = enabled
	}
}

// WithDisableFlagParsing sets [cobra.Command.DisableFlagParsing]: every token
// after the command name is treated as a positional argument. Often combined with
// [WithPassthrough] for exec-style commands.
//
// Example:
//
//	WithParseOptions(WithDisableFlagParsing())
func WithDisableFlagParsing() ParseOption {
	return func(p *parseConfig) {
		p.disableFlagParsing = true
	}
}

func applyParseOptions(cmd *cobra.Command, opts []ParseOption) {
	if len(opts) == 0 {
		return
	}
	var pc parseConfig
	for _, o := range opts {
		o(&pc)
	}
	if pc.allowUnknownFlags {
		cmd.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	}
	if pc.traverseChildrenSet {
		cmd.TraverseChildren = pc.traverseChildrenValue
	}
	if pc.disableFlagParsing {
		cmd.DisableFlagParsing = true
	}
}
