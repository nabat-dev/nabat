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
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTableWithASCIIBorder(t *testing.T) {
	t.Parallel()

	io, _, out, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("table", WithRun(func(c *Context) error {
		c.Table(
			[]string{"name", "status"},
			[][]string{{"api", "running"}, {"web", "stopped"}},
			WithTableBorder(BorderASCII()),
			WithTableBorderRow(true),
		)
		return nil
	}))

	err := Run(t, app, []string{"table"})
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "+")
	assert.Contains(t, s, "|")
	assert.Contains(t, s, "name")
	assert.Contains(t, s, "running")
}

func TestTableStyleFuncReceivesHeaderRow(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	calledHeader := false
	app.MustCommand("table", WithRun(func(c *Context) error {
		c.Table(
			[]string{"name"},
			[][]string{{"api"}},
			WithTableStyleFunc(func(row, col int) lipgloss.Style {
				if row == HeaderRow && col == 0 {
					calledHeader = true
				}
				return lipgloss.NewStyle()
			}),
		)
		return nil
	}))

	err := Run(t, app, []string{"table"})
	require.NoError(t, err)
	assert.True(t, calledHeader)
}
