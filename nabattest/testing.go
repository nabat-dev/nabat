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

package nabattest

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"nabat.dev/nabat"
)

// NewIO returns an [nabat.IOStreams] whose three streams are *[bytes.Buffer]
// values, suitable for use in unit tests. The buffers are returned alongside
// the IOStreams so tests can assert on captured output without rummaging
// through internals.
//
// All three streams report as non-TTY by default. Use [NewTTYIO] when a test
// exercises the interactive code path.
func NewIO() (*nabat.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return nabat.NewIO(in, out, errOut), in, out, errOut
}

// NewTTYIO is like [NewIO] but reports all three streams as terminals. Use it
// when a test exercises the interactive code path against buffer-backed streams.
func NewTTYIO() (*nabat.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	ios, in, out, errOut := NewIO()
	ios.SetStdinTTY(true)
	ios.SetStdoutTTY(true)
	ios.SetStderrTTY(true)
	return ios, in, out, errOut
}

type runConfig struct {
	ctx context.Context
	env map[string]string
}

// RunOption configures [Run].
type RunOption func(*runConfig)

// WithContext sets the context used by [Run].
func WithContext(ctx context.Context) RunOption {
	return func(c *runConfig) {
		c.ctx = ctx
	}
}

// WithEnvVars sets process environment variables for this run. The name makes
// it obvious this manipulates process env, not CLI flag/env wiring declared
// by [nabat.WithEnv].
//
// Compatibility:
//   - With [Run] and a non-nil [testing.TB], values are applied via
//     [testing.TB.Setenv] and restored at the end of the test. The test must
//     NOT call [testing.T.Parallel] before [Run]; see [Run] for the panic
//     contract.
//   - With [Run] and a nil tb (examples), values are restored when [Run]
//     returns.
//   - With [RunParallel], passing this option fails fast with a clear error;
//     parallel tests must set env before calling [testing.T.Parallel].
func WithEnvVars(values map[string]string) RunOption {
	return func(c *runConfig) {
		c.env = values
	}
}

// Run executes app with args as the command line and returns any error.
// It is equivalent to [nabat.App.RunArgs] with test helper attribution.
// Pass a non-nil tb so failures are attributed in test output; tb may be nil
// for examples.
//
// Use [RunParallel] for tests that have called [testing.T.Parallel].
//
// Errors:
//   - any error returned by [nabat.App.RunArgs] (see its documentation)
//   - errors from saving / restoring process env when tb is nil
//
// Panics if [WithEnvVars] is supplied and the test has already called
// [testing.T.Parallel] (the underlying [testing.TB.Setenv] panics in that
// case so values cannot leak between parallel tests).
func Run(tb testing.TB, app *nabat.App, args []string, opts ...RunOption) error {
	return runInternal(tb, app, args, false, opts...)
}

// RunParallel is the parallel-safe sibling of [Run]. The test may call
// [testing.T.Parallel] before invoking RunParallel.
//
// [WithEnvVars] is rejected at call time with a clear error so the panic
// from [testing.TB.Setenv]-after-[testing.T.Parallel] cannot happen. Set
// process environment with [testing.TB.Setenv] before calling
// [testing.T.Parallel] (or use [Run] in serial tests) when env wiring is
// required.
//
// Errors:
//   - "nabattest: RunParallel does not support WithEnvVars; set env before
//     t.Parallel or call nabattest.Run instead"
//   - any error returned by [nabat.App.RunArgs]
func RunParallel(tb testing.TB, app *nabat.App, args []string, opts ...RunOption) error {
	return runInternal(tb, app, args, true, opts...)
}

func runInternal(tb testing.TB, app *nabat.App, args []string, parallel bool, opts ...RunOption) error {
	if tb != nil {
		tb.Helper()
	}
	cfg := runConfig{ctx: context.Background()}
	if tb != nil {
		// Bind the run to the test's lifecycle so handlers blocked on
		// c.Context().Done() unwedge automatically when the test ends,
		// instead of leaking until the package timeout. WithContext still
		// overrides this for callers that need a custom context.
		cfg.ctx = tb.Context()
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	if parallel && len(cfg.env) > 0 {
		return errors.New("nabattest: RunParallel does not support WithEnvVars; set env before t.Parallel or call nabattest.Run instead")
	}

	if tb != nil {
		for k, v := range cfg.env {
			tb.Setenv(k, v)
		}
	} else {
		restore, err := setProcessEnv(cfg.env)
		if err != nil {
			return err
		}
		defer restore()
	}

	return app.RunArgs(cfg.ctx, args...)
}

// setProcessEnv mutates os env directly and returns a restore func. Used only
// when [Run] has no [testing.TB] (examples); tests go through [testing.TB.Setenv]
// so that a caller of [WithEnvVars] cannot also call [testing.T.Parallel].
func setProcessEnv(values map[string]string) (func(), error) {
	restore := make([]func(), 0, len(values))
	undo := func() {
		for i := len(restore) - 1; i >= 0; i-- {
			restore[i]()
		}
	}
	for k, v := range values {
		old, had := os.LookupEnv(k)
		if err := os.Setenv(k, v); err != nil {
			undo()
			return nil, err
		}
		restore = append(restore, func() {
			// Restore-time errors are unrecoverable here: the helper is
			// invoked from defer/cleanup paths after the test body has
			// completed, so there is no caller to surface the error to and
			// no useful recovery action. os.Setenv/Unsetenv only fail on
			// invalid keys, which would have already failed above.
			if had {
				_ = os.Setenv(k, old) //nolint:errcheck // see comment above
			} else {
				_ = os.Unsetenv(k) //nolint:errcheck // see comment above
			}
		})
	}
	return undo, nil
}
