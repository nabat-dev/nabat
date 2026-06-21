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
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"nabat.dev/theme"

	tea "charm.land/bubbletea/v2"
)

// spinnerDoneMsg carries the result of the worker function back to the
// Bubble Tea event loop.
type spinnerDoneMsg struct{ err error }

// spinnerModel is the Bubble Tea model for [Context.Spinner]. It renders a
// single animated header line that shows the spinner animation while running
// and a completion icon when done.
type spinnerModel struct {
	spin        spinner.Model
	handle      *Spinner
	titleStyle  lipgloss.Style
	activeStyle lipgloss.Style
	th          theme.ResolvedTheme
	action      func(*Spinner) error
	err         error
	done        bool
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
	var sb strings.Builder
	if m.done {
		sb.WriteString(m.completionIcon())
		sb.WriteString(" ")
	} else {
		sb.WriteString(m.spin.View())
	}
	if t := m.handle.title(); t != "" {
		sb.WriteString(m.titleStyle.Render(t))
	}
	sb.WriteString("\n")
	return tea.NewView(sb.String())
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
