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
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"nabat.dev/nabat"
)

// NewIO returns an [nabat.IOStreams] backed by three *[bytes.Buffer]
// values for unit tests. All three streams report as non-TTY; use
// [NewTTYIO] for the interactive path.
func NewIO() (*nabat.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return nabat.NewIO(in, out, errOut), in, out, errOut
}

// NewTTYIO is like [NewIO] but reports all three streams as terminals.
func NewTTYIO() (*nabat.IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	ios, in, out, errOut := NewIO()
	ios.SetStdinTTY(true)
	ios.SetStdoutTTY(true)
	ios.SetStderrTTY(true)
	return ios, in, out, errOut
}

type runConfig struct {
	ctx    context.Context
	env    map[string]string
	dir    string
	dirSet bool
}

// RunOption configures [Run].
type RunOption func(*runConfig)

// WithContext sets the context used by [Run].
func WithContext(ctx context.Context) RunOption {
	return func(c *runConfig) {
		c.ctx = ctx
	}
}

// WithEnvVars sets process environment variables for this run (not
// [nabat.WithEnv] wiring). With [Run] and a non-nil tb, uses
// [testing.TB.Setenv] (do not call [testing.T.Parallel] first). With a
// nil tb, restores when [Run] returns. Rejected by [RunParallel].
func WithEnvVars(values map[string]string) RunOption {
	return func(c *runConfig) {
		c.env = values
	}
}

// WithDir sets the virtual working directory for this run. [RunParallel]
// accepts it. An empty dir is an error. A relative dir is made absolute
// on the calling goroutine against the process working directory.
func WithDir(dir string) RunOption {
	return func(c *runConfig) {
		c.dir = dir
		c.dirSet = true
	}
}

// Run executes app with args and returns any error from [nabat.App.RunArgs].
// Pass a non-nil tb so failures are attributed; tb may be nil for examples.
// Use [RunParallel] after [testing.T.Parallel].
//
// Errors include those from [nabat.App.RunArgs] and from saving or
// restoring process env when tb is nil.
//
// Panics if [WithEnvVars] is supplied and the test has already called
// [testing.T.Parallel] ([testing.TB.Setenv] panics in that case).
func Run(tb testing.TB, app *nabat.App, args []string, opts ...RunOption) error {
	return runInternal(tb, app, args, false, opts...)
}

// RunParallel is like [Run] but safe after [testing.T.Parallel].
// [WithEnvVars] is rejected; set env via [testing.TB.Setenv] before
// Parallel, or use [Run] serially. Do not share an [nabat.App] across
// parallel [Run] or [RunParallel] calls; [nabat.App.RunArgs] mutates
// cobra args. Other failures come from [nabat.App.RunArgs].
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

	if cfg.dirSet {
		dir, err := resolveDir(cfg.dir)
		if err != nil {
			return err
		}
		cfg.ctx = nabat.WithDir(cfg.ctx, dir)
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

// CaptureResult holds the captured output from [Capture].
type CaptureResult struct {
	// Stdout is the app's stdout buffer when the app was built with [NewIO]
	// or [NewTTYIO]; nil otherwise.
	Stdout *bytes.Buffer
	// Stderr is the app's stderr buffer when the app was built with [NewIO]
	// or [NewTTYIO]; nil otherwise.
	Stderr *bytes.Buffer
	// Err is the error returned by the command run.
	Err error
}

// Capture runs app with args and returns captured stdout/stderr plus any
// error. The app must use [NewIO] or [NewTTYIO] via [nabat.WithIO].
//
// Example:
//
//	io, _, _, _ := nabattest.NewIO()
//	app := nabat.MustNew("myctl", nabat.WithIO(io), ...)
//	got := nabattest.Capture(t, app, []string{"plan"})
//	require.NoError(t, got.Err)
//	require.Contains(t, got.Stdout.String(), "...")
func Capture(tb testing.TB, app *nabat.App, args []string, opts ...RunOption) CaptureResult {
	if tb != nil {
		tb.Helper()
	}
	result := CaptureResult{}
	if app != nil {
		if ios := app.IO(); ios != nil {
			if b, ok := ios.RawOut().(*bytes.Buffer); ok {
				result.Stdout = b
			}
			if b, ok := ios.RawErrOut().(*bytes.Buffer); ok {
				result.Stderr = b
			}
		}
	}
	result.Err = Run(tb, app, args, opts...)
	return result
}

// Context returns a [*nabat.Context] for helpers that take a Context
// directly. It wraps [nabat.App.NewBareContext] and binds the context to
// the test lifecycle when tb is non-nil. Pass [WithContext] to set the
// underlying [context.Context]. Pass [WithDir] to set the virtual working
// directory; an empty or unresolvable dir calls tb.Fatal when tb is
// non-nil, and is skipped when tb is nil.
//
// Example:
//
//	io, _, out, _ := nabattest.NewIO()
//	app := nabat.MustNew("myctl", nabat.WithIO(io))
//	c := nabattest.Context(t, app)
//	c.Fields([]nabat.Field{{Key: "K", Value: "V"}}).Print()
//	require.Contains(t, out.String(), "K")
func Context(tb testing.TB, app *nabat.App, opts ...RunOption) *nabat.Context {
	if tb != nil {
		tb.Helper()
	}
	cfg := runConfig{ctx: context.Background()}
	if tb != nil {
		cfg.ctx = tb.Context()
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	c := app.NewBareContext()
	if c != nil {
		c.SetContext(cfg.ctx)
		if cfg.dirSet {
			dir, err := resolveDir(cfg.dir)
			if err != nil {
				if tb != nil {
					tb.Fatal(err)
				}
				return c
			}
			c.SetDir(dir)
		}
	}
	return c
}

// resolveDir absolutizes dir for [WithDir]. Empty dir is an error.
func resolveDir(dir string) (string, error) {
	if dir == "" {
		return "", errors.New("nabattest: WithDir: dir cannot be empty")
	}
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("nabattest: WithDir: resolve relative path: %w", err)
		}
		dir = filepath.Join(wd, dir)
	}
	return filepath.Clean(dir), nil
}

// setProcessEnv mutates process env directly and returns a restore func.
// Used only when [Run] has no [testing.TB] (examples); tests go through
// [testing.TB.Setenv] so a caller of [WithEnvVars] can also use
// [testing.T.Parallel].
func setProcessEnv(values map[string]string) (func(), error) {
	restore := make([]func(), 0, len(values))
	undo := func() {
		for _, fn := range slices.Backward(restore) {
			fn()
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
