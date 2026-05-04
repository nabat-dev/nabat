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
	"strings"

	"charm.land/lipgloss/v2"

	"nabat.dev/theme"
)

// Success writes msg to stderr with a success symbol and optional key/value
// pairs.
//
// Status and confirmation messages travel on the diagnostics stream so a
// piped stdout (e.g. `mycli deploy --json | jq`) carries only the command's
// product. The "product" itself goes through [Context.Print], [Context.JSON],
// [Context.Table], etc., which write to [Context.IO.Out].
//
// Example:
//
//	c.Success("deployed", "environment", env)
func (c *Context) Success(msg string, args ...any) {
	c.writeUserMessage(c.io.ErrOut, c.app.Theme().Style(theme.StatusSuccess), "✓", msg, args...)
}

// Warn writes msg to stderr with a warning symbol and optional key/value pairs.
//
// Diagnostics go to stderr (POSIX) so they remain visible when the command's
// stdout is piped or redirected (e.g. `mycli deploy | jq`).
//
// Example:
//
//	c.Warn("slow query", "ms", elapsed)
func (c *Context) Warn(msg string, args ...any) {
	c.writeUserMessage(c.io.ErrOut, c.app.Theme().Style(theme.StatusWarning), "⚠", msg, args...)
}

// Error writes msg to stderr with an error symbol and optional key/value pairs.
//
// Diagnostics go to stderr (POSIX) so error messages do not corrupt downstream
// pipeline data on stdout.
//
// Example:
//
//	c.Error("deploy blocked", "reason", err)
func (c *Context) Error(msg string, args ...any) {
	c.writeUserMessage(c.io.ErrOut, c.app.Theme().Style(theme.StatusError), "✗", msg, args...)
}

// Info writes msg to stderr with an info symbol and optional key/value pairs.
//
// Info messages are status narrative ("retrying", "connecting", "skipped")
// and so travel on the diagnostics stream alongside [Context.Success],
// [Context.Warn], and [Context.Error]. This keeps stdout reserved for the
// command's product, matching clig.dev's "send status output to stderr"
// guidance and so a piped stdout stays uncorrupted.
//
// Example:
//
//	c.Info("retrying", "attempt", n)
func (c *Context) Info(msg string, args ...any) {
	c.writeUserMessage(c.io.ErrOut, c.app.Theme().Style(theme.StatusInfo), "•", msg, args...)
}

// Print writes msg as plain text to [Context.IO.Out] without an implicit newline.
// Use [Context.Println] for a newline-terminated line.
func (c *Context) Print(msg string) {
	w := writer{w: c.io.Out}
	w.printf("%s", msg)
}

// Println writes msg followed by a newline to [Context.IO.Out].
//
// Example:
//
//	c.Println("plain status line")
func (c *Context) Println(msg string) {
	w := writer{w: c.io.Out}
	w.println(msg)
}

// Printf writes formatted text to [Context.IO.Out] without an implicit newline.
func (c *Context) Printf(format string, args ...any) {
	w := writer{w: c.io.Out}
	w.printf(format, args...)
}

func (c *Context) writeUserMessage(w io.Writer, style lipgloss.Style, symbol, msg string, args ...any) {
	rt := c.app.Theme()
	labelStyle := rt.Style(theme.AccentPrimary)
	valueStyle := rt.Style(theme.TextPrimary)

	builder := strings.Builder{}
	builder.WriteString(style.Render(symbol))
	builder.WriteString(" ")
	builder.WriteString(msg)
	// k/v args follow log/slog semantics: pairs of (string-key, any-value).
	// Empty-string keys are silently skipped; non-string keys use !BADKEY to
	// match slog's convention; a trailing unpaired key renders as key=!MISSING.
	for i := 0; i < len(args); i += 2 {
		key, ok := args[i].(string)
		if ok && key == "" {
			continue
		}
		if !ok {
			key = "!BADKEY"
		}
		builder.WriteString(" ")
		builder.WriteString(labelStyle.Render(key + "="))
		if i+1 >= len(args) {
			builder.WriteString(valueStyle.Render("!MISSING"))
			break
		}
		builder.WriteString(valueStyle.Render(fmt.Sprint(args[i+1])))
	}
	out := writer{w: w}
	out.println(builder.String())
}
