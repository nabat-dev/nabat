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

package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// Handler is a styled [slog.Handler] that writes human-readable lines
// using a [Styles] value (typically from [FromTheme]). Output format is
// LEVL message plus optional key=value pairs. Safe for concurrent use,
// including handlers derived via [Handler.WithAttrs] and
// [Handler.WithGroup], which share the writer mutex.
type Handler struct {
	w      io.Writer
	level  *slog.LevelVar
	styles Styles
	ts     bool

	// mu guards w and styles. It is a pointer so [WithAttrs]/[WithGroup]
	// clones share one lock for the underlying writer.
	mu    *sync.Mutex
	group string
	attrs []slog.Attr
}

// HandlerOptions configures a [Handler].
type HandlerOptions struct {
	// Level is the minimum enabled level. When nil, [NewHandler] installs a
	// fresh [slog.LevelVar] at [slog.LevelInfo] (zero value of LevelVar).
	Level *slog.LevelVar
	// Styles controls level badges and key/value coloring. The zero value
	// renders without lipgloss styling.
	Styles Styles
	// Timestamp, when true, prefixes each line with a Kitchen-format clock.
	Timestamp bool
}

// NewHandler returns a [Handler] that writes styled log output to w.
//
// Panics if w is nil: a handler without a writer is a programmer error.
func NewHandler(w io.Writer, opts HandlerOptions) *Handler {
	if w == nil {
		panic("nabat/logging: NewHandler: writer is nil")
	}
	lv := opts.Level
	if lv == nil {
		lv = new(slog.LevelVar)
	}
	return &Handler{
		w:      w,
		level:  lv,
		styles: opts.Styles,
		ts:     opts.Timestamp,
		mu:     new(sync.Mutex),
	}
}

// Enabled reports whether records at level should be handled.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle formats and writes a log record.
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	styles := h.styles
	h.mu.Unlock()

	buf := make([]byte, 0, 128)

	buf = append(buf, levelBadge(styles, r.Level)...)
	buf = append(buf, ' ')

	if h.ts {
		buf = append(buf, r.Time.Format(time.Kitchen)...)
		buf = append(buf, ' ')
	}

	buf = append(buf, r.Message...)

	appendResolved := func(a slog.Attr) {
		buf = appendAttr(buf, styles, h.group, a)
	}
	for _, a := range h.attrs {
		appendResolved(a)
	}
	r.Attrs(func(a slog.Attr) bool {
		appendResolved(a)
		return true
	})

	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

// WithAttrs returns a derived handler that includes attrs on every record.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		w:      h.w,
		level:  h.level,
		styles: h.styles,
		ts:     h.ts,
		mu:     h.mu,
		group:  h.group,
		attrs:  append(cloneAttrs(h.attrs), attrs...),
	}
}

// WithGroup returns a derived handler that prefixes attribute keys with name.
func (h *Handler) WithGroup(name string) slog.Handler {
	g := name
	if h.group != "" {
		g = h.group + "." + name
	}
	return &Handler{
		w:      h.w,
		level:  h.level,
		styles: h.styles,
		ts:     h.ts,
		mu:     h.mu,
		group:  g,
		attrs:  cloneAttrs(h.attrs),
	}
}

func levelBadge(styles Styles, l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return styles.Error.String()
	case l >= slog.LevelWarn:
		return styles.Warn.String()
	case l >= slog.LevelInfo:
		return styles.Info.String()
	default:
		return styles.Debug.String()
	}
}

func appendAttr(buf []byte, styles Styles, prefix string, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}
		groupPrefix := prefix
		if a.Key != "" {
			if groupPrefix != "" {
				groupPrefix = groupPrefix + "." + a.Key
			} else {
				groupPrefix = a.Key
			}
		}
		for _, ga := range attrs {
			buf = appendAttr(buf, styles, groupPrefix, ga)
		}
		return buf
	}

	key := a.Key
	if prefix != "" {
		key = prefix + "." + key
	}
	buf = append(buf, ' ')
	buf = append(buf, styles.Key.Render(key)...)
	buf = append(buf, '=')
	buf = append(buf, styles.Value.Render(formatValue(a.Value))...)
	return buf
}

func formatValue(v slog.Value) string {
	v = v.Resolve()
	if v.Kind() == slog.KindString {
		return v.String()
	}
	return fmt.Sprintf("%v", v.Any())
}

func cloneAttrs(a []slog.Attr) []slog.Attr {
	if len(a) == 0 {
		return nil
	}
	return append(make([]slog.Attr, 0, len(a)), a...)
}

// SetStyles updates the handler's styles.
// Safe to call concurrently with log writes.
func (h *Handler) SetStyles(s Styles) {
	h.mu.Lock()
	h.styles = s
	h.mu.Unlock()
}

// compile-time check
var _ slog.Handler = (*Handler)(nil)
