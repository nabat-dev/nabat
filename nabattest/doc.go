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

// Package nabattest provides test helpers for Nabat CLI applications.
//
// Import this package in test files to get ergonomic helpers for executing
// apps with explicit argument lists and for constructing buffer-backed IO
// bundles for output capture.
//
//	import (
//	    "nabat.dev/nabat"
//	    "nabat.dev/nabattest"
//	)
//
//	func TestDeploy(t *testing.T) {
//	    io, _, out, _ := nabattest.NewIO()
//	    app := nabat.MustNew("myctl",
//	        nabat.WithIO(io),
//	        nabat.WithCommand("deploy", nabat.WithRun(handler)),
//	    )
//	    require.NoError(t, nabattest.Run(t, app, []string{"deploy", "--env=staging"}))
//	    require.Contains(t, out.String(), "deployed")
//	}
package nabattest
