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

// Command deploy is a richer Nabat example showing typed args, persistent
// flags, env-var resolution, interactive confirmation, a spinner, a progress
// bar, multiple output formats, structured logging, and the manpage extension
// wired onto a realistic deployment workflow.
//
// Try it:
//
//	go run ./examples/deploy configure           # multi-page wizard
//	go run ./examples/deploy deploy staging      # table output (default)
//	go run ./examples/deploy deploy staging -o json
//	go run ./examples/deploy deploy staging -o tree
//	go run ./examples/deploy deploy --help
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"nabat.dev/logging"
	"nabat.dev/manpage"
	"nabat.dev/nabat"
	"nabat.dev/theme"
)

func main() {
	type deployArgs struct {
		Environment string `nabat:"environment"`
		Replicas    int    `nabat:"replicas"`
		Tag         string `nabat:"tag"`
		Yes         bool   `nabat:"yes"`
		Output      string `nabat:"output"`
	}
	type configResult struct {
		name    string
		env     string
		proceed bool
	}
	type podStatus struct {
		Pod    string `json:"pod"`
		Status string `json:"status"`
		Ready  string `json:"ready"`
		Image  string `json:"image"`
	}

	makePods := func(env, tag string, replicas int) []podStatus {
		pods := make([]podStatus, 0, replicas)
		for i := range replicas {
			pods = append(pods, podStatus{
				Pod:    fmt.Sprintf("%s-web-%d", env, i+1),
				Status: "Running",
				Ready:  "1/1",
				Image:  fmt.Sprintf("app:%s", tag),
			})
		}
		return pods
	}

	app, err := nabat.New("deployctl",
		nabat.WithTheme(theme.Dracula),
		nabat.WithDescription("Deploy demo built with Nabat"),
		nabat.WithFlag("verbose", false,
			nabat.WithShort('v'),
			nabat.WithPersistent(),
			nabat.WithEnv("verbose"),
		),
		nabat.WithVersion("0.1.0", nabat.WithoutVersionShorthand()),
		nabat.WithCompletion(),
		nabat.WithExtension(manpage.New()),
		nabat.WithExtension(logging.New(logging.WithVerboseFlag("verbose"), logging.WithTimestamp())),

		// "configure" walks the user through a two-page identity + environment
		// form, then prints a summary and the exact command to run next.
		nabat.WithCommand("configure",
			nabat.WithDescription("Interactive configuration wizard"),
			nabat.WithRun(func(c *nabat.Context) error {
				var result configResult
				formOpts := []nabat.FormOption{
					nabat.WithFormGroup(
						nabat.WithGroupTitle("Identity"),
						nabat.WithGroupDescription("Who is deploying?"),
						nabat.WithFormNote(
							"Pre-deploy checklist",
							"- Database backup verified\n- Staging smoke test passed\n- On-call notified",
						),
						nabat.WithFormField(&result.name, "Name", "Your name for the deploy log",
							nabat.WithHint("alice"),
							nabat.WithDefault(""),
						),
					),
					nabat.WithFormGroup(
						nabat.WithGroupTitle("Deployment"),
						nabat.WithSelectField(&result.env, "Environment", "Target environment",
							[]string{"staging", "production"},
							"staging",
						),
						nabat.WithFormField(&result.proceed, "Confirm?", "This will start the deploy",
							nabat.WithAffirmative("Deploy"),
							nabat.WithNegative("Abort"),
							nabat.WithDefault(false),
						),
					),
				}
				if os.Getenv("ACCESSIBLE") != "" {
					formOpts = append(formOpts, nabat.WithFormAccessible())
				}
				if err := c.Form(formOpts...); err != nil {
					return err
				}
				if !result.proceed {
					c.Warn("deploy aborted")
					return nil
				}
				c.Success("configuration saved",
					"deployer", result.name,
					"environment", result.env,
				)
				c.Table(
					[]string{"Setting", "Value"},
					[][]string{
						{"Deployer", result.name},
						{"Environment", result.env},
					},
					nabat.WithTableBorder(nabat.BorderASCII()),
				)
				c.Info("next step", "run", fmt.Sprintf("deployctl deploy %s", result.env))
				return nil
			}),
		),

		// "deploy" resolves the environment from CLI → env var → prompt, confirms
		// unless --yes is set, spins up a connection, rolls out pods with a
		// progress bar, then prints the pod status in the requested format.
		nabat.WithCommand("deploy",
			nabat.WithDescription("Deploy an application"),
			nabat.WithGroup("Operations"),
			nabat.WithExample(`# Production — pin a specific image and scale up before the release:
deployctl deploy production --replicas 3 --tag v1.2.0

# Staging — quick smoke-test with defaults:
deployctl deploy staging

# CI — set replicas and tag via env vars, skip the prompt:
export DEPLOYCTL_REPLICAS=4
export DEPLOY_TAG=canary
deployctl deploy staging --yes
`),
			nabat.WithSelectArg("environment", "", []string{"staging", "production"},
				nabat.WithRequired(),
				nabat.WithEnv("environment"),
				nabat.WithPrompt("Target environment", "Where to deploy",
					nabat.WithHint("staging"),
				),
			),
			nabat.WithFlag("replicas", 2,
				nabat.WithEnv("replicas"),
				nabat.WithUsage("Number of pod replicas"),
			),
			nabat.WithFlag("tag", "latest",
				nabat.WithEnv("tag"),
				nabat.WithEnvAlias("DEPLOY_TAG"),
				nabat.WithUsage("Container image tag"),
			),
			nabat.WithFlag("yes", false,
				nabat.WithShort('y'),
				nabat.WithUsage("Skip confirmation prompt"),
			),
			nabat.WithSelectFlag("output", "table", []string{"table", "json", "tree"},
				nabat.WithShort('o'),
				nabat.WithUsage("Output format (table, json, tree)"),
			),
			nabat.WithRun(func(c *nabat.Context) error {
				var args deployArgs
				if err := c.Bind(&args); err != nil {
					return err
				}
				c.Logger().Debug("starting deployment",
					"env", args.Environment,
					"replicas", args.Replicas,
					"tag", args.Tag,
				)

				if !args.Yes {
					ok, err := c.Confirm(
						fmt.Sprintf("Deploy %s to %s?", args.Tag, args.Environment),
						nabat.WithDefault(false),
					)
					if err != nil {
						return err
					}
					if !ok {
						c.Warn("deploy aborted")
						return nil
					}
				}

				if err := c.Spinner("Connecting to cluster...", func() error {
					time.Sleep(800 * time.Millisecond)
					return nil
				}, nabat.WithSpinnerType(nabat.SpinnerDots())); err != nil {
					return err
				}

				bar, err := c.ProgressBar(args.Replicas, nabat.WithProgressBarWidth(40))
				if err != nil {
					return err
				}
				for i := range args.Replicas {
					time.Sleep(300 * time.Millisecond)
					bar.Set(i + 1)
				}
				bar.Done()

				c.Success("deployment complete",
					"environment", args.Environment,
					"replicas", args.Replicas,
					"tag", args.Tag,
				)

				pods := makePods(args.Environment, args.Tag, args.Replicas)
				switch args.Output {
				case "json":
					return c.JSON(pods)
				case "tree":
					nodes := make([]nabat.TreeNode, 0, len(pods))
					for _, p := range pods {
						nodes = append(nodes, nabat.TreeNode{
							Value: p.Pod,
							Children: []nabat.TreeNode{
								{Value: "status: " + p.Status},
								{Value: "ready: " + p.Ready},
								{Value: "image: " + p.Image},
							},
						})
					}
					c.Tree(args.Environment, nodes,
						nabat.WithTreeEnumerator(nabat.TreeDefaultEnumerator()),
					)
				default: // table
					rows := make([][]string, 0, len(pods))
					for _, p := range pods {
						rows = append(rows, []string{p.Pod, p.Status, p.Ready, p.Image})
					}
					c.Table(
						[]string{"Pod", "Status", "Ready", "Image"},
						rows,
						nabat.WithTableBorder(nabat.BorderASCII()),
					)
				}
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
