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

// VersionOption configures the built-in version feature inside [WithVersion].
//
// Version is a built-in core feature: pass [WithVersion] to [New] or [MustNew]
// and the App grows a `version` subcommand and a `--version` (`-v`) flag with
// Nabat's themed output. Without [WithVersion], no version surface is installed.
//
// Defaults: subcommand "version", flag "--version", shorthand "-v". Tweak the
// defaults by passing [VersionOption] values to [WithVersion]; disable a piece
// with [WithoutVersionCommand], [WithoutVersionFlag], or
// [WithoutVersionShorthand]. To opt out of the feature entirely, simply omit
// [WithVersion].
//
// All polish options are [VersionOption] values that nest inside [WithVersion]
// rather than living at the App level. This groups version-related config in
// one place and keeps the App-level option list short.
type VersionOption func(*versionConfig) error

type versionConfig struct {
	version        string
	commit         string
	date           string
	dateTimeFormat string
	commandName    string
	flagName       string
	shorthand      rune

	flagDisabled      bool
	commandDisabled   bool
	shorthandDisabled bool
	shorthandSet      bool
}

func defaultVersionConfig() *versionConfig {
	return &versionConfig{
		commandName: "version",
		flagName:    "version",
		shorthand:   'v',
	}
}

func (vc *versionConfig) validate() error {
	if vc.flagDisabled && vc.commandDisabled {
		return fmt.Errorf("%w: WithVersion cannot disable both flag and command (omit WithVersion entirely instead)", ErrInvalidOption)
	}
	if vc.shorthandDisabled && vc.shorthandSet {
		return fmt.Errorf("%w: WithoutVersionShorthand cannot be combined with WithVersionShorthand", ErrInvalidOption)
	}
	return nil
}

// WithVersion enables the built-in version feature. The version string is
// required and printed by both the `version` subcommand and the `--version`
// flag. Empty strings return [ErrInvalidOption]; pass an explicit value such
// as one read from [runtime/debug.ReadBuildInfo] for build-info derivation.
//
// Pass [VersionOption] values to override defaults or disable individual
// pieces. Disabling both the flag and the command at the same time is an
// error: omit [WithVersion] entirely instead.
//
// Example:
//
//	New("ctl",
//	    WithVersion("1.2.3",
//	        WithVersionCommit("abc1234"),
//	        WithVersionShorthand('V'),
//	    ),
//	)
//
// Result: `ctl version`, `ctl version --format short`, `ctl --version`, `ctl -V`.
func WithVersion(version string, opts ...VersionOption) Option {
	return optionFn(func(c *config) error {
		if version == "" {
			return fmt.Errorf("%w: WithVersion: version string cannot be empty", ErrInvalidOption)
		}
		vc := defaultVersionConfig()
		vc.version = version
		for i, opt := range opts {
			if opt == nil {
				return fmt.Errorf("%w at WithVersion option index %d", ErrNilOption, i)
			}
			if err := opt(vc); err != nil {
				return fmt.Errorf("nabat: WithVersion: %w", err)
			}
		}
		if err := vc.validate(); err != nil {
			return err
		}
		c.version = vc
		return nil
	})
}

// WithVersionCommit explicitly sets the VCS commit SHA shown next to the
// version string. When omitted, the value is read from
// [runtime/debug.ReadBuildInfo] (vcs.revision, truncated to 7 chars).
func WithVersionCommit(commit string) VersionOption {
	return func(vc *versionConfig) error {
		vc.commit = commit
		return nil
	}
}

// WithVersionCommitDate explicitly sets the commit date shown next to the version
// string. The value is used verbatim; [WithVersionCommitDateTimeFormat] has no
// effect on it. When omitted, the value is read from
// [runtime/debug.ReadBuildInfo] (vcs.time).
func WithVersionCommitDate(date string) VersionOption {
	return func(vc *versionConfig) error {
		vc.date = date
		return nil
	}
}

// WithVersionCommitDateTimeFormat sets the Go time layout used to format the
// commit timestamp read from [runtime/debug.ReadBuildInfo] (vcs.time, always
// RFC3339). Has no effect when the date is set explicitly via
// [WithVersionCommitDate], since that value is used verbatim.
//
// Example:
//
//	WithVersionCommitDateTimeFormat("2006-01-02")          // date only
//	WithVersionCommitDateTimeFormat("2006-01-02 15:04 MST") // compact with timezone
func WithVersionCommitDateTimeFormat(layout string) VersionOption {
	return func(vc *versionConfig) error {
		if layout == "" {
			return fmt.Errorf("%w: WithVersionCommitDateTimeFormat: layout cannot be empty", ErrInvalidOption)
		}
		vc.dateTimeFormat = layout
		return nil
	}
}

// WithVersionCommandName overrides the subcommand name (default "version").
// Empty strings return [ErrInvalidOption]; use [WithoutVersionCommand] to
// disable the subcommand.
func WithVersionCommandName(name string) VersionOption {
	return func(vc *versionConfig) error {
		if name == "" {
			return fmt.Errorf("%w: WithVersionCommandName: name cannot be empty (use WithoutVersionCommand to disable)", ErrInvalidOption)
		}
		vc.commandName = name
		return nil
	}
}

// WithoutVersionCommand disables the `version` subcommand. The `--version`
// flag is still installed unless [WithoutVersionFlag] is also passed (in
// which case [WithVersion] returns [ErrInvalidOption]).
func WithoutVersionCommand() VersionOption {
	return func(vc *versionConfig) error {
		vc.commandDisabled = true
		return nil
	}
}

// WithVersionFlagName overrides the long flag name (default "version").
// Empty strings return [ErrInvalidOption]; use [WithoutVersionFlag] to
// disable the flag.
func WithVersionFlagName(name string) VersionOption {
	return func(vc *versionConfig) error {
		if name == "" {
			return fmt.Errorf("%w: WithVersionFlagName: name cannot be empty (use WithoutVersionFlag to disable)", ErrInvalidOption)
		}
		vc.flagName = name
		return nil
	}
}

// WithoutVersionFlag disables the `--version` flag. The `version` subcommand
// is still installed unless [WithoutVersionCommand] is also passed (in which
// case [WithVersion] returns [ErrInvalidOption]).
func WithoutVersionFlag() VersionOption {
	return func(vc *versionConfig) error {
		vc.flagDisabled = true
		vc.shorthand = 0
		return nil
	}
}

// WithVersionShorthand sets the one-character shorthand for the version flag
// (default 'v'). The rune must be a printable ASCII character (0x21 to 0x7E);
// Cobra does not support multi-byte shorthands.
func WithVersionShorthand(r rune) VersionOption {
	return func(vc *versionConfig) error {
		if r < '!' || r > '~' {
			return fmt.Errorf("%w: WithVersionShorthand: must be a printable ASCII character (0x21-0x7E), got %q", ErrInvalidOption, r)
		}
		vc.shorthand = r
		vc.shorthandSet = true
		return nil
	}
}

// WithoutVersionShorthand disables the version flag's shorthand. The long
// form (default --version) keeps working.
func WithoutVersionShorthand() VersionOption {
	return func(vc *versionConfig) error {
		vc.shorthandDisabled = true
		vc.shorthand = 0
		return nil
	}
}
