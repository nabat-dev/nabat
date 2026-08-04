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

	"nabat.dev/theme"
)

// Icon bundles a glyph and a theme token for use with [Context.Badge].
// Apps map domain words ("running", "stopped") onto icons; Nabat does not
// interpret status strings. Use the [IconSuccess] family for common statuses,
// or [NewIcon] for custom chrome. [WithSpinnerIcons] does not affect Badge.
type Icon struct {
	glyph string
	token theme.Token
}

// NewIcon returns an [Icon] with the given glyph and theme token. Empty glyph
// or token fall back to [IconUnknown] / [theme.TextMuted] inside [Context.Badge].
func NewIcon(glyph string, tok theme.Token) Icon {
	return Icon{glyph: glyph, token: tok}
}

// Glyph returns the icon's Unicode symbol.
func (i Icon) Glyph() string { return i.glyph }

// Token returns the theme token used to style the icon.
func (i Icon) Token() theme.Token { return i.token }

// Predefined icons for [Context.Badge]. Glyph defaults match the defaults on
// [Icons] used by [Context.Spinner] and [Context.Status].
var (
	IconSuccess = NewIcon("✓", theme.StatusSuccess)
	IconError   = NewIcon("✗", theme.StatusError)
	IconWarning = NewIcon("!", theme.StatusWarning)
	IconInfo    = NewIcon("•", theme.StatusInfo)
	IconUnknown = NewIcon("?", theme.TextMuted)
)

// Field is a key/value pair for [Context.Fields].
type Field struct {
	Key   string
	Value string
}

type fieldsConfig struct {
	keyWidth int
}

// FieldsOption configures [Context.Fields].
type FieldsOption func(*fieldsConfig)

// WithFieldKeyWidth sets the padded width of field keys. When unset or <= 0,
// Fields uses the width of the longest key in the block.
func WithFieldKeyWidth(n int) FieldsOption {
	return func(c *fieldsConfig) {
		c.keyWidth = n
	}
}

// FieldBlock is an aligned key/value block produced by [Context.Fields].
// Call [FieldBlock.String] to compose it into other output, or
// [FieldBlock.Print] to write it to the context's stdout.
type FieldBlock struct {
	c    *Context
	text string
}

// String returns the pre-rendered block, including a trailing newline when
// the block is non-empty.
func (fb FieldBlock) String() string { return fb.text }

// Print writes the block to [Context.IO].Out.
func (fb FieldBlock) Print() {
	if fb.c == nil || fb.text == "" {
		return
	}
	fb.c.Print(fb.text)
}

// Badge renders a styled status chip: icon glyph plus label. IconInfo is a
// product chrome chip and is distinct from [Context.Info], which writes a
// diagnostic message to stderr.
//
// Example:
//
//	c.Printf("%s %s\n", c.Render(theme.TextTitle, name), c.Badge(nabat.IconSuccess, "running"))
func (c *Context) Badge(icon Icon, label string) string {
	glyph := icon.glyph
	if glyph == "" {
		glyph = IconUnknown.glyph
	}
	tok := icon.token
	if tok == "" {
		tok = theme.TextMuted
	}
	styled := c.Render(tok, glyph)
	if label == "" {
		return styled
	}
	return styled + " " + label
}

// Fields builds an aligned key/value block. Keys are styled with
// [theme.TextMuted]. Padding is applied to the plain key text before styling
// so ANSI width does not break alignment.
//
// Example:
//
//	c.Fields([]nabat.Field{
//	    {Key: "Backend", Value: "kind (docker)"},
//	    {Key: "Context", Value: "kind-deployah"},
//	    {Key: "Cloud provider", Value: c.Badge(nabat.IconSuccess, "running")},
//	}, nabat.WithFieldKeyWidth(14)).Print()
func (c *Context) Fields(fields []Field, opts ...FieldsOption) FieldBlock {
	cfg := fieldsConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	width := cfg.keyWidth
	if width <= 0 {
		for _, f := range fields {
			if w := visibleWidth(f.Key); w > width {
				width = w
			}
		}
	}

	var b strings.Builder
	keyStyle := c.Style(theme.TextMuted)
	for _, f := range fields {
		padded := padRight(f.Key, width)
		b.WriteString(keyStyle.Render(padded))
		if f.Value != "" {
			b.WriteString(" ")
			b.WriteString(f.Value)
		}
		b.WriteString("\n")
	}
	return FieldBlock{c: c, text: b.String()}
}
