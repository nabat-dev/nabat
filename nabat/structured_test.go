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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

// TestFormatString_returnsCanonicalLowercaseName locks the public wire-format
// names returned by [Format.String]. These strings appear in error messages
// (for example "nabat: unknown format %q") and are documented as stable; a
// drift here would change observable output.
func TestFormatString_returnsCanonicalLowercaseName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   Format
		want string
	}{
		{"FormatJSON renders as json", FormatJSON, "json"},
		{"FormatYAML renders as yaml", FormatYAML, "yaml"},
		{"FormatTOML renders as toml", FormatTOML, "toml"},
		{"zero value renders as unknown", Format(0), "unknown"},
		{"out-of-range value renders as unknown", Format(99), "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.in.String())
		})
	}
}

func TestStructuredEncodeOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		encode func(*Context) error
	}{
		{
			name: "YAML output contains map keys and values",
			encode: func(c *Context) error {
				return c.YAML(map[string]string{"name": "nabat"})
			},
		},
		{
			name: "TOML output contains map keys and values",
			encode: func(c *Context) error {
				return c.TOML(map[string]string{"name": "nabat"})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, stdout, _ := testIO()
			app := MustNew("test", WithIO(io))
			enc := tt.encode
			app.MustCommand("test", WithRun(func(c *Context) error {
				return enc(c)
			}))
			err := Run(t, app, []string{"test"})
			require.NoError(t, err)
			got := stdout.String()
			assert.Contains(t, got, "name")
			assert.Contains(t, got, "nabat")
		})
	}
}

func TestEncodeDispatch(t *testing.T) {
	tests := []struct {
		name   string
		format Format
	}{
		{name: "Encode JSON format produces output", format: FormatJSON},
		{name: "Encode YAML format produces output", format: FormatYAML},
		{name: "Encode TOML format produces output", format: FormatTOML},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, stdout, _ := testIO()
			app := MustNew("test", WithIO(io))
			f := tt.format
			app.MustCommand("test", WithRun(func(c *Context) error {
				return c.Encode(map[string]string{"key": "value"}, f)
			}))

			err := Run(t, app, []string{"test"})
			require.NoError(t, err)
			assert.NotEmpty(t, stdout.String())
		})
	}
}

func TestEncodeUnknownFormat(t *testing.T) {
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("test", WithRun(func(c *Context) error {
		return c.Encode(nil, Format(99))
	}))

	err := Run(t, app, []string{"test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")
}

func TestHighlight(t *testing.T) {
	io, _, stdout, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("test", WithRun(func(c *Context) error {
		return c.Highlight(`{"key": "value"}`, "json")
	}))

	err := Run(t, app, []string{"test"})
	require.NoError(t, err)
	got := stdout.String()
	assert.Contains(t, got, `"key"`)
	assert.Contains(t, got, `"value"`)
}

func TestHighlightUnknownLanguage(t *testing.T) {
	io, _, stdout, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("test", WithRun(func(c *Context) error {
		return c.Highlight("some text", "nonexistent-lang-xyz")
	}))

	err := Run(t, app, []string{"test"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "some text")
}

func TestMinimalThemeNoHighlighting(t *testing.T) {
	io, _, stdout, _ := testIO()
	app := MustNew("test",
		WithTheme(theme.Minimal),
		WithIO(io),
	)
	app.MustCommand("test", WithRun(func(c *Context) error {
		return c.JSON(map[string]string{"key": "value"})
	}))

	err := Run(t, app, []string{"test"})
	require.NoError(t, err)
	assert.NotContains(t, stdout.String(), "\033[")
}

func TestJSONMarshalError(t *testing.T) {
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("test", WithRun(func(c *Context) error {
		return c.JSON(make(chan int))
	}))

	err := Run(t, app, []string{"test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json encoding failed")
}
