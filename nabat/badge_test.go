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
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
	"nabat.dev/theme"
)

func TestBadgeRendersIconAndLabel(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var got string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		got = c.Badge(nabat.IconSuccess, "running")
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	plain := ansi.Strip(got)
	assert.Contains(t, plain, "✓")
	assert.Contains(t, plain, "running")
}

func TestBadgeRespectsTheme(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	appNabat := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	appMinimal := nabat.MustNew("test2", nabat.WithIO(io), nabat.WithTheme(theme.Minimal))

	var a, b string
	appNabat.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		a = c.Badge(nabat.IconSuccess, "ok")
		return nil
	}))
	appMinimal.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		b = c.Badge(nabat.IconSuccess, "ok")
		return nil
	}))
	require.NoError(t, nabattest.Run(t, appNabat, []string{"run"}))
	require.NoError(t, nabattest.Run(t, appMinimal, []string{"run"}))
	assert.Equal(t, ansi.Strip(a), ansi.Strip(b), "plain text should match")
	// Styled output may differ by theme; at least both contain the label.
	assert.Contains(t, a, "ok")
	assert.Contains(t, b, "ok")
}

func TestFieldsAlignKeys(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var block string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		block = c.Fields([]nabat.Field{
			{Key: "A", Value: "one"},
			{Key: "Longer", Value: "two"},
		}).String()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))

	lines := strings.Split(strings.TrimRight(ansi.Strip(block), "\n"), "\n")
	require.Len(t, lines, 2)
	// Both keys padded to "Longer" width (6), then a space, then value.
	assert.True(t, strings.HasPrefix(lines[0], "A      "),
		"short key should be padded: %q", lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "Longer "),
		"long key should keep its width: %q", lines[1])
}

func TestFieldsWithKeyWidth(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var block string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		block = c.Fields([]nabat.Field{
			{Key: "Backend", Value: "kind"},
		}, nabat.WithFieldKeyWidth(14)).String()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))

	line := strings.TrimRight(ansi.Strip(block), "\n")
	// "Backend" is 7 runes; padded to 14 then space then value.
	assert.True(t, strings.HasPrefix(line, "Backend       "),
		"key should be padded to width 14: %q", line)
	assert.Contains(t, line, "kind")
}

func TestFieldBlockStringAndPrint(t *testing.T) {
	t.Parallel()

	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var asString string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		fb := c.Fields([]nabat.Field{{Key: "K", Value: "V"}})
		asString = fb.String()
		fb.Print()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	// String keeps theme ANSI; Print routes through Out, which may strip color
	// on non-TTY writers. Content must match after stripping.
	assert.Equal(t, ansi.Strip(asString), ansi.Strip(stdout.String()))
	assert.Contains(t, ansi.Strip(asString), "K V")
}

func TestFieldsKeyUsesTextMuted(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	var block, wantKey string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		wantKey = c.Render(theme.TextMuted, "Name")
		block = c.Fields([]nabat.Field{{Key: "Name", Value: "web"}}).String()
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	assert.Contains(t, block, wantKey)
}

func TestIconGlyphAndToken(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "✓", nabat.IconSuccess.Glyph())
	assert.Equal(t, theme.StatusSuccess, nabat.IconSuccess.Token())
	assert.Equal(t, "?", nabat.IconUnknown.Glyph())
	assert.Equal(t, theme.TextMuted, nabat.IconUnknown.Token())
}

func TestNewIconCustomBadge(t *testing.T) {
	t.Parallel()

	io, _, _, _ := nabattest.NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io), nabat.WithTheme(theme.Nabat))
	custom := nabat.NewIcon("★", theme.AccentPrimary)
	var got string
	app.MustCommand("run", nabat.WithRun(func(c *nabat.Context) error {
		got = c.Badge(custom, "featured")
		return nil
	}))
	require.NoError(t, nabattest.Run(t, app, []string{"run"}))
	plain := ansi.Strip(got)
	assert.Contains(t, plain, "★")
	assert.Contains(t, plain, "featured")
	assert.Equal(t, "★", custom.Glyph())
	assert.Equal(t, theme.AccentPrimary, custom.Token())
}
