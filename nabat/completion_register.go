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
	"io"
	"os"
	"strings"
)

// registerCompletion wires Nabat's built-in completion feature onto the root
// command. It is called from [New] after [App.registerVersion] and before
// user-defined root flags are registered on Cobra. The behavior is driven
// entirely by the completion field on [config]; see completion_options.go for
// the public option surface.
//
// Steps:
//  1. If WithCompletion was not passed, leave the root untouched.
//  2. Register a `completion` parent subcommand (name configurable, optionally
//     hidden) carrying a persistent `--output` flag.
//  3. For each enabled shell, register a leaf subcommand that emits the
//     corresponding script via Cobra's generators and includes install
//     instructions in `--help`.
func (a *App) registerCompletion() error {
	cc := a.cfg.completion
	if cc == nil {
		return nil
	}

	parentOpts := []CommandOption{
		WithDescription("Generate shell completion scripts"),
		WithFlag("output", "",
			WithShort('o'),
			WithUsage("write to `file` instead of stdout"),
			WithPersistent(),
		),
	}
	if cc.hidden {
		parentOpts = append(parentOpts, WithHidden())
	}
	parent, err := a.Command(cc.commandName, parentOpts...)
	if err != nil {
		return fmt.Errorf("nabat: registerCompletion: %w", err)
	}

	write := func(c *Context, gen func(io.Writer) error) error {
		output, bindErr := BindAs[string](c, "output")
		if bindErr != nil || output == "" {
			return gen(a.io.Out)
		}
		// #nosec G304 -- output path is explicitly chosen by the user via --output.
		f, ferr := os.Create(output)
		if ferr != nil {
			return fmt.Errorf("creating output file: %w", ferr)
		}
		ferr = gen(f)
		if cerr := f.Close(); cerr != nil && ferr == nil {
			ferr = fmt.Errorf("closing output file: %w", cerr)
		}
		return ferr
	}

	shells := cc.shells
	if len(shells) == 0 {
		shells = []string{"bash", "zsh", "fish", "powershell"}
	}
	name := a.cfg.name
	root := a.root
	for _, sh := range shells {
		switch sh {
		case "bash":
			_, err = parent.Command("bash",
				WithDescription("Generate bash completion script"),
				WithLongDescription("Generate the autocompletion script for bash."),
				WithExample(bashCompletionInstructions(name)),
				WithRun(func(c *Context) error {
					return write(c, root.GenBashCompletion)
				}),
			)
		case "zsh":
			_, err = parent.Command("zsh",
				WithDescription("Generate zsh completion script"),
				WithLongDescription("Generate the autocompletion script for zsh."),
				WithExample(zshCompletionInstructions(name)),
				WithRun(func(c *Context) error {
					return write(c, root.GenZshCompletion)
				}),
			)
		case "fish":
			_, err = parent.Command("fish",
				WithDescription("Generate fish completion script"),
				WithLongDescription("Generate the autocompletion script for fish."),
				WithExample(fishCompletionInstructions(name)),
				WithRun(func(c *Context) error {
					return write(c, func(w io.Writer) error {
						return root.GenFishCompletion(w, true)
					})
				}),
			)
		case "powershell":
			_, err = parent.Command("powershell",
				WithDescription("Generate PowerShell completion script"),
				WithLongDescription("Generate the autocompletion script for PowerShell."),
				WithExample(powershellCompletionInstructions(name)),
				WithRun(func(c *Context) error {
					return write(c, root.GenPowerShellCompletionWithDesc)
				}),
			)
		}
		if err != nil {
			return fmt.Errorf("nabat: registerCompletion: %w", err)
		}
	}
	return nil
}

func bashCompletionInstructions(name string) string {
	return strings.Join([]string{
		"# Load in the current session:",
		fmt.Sprintf("source <(%s completion bash)", name),
		"",
		"# Load for every new session — add to ~/.bashrc:",
		fmt.Sprintf(`echo 'source <(%s completion bash)' >> ~/.bashrc`, name),
		"",
		"# Save directly to a file (system-wide):",
		fmt.Sprintf("%s completion bash --output /etc/bash_completion.d/%s", name, name),
	}, "\n")
}

func zshCompletionInstructions(name string) string {
	return strings.Join([]string{
		"# Load in the current session:",
		fmt.Sprintf("source <(%s completion zsh)", name),
		"",
		"# Load for every new session — execute once:",
		`echo "autoload -U compinit; compinit" >> ~/.zshrc`,
		fmt.Sprintf(`%s completion zsh --output "${fpath[1]}/_%s"`, name, name),
		"",
		"# Or add to ~/.zshrc:",
		fmt.Sprintf(`echo 'source <(%s completion zsh)' >> ~/.zshrc`, name),
	}, "\n")
}

func fishCompletionInstructions(name string) string {
	return strings.Join([]string{
		"# Load in the current session:",
		fmt.Sprintf("%s completion fish | source", name),
		"",
		"# Load for every new session — execute once:",
		fmt.Sprintf("%s completion fish --output ~/.config/fish/completions/%s.fish", name, name),
	}, "\n")
}

func powershellCompletionInstructions(name string) string {
	return strings.Join([]string{
		"# Load in the current session:",
		fmt.Sprintf("%s completion powershell | Out-String | Invoke-Expression", name),
		"",
		"# Load for every new session — add to your PowerShell profile ($PROFILE):",
		fmt.Sprintf("Add-Content $PROFILE \"`n%s completion powershell | Out-String | Invoke-Expression\"", name),
		"",
		"# Or save to a file and dot-source it from $PROFILE:",
		fmt.Sprintf(`%s completion powershell --output "%s.ps1"`, name, name),
		fmt.Sprintf(`Add-Content $PROFILE ". ${PSScriptRoot}\\%s.ps1"`, name),
	}, "\n")
}
