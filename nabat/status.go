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

// WithTitle sets the header title for [Context.Spinner] or [Context.Status].
// For Status it appears above the rows with an animated spinner while the work
// function runs and a completion icon when it returns. For Spinner it is the
// initial animated line (and the plain line printed in the non-TTY path).
//
// Example:
//
//	c.Spinner(fn, nabat.WithTitle("Deploying"))
//	c.Status(fn, nabat.WithTitle("Deploying"))
func WithTitle(title string) spinnerStatusOption {
	return sharedOption{
		spinnerFn: func(c *spinnerConfig) error { c.title = title; return nil },
		statusFn:  func(c *statusConfig) error { c.title = title; return nil },
	}
}

// WithColumns sets status column headers for the label and Set() cells.
// An "AGE" header is appended unless [WithoutElapsed] is set.
//
// Example:
//
//	c.Status(fn, WithTitle("Events"), WithColumns("OBJECT", "REASON", "MESSAGE"))
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

// Status is the live handle passed to [Context.Status]. Use [Status.Row],
// [Status.SetTitle], and [Status.SetCompletion]. Row is safe for concurrent
// use; a snapshot may miss rows added mid-collection until the next tick.
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

// Row returns the keyed row, creating it on first use. The key is for
// dedup only; set the visible label with [StatusRow.Label]. Safe for
// concurrent use.
//
// Example:
//
//	st.Row(id).Label(obj).Set(reason, msg).Warn()
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
	// priority is copied under the row lock so sort does not race with
	// concurrent [StatusRow.Priority] writers.
	type indexed struct {
		row      *StatusRow
		insertID int
		priority int
	}
	var prioritized, unprioritized []indexed
	for i, r := range rows {
		r.mu.Lock()
		hidden := r.hidden
		var (
			hasPrio bool
			prioVal int
		)
		if r.priority != nil {
			hasPrio = true
			prioVal = *r.priority
		}
		r.mu.Unlock()
		if hidden {
			continue
		}
		if hasPrio {
			prioritized = append(prioritized, indexed{r, i, prioVal})
		} else {
			unprioritized = append(unprioritized, indexed{r, i, 0})
		}
	}

	// Sort prioritized rows by priority value, preserving insertion order on ties.
	sort.SliceStable(prioritized, func(a, b int) bool {
		if prioritized[a].priority != prioritized[b].priority {
			return prioritized[a].priority < prioritized[b].priority
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

// Status runs fn with a live multi-row status display on stderr.
// On a TTY, fn runs while the UI animates; Status returns only after fn
// returns. On cancel or ctrl+c it stops the UI, waits for fn, then returns
// [context.Canceled] unless fn returned a non-nil error. Bad options yield
// [*ConfigErrors].
//
// Use [WithTitle], [WithColumns], and [Status.Row] to shape the display.
// Non-TTY output is a plain title plus a final table.
//
// Example:
//
//	return c.Status(func(st *Status) error {
//	    st.Row("build").Set("compiling").Success()
//	    return nil
//	}, WithTitle("Deploy"))
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

	var (
		fnErr  error
		fnDone sync.WaitGroup
	)
	fnDone.Go(func() {
		fnErr = fn(handle)
	})

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
		&fnDone,
		&fnErr,
	)

	final, runErr := tea.NewProgram(
		model,
		tea.WithContext(c),
		tea.WithOutput(c.io.RawErrOut()),
		tea.WithInput(c.io.RawIn()),
	).Run()

	// Always wait for fn, including on interrupt / program kill, so Status
	// never returns while the worker is still running.
	fnDone.Wait()

	if fnErr != nil {
		return fnErr
	}
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
