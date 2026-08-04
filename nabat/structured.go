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
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
)

// Format names a structured encoding used by [Context.Encode] and [Marshal].
// It is a type-safe enum; the zero value is invalid so accidentally defaulting
// to JSON is impossible.
type Format uint8

const (
	// FormatJSON selects indented JSON output.
	FormatJSON Format = iota + 1
	// FormatYAML selects YAML output.
	FormatYAML
	// FormatTOML selects TOML output.
	FormatTOML
)

// String implements [fmt.Stringer] and returns the canonical lowercase name of
// the format ("json", "yaml", "toml"). Unknown values return "unknown".
func (f Format) String() string {
	switch f {
	case FormatJSON:
		return "json"
	case FormatYAML:
		return "yaml"
	case FormatTOML:
		return "toml"
	default:
		return "unknown"
	}
}

// JSON writes v as indented JSON to [Context.IO.Out], using the active [Theme]
// chroma style when set, or plain text when highlighting is disabled.
//
// Errors:
//   - "nabat: json encoding failed: ..." when marshaling fails
//   - errors from writing to [Context.IO.Out]
func (c *Context) JSON(v any) error {
	b, err := Marshal(v, FormatJSON)
	if err != nil {
		return err
	}
	return c.PrintHighlight(string(b), "json")
}

// YAML writes v as YAML to [Context.IO.Out], with highlighting behavior like
// [Context.JSON].
//
// Errors:
//   - "nabat: yaml encoding failed: ..." when marshaling fails
//   - errors from writing to [Context.IO.Out]
func (c *Context) YAML(v any) error {
	b, err := Marshal(v, FormatYAML)
	if err != nil {
		return err
	}
	return c.PrintHighlight(strings.TrimRight(string(b), "\n"), "yaml")
}

// TOML writes v as TOML to [Context.IO.Out], with highlighting behavior like
// [Context.JSON].
//
// Errors:
//   - "nabat: toml encoding failed: ..." when encoding fails
//   - errors from writing to [Context.IO.Out]
func (c *Context) TOML(v any) error {
	b, err := Marshal(v, FormatTOML)
	if err != nil {
		return err
	}
	return c.PrintHighlight(string(b), "toml")
}

// Encode writes v using [FormatJSON], [FormatYAML], or [FormatTOML], delegating to
// [Context.JSON], [Context.YAML], or [Context.TOML].
//
// Example:
//
//	return c.Encode(payload, FormatJSON)
//
// Errors:
//   - "nabat: unknown format %d" when f is not one of the [Format] constants
//   - errors from the selected encoder
func (c *Context) Encode(v any, f Format) error {
	switch f {
	case FormatJSON:
		return c.JSON(v)
	case FormatYAML:
		return c.YAML(v)
	case FormatTOML:
		return c.TOML(v)
	default:
		return fmt.Errorf("nabat: unknown format %d", int(f))
	}
}

// HighlightString returns code with Chroma syntax highlighting applied using
// the active theme. It returns code unchanged when the lexer or formatter is
// unavailable, or when the theme disables chroma.
func (c *Context) HighlightString(code, lang string) string {
	if c == nil || c.app == nil {
		return code
	}
	s := c.app.Theme().Chroma()
	if s == nil {
		return code
	}

	l := lexers.Get(lang)
	if l == nil {
		return code
	}
	l = chroma.Coalesce(l)
	f := formatters.Get("terminal256")

	iter, tokenErr := l.Tokenise(nil, code)
	if tokenErr != nil {
		return code
	}

	var buf strings.Builder
	if formatErr := f.Format(&buf, s, iter); formatErr != nil {
		return code
	}
	return buf.String()
}

// FprintHighlight writes highlighted code to w using the active theme.
//
// Errors:
//   - errors from writing to w
func (c *Context) FprintHighlight(w io.Writer, code, lang string) error {
	out := writer{w: w}
	out.println(c.HighlightString(code, lang))
	return out.Err()
}

// PrintHighlight writes highlighted code to [Context.IO.Out].
//
// Errors:
//   - errors from writing to [Context.IO.Out]
func (c *Context) PrintHighlight(code, lang string) error {
	return c.FprintHighlight(c.io.Out, code, lang)
}

// Highlight writes code to [Context.IO.Out] using a Chroma lexer named lang.
// When the lexer or formatter is unavailable, or when the [Theme] disables chroma,
// it writes the original code unchanged.
//
// Prefer [Context.HighlightString] when you need the highlighted string without
// writing, or [Context.FprintHighlight] to target an arbitrary writer.
//
// Errors:
//   - errors from writing to [Context.IO.Out]
func (c *Context) Highlight(code, lang string) error {
	return c.PrintHighlight(code, lang)
}
