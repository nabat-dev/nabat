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
	"context"
	"log/slog"
	"time"

	"github.com/spf13/cobra"
)

// Context is the per-invocation runtime passed to each [RunFunc].
//
// It holds the resolved values for declared positional args and flags, the raw
// positional argument strings, and the [App] that created the command. Use
// [Context.Bind] (or [BindAs]) to read resolved values into a struct or a single
// typed name.
//
// Context implements [context.Context] by delegating to the context passed to
// [App.Run] (or [context.Background] if none was set on the Cobra command).
//
// Context is not safe for concurrent use. Each command invocation receives its own
// Context; do not share it across goroutines or retain it after the [RunFunc]
// returns.
type Context struct {
	ctx context.Context
	app *App
	cmd *cobra.Command

	// io is the same [IOStreams] bundle held on the parent [App].
	// Read it via [Context.IO]. Convenience methods on Context (Print, Success,
	// Warn, Error, JSON, ProgressBar, ...) route through stdout and stderr and
	// inherit their sticky-error and color policy.
	io *IOStreams

	logger          *slog.Logger
	args            []string
	passthroughArgs []string
	hasPassthrough  bool
	values          map[string]any
	set             map[string]bool // true = value came from arg/env/prompt, not a default
	interactive     bool
}

var _ context.Context = (*Context)(nil)

// Deadline reports the current deadline, if any.
func (c *Context) Deadline() (time.Time, bool) {
	if c == nil || c.ctx == nil {
		return time.Time{}, false
	}
	return c.ctx.Deadline()
}

// Done returns a channel closed when work should be canceled.
func (c *Context) Done() <-chan struct{} {
	if c == nil || c.ctx == nil {
		return nil
	}
	return c.ctx.Done()
}

// Err reports why Done was closed.
func (c *Context) Err() error {
	if c == nil || c.ctx == nil {
		return nil
	}
	return c.ctx.Err()
}

// Value looks up a value in the underlying Go context.
func (c *Context) Value(key any) any {
	if c == nil || c.ctx == nil {
		return nil
	}
	return c.ctx.Value(key)
}

// Context returns the underlying [context.Context] carried by this [Context].
// Use it as the parent when building a derived context for [Context.SetContext]:
//
//	enriched := context.WithValue(c.Context(), key, val)
//	c.SetContext(enriched)
//
// Using c itself as the parent would create an infinite delegation cycle;
// always use Context() as the parent.
func (c *Context) Context() context.Context {
	if c == nil || c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// SetContext replaces the underlying [context.Context] carried by this
// [Context]. Use it in [App.OnPreRun] or [Command.OnPreRun] hooks to
// propagate request-scoped values to downstream hooks and the command's
// [RunFunc].
//
// The typical pattern pairs SetContext with [Context.Context] and a
// package-level WithContext/FromContext accessor pair:
//
//	app.OnPreRun(func(c *Context) error {
//	    rt := runtime.New()
//	    c.SetContext(runtime.WithContext(c.Context(), rt))
//	    return nil
//	})
//
// Always pass [Context.Context] as the parent when deriving a new context.
// Passing c itself creates an infinite delegation cycle.
func (c *Context) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// Args returns a copy of the positional arguments before any "--".
// The returned slice is always non-nil; iterate or take len without a nil check.
// Arguments after "--" are accessible via [Context.Passthrough].
func (c *Context) Args() []string {
	out := make([]string, 0, len(c.args))
	return append(out, c.args...)
}

// Passthrough returns a copy of the tokens that appeared after "--", or an empty
// slice when none did. The returned slice is always non-nil. Use
// [Context.HasPassthrough] to tell whether "--" appeared on the command line at
// all (including with no tokens after it). Declare passthrough support on the
// command with [WithPassthrough] to document it in help output.
func (c *Context) Passthrough() []string {
	out := make([]string, 0, len(c.passthroughArgs))
	return append(out, c.passthroughArgs...)
}

// HasPassthrough reports whether "--" appeared on the command line, even when
// no tokens followed it. Pair with [Context.Passthrough] to read the tokens.
func (c *Context) HasPassthrough() bool {
	return c.hasPassthrough
}

// IsInteractive reports whether stdin/stdout are both terminals.
func (c *Context) IsInteractive() bool {
	return c.interactive
}

// Explicit reports whether the named arg or flag was provided via the command
// line, environment variable, or interactive prompt, as opposed to using only a
// registered default.
// Use it when handler logic must distinguish "user supplied" from "default".
func (c *Context) Explicit(name string) bool {
	if c == nil {
		return false
	}
	return c.set[name]
}

// IO returns the stdin/stdout/stderr bundle for this invocation. The result
// matches [App.IO] on the parent app and is never nil during a normal [RunFunc].
func (c *Context) IO() *IOStreams {
	if c == nil {
		return nil
	}
	return c.io
}

// discardLogger is a no-op logger returned by [Context.Logger] when no logger
// has been installed. Returning a discard logger keeps library code that grabs
// c.Logger() from accidentally writing to the process-wide slog default.
var discardLogger = slog.New(slog.DiscardHandler)

// Logger returns the structured logger for this command invocation.
//
// The logger is sourced from (in order of precedence):
//   - the logging plugin, when installed via [WithExtension]
//   - [WithLogger] (or [App.SetLogger]) when used at construction time
//   - a discard logger that silently drops all records (the default)
func (c *Context) Logger() *slog.Logger {
	if c.logger != nil {
		return c.logger
	}
	return discardLogger
}

func (a *App) newContext(cmd *cobra.Command, args []string) (*Context, error) {
	goCtx := cmd.Context()
	if goCtx == nil {
		goCtx = context.Background()
	}

	var preDash, postDash []string
	hasPassthrough := false
	if n := cmd.ArgsLenAtDash(); n >= 0 {
		hasPassthrough = true
		preDash = append([]string(nil), args[:n]...)
		postDash = append([]string(nil), args[n:]...)
	} else {
		preDash = append([]string(nil), args...)
	}

	ctx := &Context{
		ctx:             goCtx,
		app:             a,
		cmd:             cmd,
		io:              a.io,
		args:            preDash,
		passthroughArgs: postDash,
		hasPassthrough:  hasPassthrough,
		values:          map[string]any{},
		set:             map[string]bool{},
		interactive:     a.io.CanPrompt(),
	}

	if err := a.resolveArgs(ctx); err != nil {
		return nil, err
	}
	if err := a.resolveFlags(ctx); err != nil {
		return nil, err
	}
	if a.cfg.logger != nil {
		ctx.logger = a.cfg.logger
	}
	return ctx, nil
}
