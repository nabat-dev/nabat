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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"gopkg.in/yaml.v3"
)

// Format names a structured encoding used by [Context.Encode]. It is a type-safe
// enum; the zero value is invalid so accidentally defaulting to JSON is
// impossible.
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
//   - "nabat: json encoding failed: ..." when [encoding/json.MarshalIndent] fails
//   - errors from writing to [Context.IO.Out]
func (c *Context) JSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("nabat: json encoding failed: %w", err)
	}
	w := writer{w: c.io.Out}
	w.println(c.highlight(string(b), "json"))
	return w.Err()
}

// YAML writes v as YAML to [Context.IO.Out], with highlighting behavior like
// [Context.JSON].
//
// Errors:
//   - "nabat: yaml encoding failed: ..." when marshaling fails
//   - errors from writing to [Context.IO.Out]
func (c *Context) YAML(v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("nabat: yaml encoding failed: %w", err)
	}
	w := writer{w: c.io.Out}
	w.println(c.highlight(strings.TrimRight(string(b), "\n"), "yaml"))
	return w.Err()
}

// TOML writes v as TOML to [Context.IO.Out], with highlighting behavior like
// [Context.JSON].
//
// Errors:
//   - "nabat: toml encoding failed: ..." when encoding fails
//   - errors from writing to [Context.IO.Out]
func (c *Context) TOML(v any) error {
	var buf strings.Builder
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return fmt.Errorf("nabat: toml encoding failed: %w", err)
	}
	w := writer{w: c.io.Out}
	w.println(c.highlight(strings.TrimRight(buf.String(), "\n"), "toml"))
	return w.Err()
}

// Encode writes v using [FormatJSON], [FormatYAML], or [FormatTOML], delegating to
// [Context.JSON], [Context.YAML], or [Context.TOML].
//
// Example:
//
//	return c.Encode(payload, FormatJSON)
//
// Errors:
//   - "nabat: unknown format %q" when f is not one of the [Format] constants
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

// Highlight writes code to [Context.IO.Out] using a Chroma lexer named lang.
// When the lexer or formatter is unavailable, or when the [Theme] disables chroma,
// it writes the original code unchanged.
//
// Errors:
//   - errors from writing to [Context.IO.Out]
func (c *Context) Highlight(code, lang string) error {
	w := writer{w: c.io.Out}
	w.println(c.highlight(code, lang))
	return w.Err()
}

func (c *Context) highlight(code, lang string) string {
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
