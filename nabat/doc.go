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

// Package nabat is a CLI framework for Go built on Cobra.
//
// Build an [App] with [New] and [WithCommand], or register at runtime with
// [App.Command]. Handlers from [WithRun] receive a [Context].
// Args resolve as CLI, env ([WithEnv]), prompt ([WithPrompt], TTY), then
// default; flags as CLI, env, then default.
//
//	app, err := nabat.New("myctl",
//	    nabat.WithCommand("deploy", nabat.WithRun(deploy)),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	_ = app.Run(context.Background())
//
// Help is on by default; see examples/ for fuller programs.
package nabat
