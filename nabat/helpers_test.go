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

// Internal test helpers for package nabat tests.
//
// These mirror the API in nabattest but are defined here to avoid an import
// cycle: internal test files (package nabat) cannot import nabattest because
// nabattest itself imports nabat. External test packages (package nabat_test)
// should import "nabat.dev/nabattest" directly.

import (
	"bytes"
	"testing"
)

// testIO returns an IOStreams bundle backed by three independent buffers.
// Use it for every internal test that needs to assert on captured output:
// separate stdout and stderr buffers reveal stream-routing bugs that a
// merged buffer would hide.
func testIO() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	return newTestIO()
}

// runConfig holds optional knobs for [Run]. Fields are populated by
// [runOption] values and consumed by [Run]. Currently empty; the type is
// retained so adding knobs (timeouts, ctx, etc.) does not change the [Run]
// signature.
type runConfig struct{}

// runOption configures [Run].
type runOption func(*runConfig)

// Run executes app with args as the command line and returns any error.
// The signature mirrors [nabattest.Run] so internal and external test files
// read the same way.
func Run(tb testing.TB, app *App, args []string, opts ...runOption) error {
	tb.Helper()
	var cfg runConfig
	for _, o := range opts {
		o(&cfg)
	}
	return app.RunArgs(tb.Context(), args...)
}
