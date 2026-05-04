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

package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// rawTheme mirrors the top-level shape of a multi-variant theme
// manifest as defined by schema/v1.json. It is the wire-level
// intermediate the loader unmarshals into before validating
// cross-field constraints, expanding per-variant defaults, and
// producing a [*Compiled].
type rawTheme struct {
	Schema      string                  `json:"$schema,omitempty"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Default     rawVariant              `json:"default,omitempty"`
	Variants    map[rawVariant]rawSlice `json:"variants"`
	PromptKnobs *rawPromptKnobs         `json:"promptKnobs,omitempty"`
}

// rawVariant is the runtime mode a variant entry is designed for.
// Keys in the "variants" map and the "default" field both use this
// type so the schema enum stays the only source of truth.
type rawVariant string

// Variant constants. The strings match the enum in schema/v1.json
// exactly; mismatches fail TestSchemaCompiles long before they reach
// production.
const (
	variantDark  rawVariant = "dark"
	variantLight rawVariant = "light"
	variantNoTTY rawVariant = "notty"
)

// rawSlice is one entry under "variants". It holds the per-variant
// primitives, tokens, alias overrides, and optional upstream preset
// names.
type rawSlice struct {
	Primitives map[string]string   `json:"primitives"`
	Tokens     map[string]rawStyle `json:"tokens"`
	Aliases    map[string]string   `json:"aliases,omitempty"`
	Chroma     string              `json:"chroma,omitempty"`
	Glamour    string              `json:"glamour,omitempty"`
	Huh        string              `json:"huh,omitempty"`
}

// rawPromptKnobs captures root-level non-color prompt settings.
type rawPromptKnobs struct {
	SelectedPrefix   string       `json:"selectedPrefix,omitempty"`
	UnselectedPrefix string       `json:"unselectedPrefix,omitempty"`
	Border           rawBorderRef `json:"border,omitempty"`
}

// rawStyle is the JSON form of one entry in the tokens map.
//
// Boolean modifier fields are pointers so the loader can distinguish
// "explicitly set to false" from "unset (inherit)" — the latter matters
// when a spec layers on top of a $token reference.
//
// Color fields are [rawColorRef] to accept the hex literal | $primitive |
// $token union the schema declares. The zero value of a [rawColorRef] is
// "absent" (no color set).
type rawStyle struct {
	Primitive        string       `json:"$primitive,omitempty"`
	Token            string       `json:"$token,omitempty"`
	Fg               rawColorRef  `json:"fg"`
	Bg               rawColorRef  `json:"bg"`
	BorderForeground rawColorRef  `json:"borderForeground"`
	BorderBackground rawColorRef  `json:"borderBackground"`
	Border           rawBorderRef `json:"border,omitempty"`
	Bold             *bool        `json:"bold,omitempty"`
	Italic           *bool        `json:"italic,omitempty"`
	Underline        *bool        `json:"underline,omitempty"`
	Strikethrough    *bool        `json:"strikethrough,omitempty"`
	Faint            *bool        `json:"faint,omitempty"`
	Blink            *bool        `json:"blink,omitempty"`
	Reverse          *bool        `json:"reverse,omitempty"`
	Text             string       `json:"text,omitempty"`
}

// rawColorRef is a JSON value that may be either a hex literal string
// or an object with one of "$primitive" / "$token". The schema's oneOf
// guarantees exactly one form per occurrence; this Go-side struct just
// captures whichever form actually appeared so the resolver can decide
// at lookup time.
//
// All three fields are mutually exclusive in valid manifests; the
// schema enforces it but the resolver also rejects multiple-set forms
// defensively in case a manifest bypassed the schema validator.
type rawColorRef struct {
	Hex       string
	Primitive string
	Token     string
}

func (c rawColorRef) isEmpty() bool {
	return c.Hex == "" && c.Primitive == "" && c.Token == ""
}

// UnmarshalJSON decodes either a JSON string (hex literal) or a JSON
// object with a $primitive or $token key. Empty input (the JSON `null`
// or omitted field) leaves the zero value in place.
func (c *rawColorRef) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	if data[0] == '"' {
		return json.Unmarshal(data, &c.Hex)
	}
	var obj struct {
		Primitive string `json:"$primitive"`
		Token     string `json:"$token"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&obj); err != nil {
		return err
	}
	c.Primitive = obj.Primitive
	c.Token = obj.Token
	return nil
}

// rawBorderRef is a string typed so omitempty works while keeping the
// border-preset enum readable at call sites that switch over its value.
// Empty means "no border preset specified"; the resolver leaves the
// existing lipgloss border in place in that case.
type rawBorderRef string

// unmarshalManifest decodes data into a rawTheme. It exists so the
// loader has a single point where decoder configuration
// ([json.Decoder.DisallowUnknownFields], strict number handling, etc.)
// can be tightened without chasing every call site.
//
// DisallowUnknownFields is on so a typo in a manifest field name fails
// at registry init rather than silently dropping the override at render
// time. The JSON Schema validator catches the same errors for external
// authors; this is a belt-and-suspenders check for in-repo manifests.
func unmarshalManifest(data []byte) (*rawTheme, error) {
	var rt rawTheme
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}
	// Require exactly one top-level JSON document. A second decode must
	// hit EOF (ignoring trailing whitespace only).
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("manifest: trailing data after top-level object")
		}
		return nil, err
	}
	return &rt, nil
}
