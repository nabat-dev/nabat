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
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"

	"nabat.dev/theme"

	tea "charm.land/bubbletea/v2"
)

// statusConfig holds the resolved configuration for a [Context.Status] call.
type statusConfig struct {
	title       string
	columns     []string
	spinnerType SpinnerType
	icons       Icons
	noElapsed   bool
}

// WithTitle sets an optional header title shown above the status rows with an
// animated spinner while the work function runs and a completion icon when it
// returns.
//
// Example:
//
//	c.Status(fn, nabat.WithTitle("Deploying"))
func WithTitle(title string) StatusOption {
	return statusOnlyOption(func(c *statusConfig) error {
		c.title = title
		return nil
	})
}

// WithColumns sets the column header labels shown above the status rows. The
// headers cover the label column (first) and all Set() cell columns in order.
// An "AGE" header is auto-appended for the elapsed column unless
// [WithoutElapsed] is also set.
//
// When a row has fewer cells than there are headers, the missing columns
// render as blank. When a row has more cells than there are headers, the extra
// cells render without a header above them.
//
// Example:
//
//	c.Status(fn,
//	    nabat.WithTitle("Events"),
//	    nabat.WithColumns("OBJECT", "REASON", "MESSAGE"),
//	)
func WithColumns(names ...string) StatusOption {
	return statusOnlyOption(func(c *statusConfig) error {
		c.columns = append([]string(nil), names...)
		return nil
	})
}

// WithoutElapsed suppresses the auto-generated elapsed time column from the
// status display. When set, the "AGE" header is also omitted when
// [WithColumns] is in use.
func WithoutElapsed() StatusOption {
	return statusOnlyOption(func(c *statusConfig) error {
		c.noElapsed = true
		return nil
	})
}

// WithStatusIcons overrides the default row state icons for a [Context.Status]
// call. Only the non-empty fields in icons are used; empty fields keep their
// built-in defaults ("✓", "✗", "!", "•").
//
// Example:
//
//	c.Status(fn, nabat.WithStatusIcons(nabat.Icons{
//	    Success: "+",
//	    Error:   "x",
//	}))
func WithStatusIcons(icons Icons) StatusOption {
	return statusOnlyOption(func(c *statusConfig) error {
		c.icons = icons
		return nil
	})
}

// Status is the live handle passed to the [Context.Status] callback. Call
// [Status.Row] to add or update keyed status rows. Call [Status.SetTitle] to
// update the header title (only meaningful when [WithTitle] was used). Call
// [Status.SetCompletion] to control the header's completion icon.
//
// Row is safe to call from any goroutine. A snapshot is a consistent read of
// all rows that existed at the time of the call; rows added after a snapshot
// collection begins may not appear until the next render tick.
type Status struct {
	mu          sync.Mutex
	headerTitle string
	rows        []*StatusRow          // insertion-ordered
	rowIdx      map[string]*StatusRow // key -> row
	icons       Icons
	start       time.Time
	completion  *RowState // nil = derive from fn error
}

// SetTitle replaces the header title. It is safe to call from any goroutine;
// last writer wins. Only meaningful when [WithTitle] was used. A call after
// the work function returns is a harmless no-op.
func (s *Status) SetTitle(title string) {
	s.mu.Lock()
	s.headerTitle = title
	s.mu.Unlock()
}

// SetCompletion sets the header's final icon state explicitly. The value is
// used instead of deriving the state from the fn error when the work function
// returns. Call this inside fn before returning to control the result icon.
// It is safe to call from any goroutine; last writer wins.
func (s *Status) SetCompletion(state RowState) {
	s.mu.Lock()
	s.completion = &state
	s.mu.Unlock()
}

func (s *Status) getTitle() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.headerTitle
}

// Row returns the keyed row for key, creating it on first call and returning
// the same row on every subsequent call. Rows are displayed beneath the header
// in creation order unless [StatusRow.Priority] is used to control sort order.
// Row is safe to call from any goroutine.
//
// The key is used only for dedup; it is not shown in the display. Use
// [StatusRow.Label] to set the visible first column.
//
// Example:
//
//	row := st.Row(string(ev.UID))
//	row.Label(ev.Object).Set(ev.Reason, ev.Message).Warn()
//
// Later calls with the same key return the same row:
//
//	st.Row(string(ev.UID)).Set(ev.Reason, "updated").Success()
func (s *Status) Row(key string) *StatusRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.rowIdx[key]; ok {
		return r
	}
	if s.rowIdx == nil {
		s.rowIdx = make(map[string]*StatusRow)
	}
	r := &StatusRow{
		key:     key,
		created: time.Now(),
	}
	s.rows = append(s.rows, r)
	s.rowIdx[key] = r
	return r
}

// rowSnapshots returns point-in-time copies of visible rows in display order.
// Hidden rows are excluded. Rows with a priority are sorted first (ascending),
// then unprioritized rows follow in insertion order.
func (s *Status) rowSnapshots() []rowSnapshot {
	s.mu.Lock()
	rows := append([]*StatusRow(nil), s.rows...)
	s.mu.Unlock()

	// Separate prioritized and unprioritized visible rows.
	type indexed struct {
		row      *StatusRow
		insertID int
	}
	var prioritized, unprioritized []indexed
	for i, r := range rows {
		r.mu.Lock()
		hidden := r.hidden
		prio := r.priority
		r.mu.Unlock()
		if hidden {
			continue
		}
		if prio != nil {
			prioritized = append(prioritized, indexed{r, i})
		} else {
			unprioritized = append(unprioritized, indexed{r, i})
		}
	}

	// Sort prioritized rows by priority value, preserving insertion order on ties.
	sort.SliceStable(prioritized, func(a, b int) bool {
		pa := *prioritized[a].row.priority
		pb := *prioritized[b].row.priority
		if pa != pb {
			return pa < pb
		}
		return prioritized[a].insertID < prioritized[b].insertID
	})

	ordered := make([]*StatusRow, 0, len(prioritized)+len(unprioritized))
	for _, x := range prioritized {
		ordered = append(ordered, x.row)
	}
	for _, x := range unprioritized {
		ordered = append(ordered, x.row)
	}

	snaps := make([]rowSnapshot, 0, len(ordered))
	for _, r := range ordered {
		snaps = append(snaps, r.snapshot())
	}
	return snaps
}

// allRowSnapshots returns snapshots of all rows including hidden ones.
// Used for completion state so hidden error rows still count.
func (s *Status) allRowSnapshots() []rowSnapshot {
	s.mu.Lock()
	rows := append([]*StatusRow(nil), s.rows...)
	s.mu.Unlock()
	snaps := make([]rowSnapshot, 0, len(rows))
	for _, r := range rows {
		snaps = append(snaps, r.snapshot())
	}
	return snaps
}

// completionState determines the header icon. If SetCompletion was called,
// that value is returned. Otherwise: fn error != nil returns RowError, nil
// returns RowSuccess. Row states do not bubble up to the header.
func (s *Status) completionState(fnErr error) RowState {
	s.mu.Lock()
	comp := s.completion
	s.mu.Unlock()
	if comp != nil {
		return *comp
	}
	if fnErr != nil {
		return RowError
	}
	return RowSuccess
}

// headerColumns builds the column header row for rendering.
// Returns nil when no columns were configured.
func headerColumns(columns []string, noElapsed bool) []string {
	if len(columns) == 0 {
		return nil
	}
	out := append([]string(nil), columns...)
	if !noElapsed {
		out = append(out, "AGE")
	}
	return out
}

// renderPlainTable returns an aligned plain-text table of all visible rows
// with no ANSI styling. Used by the non-TTY fallback path.
func (s *Status) renderPlainTable(cfg *statusConfig) string {
	snaps := s.rowSnapshots()
	if len(snaps) == 0 {
		return ""
	}

	headers := headerColumns(cfg.columns, cfg.noElapsed)
	widths := computeColumnWidths(snaps, cfg.noElapsed)

	// Expand widths to cover header labels too.
	if len(headers) > 0 {
		for i, h := range headers {
			w := visibleWidth(h)
			if i >= len(widths) {
				widths = append(widths, w)
			} else if w > widths[i] {
				widths[i] = w
			}
		}
	}

	var sb strings.Builder

	// Header row.
	if len(headers) > 0 {
		// Indent to match the icon+space prefix of data rows: " " + icon(1) + "  " = 4.
		sb.WriteString("    ")
		for i, h := range headers {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			sb.WriteString(padRight(h, w))
			if i < len(headers)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	// Data rows.
	for _, snap := range snaps {
		icon := resolvedIcon(snap, cfg.icons)
		sb.WriteString(" ")
		sb.WriteString(icon)
		sb.WriteString("  ")
		cols := rowColumns(snap, cfg.noElapsed)
		for i, col := range cols {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			sb.WriteString(padRight(col, w))
			if i < len(cols)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// Status runs fn while showing a live status display on stderr in interactive
// terminals. The callback receives a [*Status] handle; call [Status.Row] to
// add or update keyed status rows.
//
// Each row shows its own animation and an auto-incrementing elapsed timer.
// Calling [StatusRow.Success], [StatusRow.Error], [StatusRow.Warn], or
// [StatusRow.Done] freezes the timer and replaces the animation with a static
// icon.
//
// Use [WithTitle] to show an animated header line above the rows. The header
// icon on completion is derived from the fn return value unless overridden by
// [Status.SetCompletion].
//
// Use [WithColumns] to show column header labels above the rows. An "AGE"
// header is auto-appended for the elapsed column unless [WithoutElapsed] is
// also set.
//
// In non-TTY environments (CI, piped output), the title (if set) is printed
// once as a plain line and the final row state prints as a plain-text aligned
// table after fn returns.
//
// Errors:
//   - any error returned by fn
//   - [*ConfigErrors] from option validation
//   - [context.Canceled] when the user interrupts with ctrl+c or the context
//     is canceled
func (c *Context) Status(fn func(*Status) error, opts ...StatusOption) error {
	cfg := &statusConfig{spinnerType: SpinnerDots()}
	var errs ConfigErrors
	for i, opt := range opts {
		if opt == nil {
			errs.AddErr(fmtErrInvalidOption("status option", i))
			continue
		}
		if err := opt.applyStatus(cfg); err != nil {
			errs.AddErr(err)
		}
	}
	if errs.HasIssues() {
		return &errs
	}

	handle := &Status{
		headerTitle: cfg.title,
		icons:       cfg.icons,
		start:       time.Now(),
	}

	// Non-TTY path: print title once then run fn; append final row table.
	if !c.io.IsStderrTTY() {
		if cfg.title != "" {
			if _, err := fmt.Fprintln(c.io.ErrOut, cfg.title); err != nil {
				return err
			}
		}
		fnErr := fn(handle)
		if table := handle.renderPlainTable(cfg); table != "" {
			if _, wErr := fmt.Fprint(c.io.ErrOut, table); wErr != nil && fnErr == nil {
				return wErr
			}
		}
		return fnErr
	}

	rt := c.app.Theme()
	info := rt.Style(theme.StatusInfo)
	activeStyle := rt.Style(theme.StatusActive)
	model := newStatusModel(
		spinner.New(
			spinner.WithSpinner(spinner.Spinner(cfg.spinnerType)),
			spinner.WithStyle(info),
		),
		handle,
		cfg,
		info,
		activeStyle,
		rt,
		fn,
	)

	final, runErr := tea.NewProgram(
		model,
		tea.WithContext(c),
		tea.WithOutput(c.io.RawErrOut()),
		tea.WithInput(c.io.RawIn()),
	).Run()

	if m, ok := final.(statusModel); ok && m.err != nil {
		return m.err
	}
	switch {
	case errors.Is(runErr, tea.ErrInterrupted):
		return context.Canceled
	case errors.Is(runErr, tea.ErrProgramKilled):
		if cerr := c.Err(); cerr != nil {
			return cerr
		}
		return context.Canceled
	}
	return runErr
}
