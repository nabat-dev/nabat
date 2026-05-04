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

// Completion is a built-in core feature: pass [WithCompletion] to [New] or
// [MustNew] and the App grows a `completion` subcommand that emits shell
// completion scripts for bash, zsh, fish, and PowerShell. Without
// [WithCompletion], no `completion` subcommand is installed.
//
// Per-flag and per-positional completers ([WithCompleter] and
// [WithPositionalCompleter]) are independent of this option: they always work
// because Cobra's hidden `__complete` command is part of every Cobra binary.
// [WithCompletion] is the convenience surface users invoke to obtain the
// install scripts for their shell.
//
// Defaults: subcommand "completion", visible in help, all four shell
// generators installed. Tweak with [CompletionOption] values nested inside
// [WithCompletion] or omit it entirely to skip the subcommand.

// CompletionOption configures the built-in completion feature inside
// [WithCompletion].
type CompletionOption func(*completionConfig) error

type completionConfig struct {
	commandName string
	hidden      bool
	shells      []string
	shellsSet   bool
}

func defaultCompletionConfig() *completionConfig {
	return &completionConfig{commandName: "completion"}
}

func (cc *completionConfig) validate() error {
	if cc.commandName == "" {
		return fmt.Errorf("%w: WithCompletionName: name cannot be empty", ErrInvalidOption)
	}
	if cc.shellsSet && len(cc.shells) == 0 {
		return fmt.Errorf("%w: WithCompletionShells requires at least one shell", ErrInvalidOption)
	}
	for _, s := range cc.shells {
		switch s {
		case "bash", "zsh", "fish", "powershell":
		default:
			return fmt.Errorf("%w: WithCompletionShells: unsupported shell %q (want bash|zsh|fish|powershell)", ErrInvalidOption, s)
		}
	}
	return nil
}

// WithCompletion enables the built-in completion feature, installing a
// `completion` subcommand with bash, zsh, fish, and PowerShell generators.
// Pass [CompletionOption] values to override defaults or restrict the set of
// installed generators. Omit [WithCompletion] entirely to skip the subcommand
// surface (per-flag [WithCompleter] still works without it).
//
// Example:
//
//	New("ctl",
//	    WithCompletion(),
//	)
//
//	New("ctl",
//	    WithCompletion(
//	        WithCompletionName("comp"),
//	        WithCompletionHidden(),
//	        WithCompletionShells("bash", "zsh"),
//	    ),
//	)
//
// Result: `ctl completion bash`, `ctl completion zsh`, etc., each emitting a
// ready-to-source shell script with install instructions in `--help`.
func WithCompletion(opts ...CompletionOption) Option {
	return optionFn(func(c *config) error {
		cc := defaultCompletionConfig()
		for i, opt := range opts {
			if opt == nil {
				return fmt.Errorf("%w at WithCompletion option index %d", ErrNilOption, i)
			}
			if err := opt(cc); err != nil {
				return fmt.Errorf("nabat: WithCompletion: %w", err)
			}
		}
		if err := cc.validate(); err != nil {
			return err
		}
		c.completion = cc
		return nil
	})
}

// WithCompletionName overrides the subcommand name (default "completion").
// Empty strings return [ErrInvalidOption]; omit [WithCompletion] entirely to
// disable the subcommand surface.
func WithCompletionName(name string) CompletionOption {
	return func(cc *completionConfig) error {
		if name == "" {
			return fmt.Errorf("%w: WithCompletionName: name cannot be empty (omit WithCompletion to disable the subcommand)", ErrInvalidOption)
		}
		cc.commandName = name
		return nil
	}
}

// WithCompletionHidden hides the completion subcommand from help listings.
// The command remains invokable; it just does not appear in `--help` output.
func WithCompletionHidden() CompletionOption {
	return func(cc *completionConfig) error {
		cc.hidden = true
		return nil
	}
}

// WithCompletionShells restricts the installed generators to the listed
// shells. When omitted, all four (bash, zsh, fish, powershell) are installed.
// Unknown shell names return [ErrInvalidOption].
func WithCompletionShells(shells ...string) CompletionOption {
	return func(cc *completionConfig) error {
		cc.shells = append(cc.shells[:0:0], shells...)
		cc.shellsSet = true
		return nil
	}
}
