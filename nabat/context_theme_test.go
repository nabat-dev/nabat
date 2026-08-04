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

package nabat_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
	"nabat.dev/theme"
)

func TestContextThemeEqualsAppTheme(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var ctxThemeName string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		ctxThemeName = c.Theme().Name()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Equal(t, app.Theme().Name(), ctxThemeName)
}

func TestContextStyleReturnsLipglossStyle(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var style lipgloss.Style
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		style = c.Style(theme.TextMuted)
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Equal(t, app.Theme().Style(theme.TextMuted), style)
}

func TestContextRenderReturnsStyledString(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewTTYIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var rendered string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		rendered = c.Render(theme.StatusSuccess, "ok")
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, rendered, "ok")
	assert.Equal(t, app.Theme().Style(theme.StatusSuccess).Render("ok"), rendered)
}
