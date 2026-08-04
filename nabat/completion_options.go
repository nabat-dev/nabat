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

// CompletionOption configures the built-in completion feature inside
// [WithCompletion]. Per-flag [WithCompleter] and [WithPositionalCompleter]
// work without installing the subcommand.
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

// WithCompletion installs a `completion` subcommand with shell generators.
// Pass [CompletionOption] values to override defaults. Omit it to skip the
// subcommand; [WithCompleter] still works without it.
//
// Example:
//
//	New("ctl", WithCompletion(WithCompletionShells("bash", "zsh")))
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
// Empty strings return [ErrInvalidOption].
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
func WithCompletionHidden() CompletionOption {
	return func(cc *completionConfig) error {
		cc.hidden = true
		return nil
	}
}

// WithCompletionShells restricts installed generators to the listed shells
// (default: bash, zsh, fish, powershell). Unknown names return [ErrInvalidOption].
func WithCompletionShells(shells ...string) CompletionOption {
	return func(cc *completionConfig) error {
		cc.shells = append(cc.shells[:0:0], shells...)
		cc.shellsSet = true
		return nil
	}
}
