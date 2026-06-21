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
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"nabat.dev/theme"

	tea "charm.land/bubbletea/v2"
)

// statusDoneMsg carries the result of the worker function back to the
// Bubble Tea event loop.
type statusDoneMsg struct{ err error }

// statusModel is the Bubble Tea model for [Context.Status]. It manages an
// optional animated header, column headers, live rows, and truncation when
// the number of rows exceeds the terminal height.
type statusModel struct {
	spin        spinner.Model
	handle      *Status
	cfg         *statusConfig
	titleStyle  lipgloss.Style
	activeStyle lipgloss.Style
	th          theme.ResolvedTheme
	action      func(*Status) error
	err         error
	done        bool
	height      int // terminal height from tea.WindowSizeMsg
	width       int // terminal width from tea.WindowSizeMsg
}

// newStatusModel constructs a statusModel. It is called from [Context.Status]
// after option validation.
func newStatusModel(
	spin spinner.Model,
	handle *Status,
	cfg *statusConfig,
	titleStyle lipgloss.Style,
	activeStyle lipgloss.Style,
	th theme.ResolvedTheme,
	action func(*Status) error,
) statusModel {
	return statusModel{
		spin:        spin,
		handle:      handle,
		cfg:         cfg,
		titleStyle:  titleStyle,
		activeStyle: activeStyle,
		th:          th,
		action:      action,
	}
}

func (m statusModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		func() tea.Msg { return statusDoneMsg{err: m.action(m.handle)} },
	)
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case statusDoneMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Interrupt
		}
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

func (m statusModel) View() tea.View {
	snaps := m.handle.rowSnapshots()
	var sb strings.Builder

	// Header line (only when WithTitle was used).
	hasTitle := m.cfg.title != ""
	if hasTitle {
		if m.done {
			icon := m.completionIcon()
			sb.WriteString(icon)
			sb.WriteString(" ")
		} else {
			sb.WriteString(m.spin.View())
		}
		if t := m.handle.getTitle(); t != "" {
			sb.WriteString(m.titleStyle.Render(t))
		}
		sb.WriteString("\n")
	}

	if len(snaps) == 0 {
		return tea.NewView(sb.String())
	}

	// Determine visible rows and column widths.
	visible, hidden := m.visibleRows(snaps)
	widths := computeColumnWidths(visible, m.cfg.noElapsed)

	// Expand widths to cover column header labels.
	headers := headerColumns(m.cfg.columns, m.cfg.noElapsed)
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

	// Column header row.
	if len(headers) > 0 {
		muted := m.th.Style(theme.TextMuted)
		// Indent to align with the data rows: " " + icon(1) + "  " = 4.
		sb.WriteString("    ")
		for i, h := range headers {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			sb.WriteString(muted.Render(padRight(h, w)))
			if i < len(headers)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	// Data rows.
	spinFrame := m.spin.View()
	for _, snap := range visible {
		rowIcon := m.rowIcon(snap, spinFrame)
		sb.WriteString(" ")
		sb.WriteString(rowIcon)
		sb.WriteString("  ")

		cols := rowColumns(snap, m.cfg.noElapsed)
		var rowContent strings.Builder
		for i, col := range cols {
			w := 0
			if i < len(widths) {
				w = widths[i]
			}
			rowContent.WriteString(padRight(col, w))
			if i < len(cols)-1 {
				rowContent.WriteString("  ")
			}
		}
		raw := rowContent.String()
		sb.WriteString(m.applyRowStyle(snap, raw))
		sb.WriteString("\n")
	}

	if hidden > 0 {
		muted := m.th.Style(theme.TextMuted)
		sb.WriteString(muted.Render(fmt.Sprintf("  ... and %d more rows", hidden)))
		sb.WriteString("\n")
	}

	return tea.NewView(sb.String())
}

// visibleRows selects which rows to display when the total count exceeds the
// available terminal lines. When priorities are set, higher-priority (lower
// number) rows are kept visible. Without priorities, the most recently added
// rows are kept. The hidden count is returned so the caller can show a summary.
func (m statusModel) visibleRows(snaps []rowSnapshot) (visible []rowSnapshot, hidden int) {
	if m.height == 0 {
		return snaps, 0
	}
	// Reserve lines for header (if shown), column headers (if shown), summary,
	// and a small margin.
	reserve := 3
	if m.cfg.title != "" {
		reserve++
	}
	if len(m.cfg.columns) > 0 {
		reserve++
	}
	maxRows := max(m.height-reserve, 1)
	if len(snaps) <= maxRows {
		return snaps, 0
	}

	// Keep the first maxRows by insertion order (which is already the display
	// order: prioritized first, then insertion order).
	return snaps[:maxRows], len(snaps) - maxRows
}

// rowIcon returns the icon string to display for snap. Active rows receive the
// animated spinner frame; completed rows receive a styled static icon based on
// the per-row icon override or the state default.
func (m statusModel) rowIcon(snap rowSnapshot, spinFrame string) string {
	if snap.state == RowActive {
		// Workaround for charmbracelet/bubbles#999: some spinner types embed a
		// trailing space in their frame strings (e.g. Dot uses "⣾ ") while
		// others do not. Strip ANSI codes and whitespace to get the bare glyph,
		// then re-style with StatusActive so the icon is exactly 1 visible
		// column regardless of spinner type.
		// TODO(mohammad): remove the strip once bubbles#999 is fixed upstream.
		plain := strings.TrimSpace(ansi.Strip(spinFrame))
		return m.activeStyle.Render(plain)
	}

	// Resolve icon: per-row override wins, then state default.
	var iconStr string
	if snap.icon != nil && *snap.icon != "" {
		iconStr = *snap.icon
	} else {
		switch snap.state {
		case RowSuccess:
			iconStr = m.handle.icons.successIcon()
		case RowError:
			iconStr = m.handle.icons.errorIcon()
		case RowWarning:
			iconStr = m.handle.icons.warningIcon()
		default:
			iconStr = m.handle.icons.doneIcon()
		}
	}

	switch snap.state {
	case RowSuccess:
		return m.th.Style(theme.StatusSuccess).Render(iconStr)
	case RowError:
		return m.th.Style(theme.StatusError).Render(iconStr)
	case RowWarning:
		return m.th.Style(theme.StatusWarning).Render(iconStr)
	default:
		return m.th.Style(theme.TextMuted).Render(iconStr)
	}
}

// completionIcon returns the styled header icon for the completed state.
func (m statusModel) completionIcon() string {
	switch m.handle.completionState(m.err) {
	case RowSuccess:
		return m.th.Style(theme.StatusSuccess).Render(m.handle.icons.successIcon())
	case RowError:
		return m.th.Style(theme.StatusError).Render(m.handle.icons.errorIcon())
	case RowWarning:
		return m.th.Style(theme.StatusWarning).Render(m.handle.icons.warningIcon())
	default:
		return m.th.Style(theme.TextMuted).Render(m.handle.icons.doneIcon())
	}
}

// applyRowStyle wraps text in the color appropriate for snap's state. Active
// rows receive no color (terminal default). Completed rows use their state
// color or the custom [theme.Token] set by [StatusRow.Style].
func (m statusModel) applyRowStyle(snap rowSnapshot, text string) string {
	if snap.style != nil {
		return m.th.Style(*snap.style).Render(text)
	}
	switch snap.state {
	case RowSuccess:
		return m.th.Style(theme.StatusSuccess).Render(text)
	case RowError:
		return m.th.Style(theme.StatusError).Render(text)
	case RowWarning:
		return m.th.Style(theme.StatusWarning).Render(text)
	default:
		return text
	}
}
