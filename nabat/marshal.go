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
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Marshal encodes v in the given format and returns the raw bytes without
// writing or highlighting. Unknown formats and encoding failures return an
// error. Use [Context.HighlightString] for theme-aware chroma styling.
func Marshal(v any, f Format) ([]byte, error) {
	switch f {
	case FormatJSON:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("nabat: json encoding failed: %w", err)
		}
		return b, nil
	case FormatYAML:
		b, err := yaml.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("nabat: yaml encoding failed: %w", err)
		}
		return b, nil
	case FormatTOML:
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(v); err != nil {
			return nil, fmt.Errorf("nabat: toml encoding failed: %w", err)
		}
		return []byte(strings.TrimRight(buf.String(), "\n")), nil
	default:
		return nil, fmt.Errorf("nabat: unknown format %d", int(f))
	}
}
