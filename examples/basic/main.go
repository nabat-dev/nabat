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

// Command basic is a minimal Nabat example: a single subcommand with one
// positional arg, an interactive prompt fallback, the built-in version and
// shell completion features, the manpage extension, and rendered Markdown
// output. Run `go run ./examples/basic` to try it.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"nabat.dev/manpage"
	"nabat.dev/nabat"
)

func main() {
	type helloArgs struct {
		Name string `nabat:"name"`
	}

	app, err := nabat.New("basic",
		nabat.WithDescription("Basic Nabat example"),
		nabat.WithVersion("0.1.0"),
		nabat.WithCompletion(),
		nabat.WithExtension(manpage.New()),

		nabat.WithCommand("hello",
			nabat.WithDescription("Print a greeting"),
			nabat.WithArg("name", "world",
				nabat.WithEnv("name"),
				nabat.WithPrompt("Your name", "",
					nabat.WithHint("e.g. alice"),
				),
			),
			nabat.WithRun(func(c *nabat.Context) error {
				var args helloArgs
				if err := c.Bind(&args); err != nil {
					return err
				}
				c.Success("hello", "name", args.Name)
				return c.Markdown(fmt.Sprintf(
					"## Hello, **%s**!\n\nWelcome to **Nabat** — adaptive CLI for Go.",
					args.Name,
				))
			}),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	if err = app.Run(context.Background()); err != nil {
		os.Exit(1)
	}
}
