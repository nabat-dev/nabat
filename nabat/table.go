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
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"

	"nabat.dev/theme"
)

// HeaderRow denotes the header row index passed to [WithTableStyleFunc].
// Use this value to identify header cells inside a style function callback.
//
// Example:
//
//	WithTableStyleFunc(func(row, col int) lipgloss.Style {
//		if row == HeaderRow {
//			return lipgloss.NewStyle().Bold(true)
//		}
//		return lipgloss.NewStyle()
//	})
const HeaderRow = table.HeaderRow

// Border presets for use with [WithTableBorder].
// These are aliases for lipgloss border types, so callers do not need to import
// lipgloss directly.
//
// Available presets:
//
//   - [BorderNormal] — single-line Unicode box drawing (default)
//   - [BorderRounded] — rounded corners
//   - [BorderBlock] — full-block characters
//   - [BorderMarkdown] — pipe-and-dash Markdown table style
//   - [BorderThick] — heavy Unicode box drawing
//   - [BorderDouble] — double-line Unicode box drawing
//   - [BorderHidden] — blank-character border that occupies the same width as a real border
//   - [BorderASCII] — plain ASCII characters (+, -, |)

// BorderNormal returns a single-line Unicode box-drawing border (default for
// [Context.Table]).
func BorderNormal() lipgloss.Border { return lipgloss.NormalBorder() }

// BorderRounded returns a single-line Unicode border with rounded corners.
func BorderRounded() lipgloss.Border { return lipgloss.RoundedBorder() }

// BorderBlock returns a border drawn with full-block characters.
func BorderBlock() lipgloss.Border { return lipgloss.BlockBorder() }

// BorderMarkdown returns a pipe-and-dash border that mimics Markdown table style.
func BorderMarkdown() lipgloss.Border { return lipgloss.MarkdownBorder() }

// BorderThick returns a heavy Unicode box-drawing border.
func BorderThick() lipgloss.Border { return lipgloss.ThickBorder() }

// BorderDouble returns a double-line Unicode box-drawing border.
func BorderDouble() lipgloss.Border { return lipgloss.DoubleBorder() }

// BorderHidden returns a border made of blank characters; it occupies the same
// width as a real border but renders nothing visible.
func BorderHidden() lipgloss.Border { return lipgloss.HiddenBorder() }

// BorderASCII returns a plain ASCII border drawn with +, -, and | characters.
func BorderASCII() lipgloss.Border { return lipgloss.ASCIIBorder() }

type tableConfig struct {
	border      lipgloss.Border
	borderStyle lipgloss.Style
	headerStyle lipgloss.Style
	cellStyle   lipgloss.Style
	styleFunc   func(row, col int) lipgloss.Style
	width       int
	wrap        bool

	borderTop    bool
	borderBottom bool
	borderLeft   bool
	borderRight  bool
	borderHeader bool
	borderColumn bool
	borderRow    bool
}

// TableOption configures the [Context.Table] output method.
// Pass one or more options to customize borders, styling, width, and wrapping.
//
// Example:
//
//	c.Table(headers, rows,
//		WithTableBorder(BorderRounded()),
//		WithTableWidth(80),
//		WithTableBorderRow(true),
//	)
type TableOption func(*tableConfig)

// WithTableBorder sets the border shape drawn around and inside the table.
// Pass one of the Border* presets or a custom [lipgloss.Border] value.
// The default is [BorderNormal].
//
// Example:
//
//	WithTableBorder(BorderRounded())
func WithTableBorder(b lipgloss.Border) TableOption {
	return func(c *tableConfig) { c.border = b }
}

// WithTableBorderStyle sets the [lipgloss] style applied to border characters.
// Use this to change border color or decoration independently of the border shape.
// The default comes from [Theme.TableBorderStyle].
//
// Example:
//
//	WithTableBorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6")))
func WithTableBorderStyle(s lipgloss.Style) TableOption {
	return func(c *tableConfig) { c.borderStyle = s }
}

// WithTableHeaderStyle sets the [lipgloss] style for header row cells.
// The default comes from [Theme.TableHeaderStyle].
//
// Example:
//
//	WithTableHeaderStyle(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("111")))
func WithTableHeaderStyle(s lipgloss.Style) TableOption {
	return func(c *tableConfig) { c.headerStyle = s }
}

// WithTableCellStyle sets the base [lipgloss] style for data cells.
// It applies to all non-header rows. The default comes from
// [Theme.TableCellStyle].
// Use [WithTableStyleFunc] for per-cell control.
//
// Example:
//
//	WithTableCellStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("252")))
func WithTableCellStyle(s lipgloss.Style) TableOption {
	return func(c *tableConfig) { c.cellStyle = s }
}

// WithTableStyleFunc sets a per-cell style function that receives the row and
// column indices and returns the [lipgloss] style for that cell. When set, it
// overrides
// [WithTableHeaderStyle] and [WithTableCellStyle].
// The row parameter is [HeaderRow] (-1) for header cells.
//
// Example:
//
//	WithTableStyleFunc(func(row, col int) lipgloss.Style {
//		if row == HeaderRow {
//			return lipgloss.NewStyle().Bold(true)
//		}
//		if row%2 == 0 {
//			return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
//		}
//		return lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
//	})
func WithTableStyleFunc(fn func(row, col int) lipgloss.Style) TableOption {
	return func(c *tableConfig) { c.styleFunc = fn }
}

// WithTableWidth sets a fixed total width for the table in columns.
// Column widths are auto-sized to fit within this constraint.
// A value of 0 or less is ignored, and the table sizes to its content.
//
// Example:
//
//	WithTableWidth(80)
func WithTableWidth(w int) TableOption {
	return func(c *tableConfig) { c.width = w }
}

// WithTableWrap enables or disables text wrapping inside data cells.
// When disabled, long cell content is truncated with an ellipsis.
// Headers are never wrapped. Defaults to enabled.
//
// Example:
//
//	WithTableWrap(false)
func WithTableWrap(enabled bool) TableOption {
	return func(c *tableConfig) { c.wrap = enabled }
}

// WithTableBorderTop toggles the top border of the table.
// Defaults to enabled.
func WithTableBorderTop(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderTop = enabled }
}

// WithTableBorderBottom toggles the bottom border of the table.
// Defaults to enabled.
func WithTableBorderBottom(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderBottom = enabled }
}

// WithTableBorderLeft toggles the left border of the table.
// Defaults to enabled.
func WithTableBorderLeft(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderLeft = enabled }
}

// WithTableBorderRight toggles the right border of the table.
// Defaults to enabled.
func WithTableBorderRight(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderRight = enabled }
}

// WithTableBorderHeader toggles the horizontal separator between the header row
// and the first data row. Defaults to enabled.
func WithTableBorderHeader(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderHeader = enabled }
}

// WithTableBorderColumn toggles the vertical separators between columns.
// Defaults to enabled.
func WithTableBorderColumn(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderColumn = enabled }
}

// WithTableBorderRow toggles the horizontal separators between data rows.
// Defaults to disabled.
func WithTableBorderRow(enabled bool) TableOption {
	return func(c *tableConfig) { c.borderRow = enabled }
}

// Table prints a styled table to the command's output writer.
// It applies the current [Theme] table styles by default. Pass [TableOption]
// values to override borders, styling, width, or wrapping.
//
// Example:
//
//	c.Table([]string{"Name", "Status"}, [][]string{
//		{"api", "running"},
//		{"web", "stopped"},
//	})
//
//	c.Table(headers, rows,
//		WithTableBorder(BorderRounded()),
//		WithTableWidth(80),
//	)
func (c *Context) Table(headers []string, rows [][]string, opts ...TableOption) {
	rt := c.app.Theme()
	cfg := &tableConfig{
		border:       rt.TableBorder(),
		borderStyle:  rt.Style(theme.TableBorder),
		headerStyle:  rt.Style(theme.TableHeader),
		cellStyle:    rt.Style(theme.TableCell),
		wrap:         true,
		borderTop:    true,
		borderBottom: true,
		borderLeft:   true,
		borderRight:  true,
		borderHeader: true,
		borderColumn: true,
		borderRow:    false,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	t := table.New().Headers(headers...).Rows(rows...)
	t.Border(cfg.border).BorderStyle(cfg.borderStyle)
	t.BorderTop(cfg.borderTop)
	t.BorderBottom(cfg.borderBottom)
	t.BorderLeft(cfg.borderLeft)
	t.BorderRight(cfg.borderRight)
	t.BorderHeader(cfg.borderHeader)
	t.BorderColumn(cfg.borderColumn)
	t.BorderRow(cfg.borderRow)
	t.Wrap(cfg.wrap)
	if cfg.width > 0 {
		t.Width(cfg.width)
	}

	if cfg.styleFunc != nil {
		t.StyleFunc(cfg.styleFunc)
	} else {
		t.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return cfg.headerStyle
			}
			return cfg.cellStyle
		})
	}

	out := writer{w: c.io.Out}
	out.println(t)
}
