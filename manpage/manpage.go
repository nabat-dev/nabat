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

package manpage

import (
	"fmt"
	"os"

	"github.com/muesli/roff"
	"github.com/spf13/cobra"

	"nabat.dev/nabat"

	mcobra "github.com/muesli/mango-cobra"
)

type config struct {
	commandName string
	section     int
	hidden      bool
}

func (c *config) validate() error {
	if c.commandName == "" {
		return fmt.Errorf("nabat/manpage: command name cannot be empty")
	}
	if c.section < 1 || c.section > 9 {
		return fmt.Errorf("nabat/manpage: section must be between 1 and 9, got %d", c.section)
	}
	return nil
}

// Option configures the manpage extension.
type Option interface {
	applyToConfig(*config) error
}

type optionFn func(*config) error

func (f optionFn) applyToConfig(c *config) error { return f(c) }

// WithCommandName overrides the man subcommand name (default "man").
func WithCommandName(name string) Option {
	return optionFn(func(c *config) error {
		c.commandName = name
		return nil
	})
}

// WithSection sets the default man page section (1-9; default 1).
// Users can override at runtime via --section.
func WithSection(n int) Option {
	return optionFn(func(c *config) error {
		c.section = n
		return nil
	})
}

// WithHidden hides the man subcommand from help listings.
func WithHidden() Option {
	return optionFn(func(c *config) error {
		c.hidden = true
		return nil
	})
}

type extension struct {
	cfg config
}

func (e *extension) String() string { return "manpage" }

func (e *extension) Init(app nabat.AppSurface) error {
	cfg := e.cfg
	root := app.UnsafeRoot()

	cmdOpts := []nabat.CommandOption{
		nabat.WithDescription("Generate man page documentation"),
		nabat.WithLongDescription("Generate the man page for " + app.Name() + ".\nPipe the output to groff or save it directly to a man page file."),
		nabat.WithFlag("section", cfg.section, nabat.WithShort('s'), nabat.WithUsage("man page section number")),
		nabat.WithFlag("output", "", nabat.WithShort('o'), nabat.WithUsage("write to `file` instead of stdout")),
	}
	if cfg.hidden {
		cmdOpts = append(cmdOpts, nabat.WithHidden())
	}
	cmdOpts = append(cmdOpts,
		nabat.WithRun(func(c *nabat.Context) error {
			section, err := nabat.BindAs[int](c, "section")
			if err != nil {
				return err
			}
			output, err := nabat.BindAs[string](c, "output")
			if err != nil {
				return err
			}
			doc, renderErr := render(root, section)
			if renderErr != nil {
				return renderErr
			}
			if output != "" {
				if writeErr := os.WriteFile(output, []byte(doc), 0o600); writeErr != nil {
					return fmt.Errorf("writing man page: %w", writeErr)
				}
				return nil
			}
			_, writeErr := fmt.Fprint(app.IO().Out, doc)
			return writeErr
		}),
	)
	_, err := app.Command(cfg.commandName, cmdOpts...)
	return err
}

// New builds a [nabat.Extension] that installs the man subcommand and returns an
// error when any option is nil, when option application fails, or when the
// configuration is invalid.
//
// Errors:
//   - "nabat/manpage: option at index N is nil": an entry in opts is nil.
//   - errors wrapped with "nabat/manpage: option at index N":
//     [Option] application failed.
//   - "nabat/manpage: command name cannot be empty":
//     [WithCommandName] set an empty name.
//   - "nabat/manpage: section must be between 1 and 9, got N":
//     the effective section is out of range.
func New(opts ...Option) (nabat.Extension, error) {
	cfg := config{
		commandName: "man",
		section:     1,
	}
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("nabat/manpage: option at index %d is nil", i)
		}
		if err := opt.applyToConfig(&cfg); err != nil {
			return nil, fmt.Errorf("nabat/manpage: option at index %d: %w", i, err)
		}
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &extension{cfg: cfg}, nil
}

// render returns the man-page (roff) document for the given Cobra command tree
// at the requested section number. Section must be 1-9.
func render(cmd *cobra.Command, section int) (string, error) {
	if section < 1 || section > 9 {
		return "", fmt.Errorf("nabat/manpage: section must be between 1 and 9, got %d", section)
	}
	man, err := mcobra.NewManPage(uint(section), cmd)
	if err != nil {
		return "", err
	}
	return man.Build(roff.NewDocument()), nil
}
