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

// Command theme is a Nabat example showing all twelve built-in themes.
// Set NABAT_THEME to any theme name before running:
//
//	NABAT_THEME=nord go run ./examples/theme show
//	NABAT_THEME=gruvbox go run ./examples/theme show
//	NABAT_THEME=catppuccin-mocha go run ./examples/theme show
package main

import (
	"context"
	"log"
	"os"

	"nabat.dev/nabat"
	"nabat.dev/theme"
)

func main() {
	themeName := os.Getenv("NABAT_THEME")
	if themeName == "" {
		themeName = theme.Default
	}

	app, err := nabat.New("theme",
		nabat.WithTheme(themeName),
		nabat.WithDescription("Theme showcase for Nabat"),

		nabat.WithCommand("show",
			nabat.WithDescription("Print a sample of all output types"),
			nabat.WithRun(func(c *nabat.Context) error {
				c.Success("deployment complete", "env", "production", "replicas", 3)
				c.Warn("high memory usage", "threshold", "80%", "current", "87%")
				c.Error("connection refused", "host", "db-primary", "port", 5432)
				c.Info("retrying", "attempt", 2, "delay", "500ms")

				c.Table(
					[]string{"Service", "Status", "Uptime"},
					[][]string{
						{"web", "running", "14d"},
						{"db", "running", "14d"},
						{"cache", "degraded", "2h"},
					},
					nabat.WithTableBorder(nabat.BorderASCII()),
				)

				c.List([]string{
					"Adaptive args: CLI → env → prompt → default",
					"Structured output: table, tree, JSON, YAML, TOML",
					"Twelve built-in themes",
				}, nabat.WithListEnumerator(nabat.ListBullet))

				return nil
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
