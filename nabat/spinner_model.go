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

	"nabat.dev/theme"

	tea "charm.land/bubbletea/v2"
)

// spinnerDoneMsg carries the result of the worker function back to the
// Bubble Tea event loop.
type spinnerDoneMsg struct{ err error }

// spinnerModel is the Bubble Tea model for [Context.Spinner]. It manages an
// animated header, optional live rows, and smart truncation when the number
// of rows exceeds the terminal height.
type spinnerModel struct {
	spin        spinner.Model
	handle      *Spinner
	titleStyle  lipgloss.Style
	activeStyle lipgloss.Style
	th          theme.ResolvedTheme
	action      func(*Spinner) error
	err         error
	done        bool
	height      int // terminal height from tea.WindowSizeMsg
	width       int // terminal width from tea.WindowSizeMsg
}

// newSpinnerModel constructs a spinnerModel. It is called from
// [Context.Spinner] after option validation.
func newSpinnerModel(
	spin spinner.Model,
	handle *Spinner,
	titleStyle lipgloss.Style,
	activeStyle lipgloss.Style,
	th theme.ResolvedTheme,
	action func(*Spinner) error,
) spinnerModel {
	return spinnerModel{
		spin:        spin,
		handle:      handle,
		titleStyle:  titleStyle,
		activeStyle: activeStyle,
		th:          th,
		action:      action,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spin.Tick,
		func() tea.Msg { return spinnerDoneMsg{err: m.action(m.handle)} },
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerDoneMsg:
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

func (m spinnerModel) View() tea.View {
	snaps := m.handle.rowSnapshots()
	var sb strings.Builder

	// Header line.
	if m.done {
		icon := m.completionIcon()
		sb.WriteString(icon)
		sb.WriteString(" ")
	} else {
		sb.WriteString(m.spin.View())
	}
	if t := m.handle.title(); t != "" {
		sb.WriteString(m.titleStyle.Render(t))
	}
	sb.WriteString("\n")

	if len(snaps) == 0 {
		return tea.NewView(sb.String())
	}

	// Row display.
	spinFrame := m.spin.View()
	visible, hidden := m.visibleRows(snaps)
	widths := computeColumnWidths(visible)

	for _, snap := range visible {
		rowIcon := m.rowIcon(snap, spinFrame)
		sb.WriteString(" ")
		sb.WriteString(rowIcon)
		sb.WriteString("  ")

		cols := rowColumns(snap)
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
		sb.WriteString(muted.Render(fmt.Sprintf("  ... and %d more completed", hidden)))
		sb.WriteString("\n")
	}

	return tea.NewView(sb.String())
}

// visibleRows selects which rows to display when the total count exceeds the
// available terminal lines. Priority order is: errors and warnings, then
// active rows, then recent completions. The remaining count is returned so the
// caller can show a summary line.
func (m spinnerModel) visibleRows(snaps []rowSnapshot) (visible []rowSnapshot, hidden int) {
	if m.height == 0 {
		return snaps, 0
	}
	// Reserve lines for header, summary, and a small margin.
	maxRows := max(m.height-3, 1)
	if len(snaps) <= maxRows {
		return snaps, 0
	}

	var errWarn, active, done []rowSnapshot
	for _, s := range snaps {
		switch s.state {
		case RowError, RowWarning:
			errWarn = append(errWarn, s)
		case RowActive:
			active = append(active, s)
		default:
			done = append(done, s)
		}
	}

	result := make([]rowSnapshot, 0, maxRows)
	result = append(result, errWarn...)
	if len(result) < maxRows {
		take := min(len(active), maxRows-len(result))
		result = append(result, active[:take]...)
	}
	if len(result) < maxRows {
		remain := min(maxRows-len(result), len(done))
		// Most recently added completions (end of slice).
		result = append(result, done[len(done)-remain:]...)
	}

	return result, len(snaps) - len(result)
}

// rowIcon returns the icon string to display for snap. Active rows receive the
// animated spinner frame; completed rows receive a styled static icon.
func (m spinnerModel) rowIcon(snap rowSnapshot, spinFrame string) string {
	switch snap.state {
	case RowActive:
		// Re-render the spinner frame with the active style so per-row icons
		// use [theme.SpinnerActive] rather than the header's [theme.StatusInfo].
		// If both tokens resolve to the same style the output is identical.
		frame := m.activeStyle.Render(strings.TrimSpace(m.spin.View()))
		_ = spinFrame // spinFrame is passed for caller clarity; we re-derive above.
		return frame
	case RowSuccess:
		return m.th.Style(theme.StatusSuccess).Render(m.handle.icons.successIcon())
	case RowError:
		return m.th.Style(theme.StatusError).Render(m.handle.icons.errorIcon())
	case RowWarning:
		return m.th.Style(theme.StatusWarning).Render(m.handle.icons.warningIcon())
	default: // RowDone and unknown states
		return m.th.Style(theme.TextMuted).Render(m.handle.icons.doneIcon())
	}
}

// completionIcon returns the styled header icon for the completed state.
func (m spinnerModel) completionIcon() string {
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
// color or the custom [theme.Token] set by [SpinnerRow.Style].
func (m spinnerModel) applyRowStyle(snap rowSnapshot, text string) string {
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
