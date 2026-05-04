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

// Handler is a styled [slog.Handler] that writes human-readable log lines
// using lipgloss styles derived from a [Styles] value (typically produced
// by [FromTheme]).
//
// Output format: LEVL message and optional structured key=value pairs.
type Handler struct {
	w      io.Writer
	level  *slog.LevelVar
	styles Styles
	ts     bool

	mu    sync.Mutex
	group string
	attrs []slog.Attr
}

// HandlerOptions configures a [Handler].
type HandlerOptions struct {
	Level     *slog.LevelVar
	Styles    Styles
	Timestamp bool
}

// NewHandler returns a [Handler] that writes styled log output to w.
func NewHandler(w io.Writer, opts HandlerOptions) *Handler {
	lv := opts.Level
	if lv == nil {
		lv = new(slog.LevelVar)
	}
	return &Handler{
		w:      w,
		level:  lv,
		styles: opts.Styles,
		ts:     opts.Timestamp,
	}
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 128)

	buf = append(buf, h.levelBadge(r.Level)...)
	buf = append(buf, ' ')

	if h.ts {
		buf = append(buf, r.Time.Format(time.Kitchen)...)
		buf = append(buf, ' ')
	}

	buf = append(buf, r.Message...)

	if h.group != "" {
		appendAttr := func(a slog.Attr) {
			buf = append(buf, ' ')
			buf = h.appendKV(buf, h.group+"."+a.Key, a.Value)
		}
		for _, a := range h.attrs {
			appendAttr(a)
		}
		r.Attrs(func(a slog.Attr) bool {
			appendAttr(a)
			return true
		})
	} else {
		appendAttr := func(a slog.Attr) {
			buf = append(buf, ' ')
			buf = h.appendKV(buf, a.Key, a.Value)
		}
		for _, a := range h.attrs {
			appendAttr(a)
		}
		r.Attrs(func(a slog.Attr) bool {
			appendAttr(a)
			return true
		})
	}

	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf)
	return err
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{
		w:      h.w,
		level:  h.level,
		styles: h.styles,
		ts:     h.ts,
		group:  h.group,
		attrs:  append(cloneAttrs(h.attrs), attrs...),
	}
}

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
		group:  g,
		attrs:  cloneAttrs(h.attrs),
	}
}

func (h *Handler) levelBadge(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return h.styles.Error.String()
	case l >= slog.LevelWarn:
		return h.styles.Warn.String()
	case l >= slog.LevelInfo:
		return h.styles.Info.String()
	default:
		return h.styles.Debug.String()
	}
}

func (h *Handler) appendKV(buf []byte, key string, val slog.Value) []byte {
	buf = append(buf, h.styles.Key.Render(key)...)
	buf = append(buf, '=')
	buf = append(buf, h.styles.Value.Render(formatValue(val))...)
	return buf
}

func formatValue(v slog.Value) string {
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
