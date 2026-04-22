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
	"testing"

	"github.com/stretchr/testify/require"
)

// minimalParseOKJSON matches [TestParseRejectsTrailingJSONData]'s valid
// prefix: single-variant manifest with one primitive-backed token.
const minimalParseOKJSON = `{
	"name":"fuzz-seed",
	"variants":{
		"dark":{
			"primitives":{"a":"#000000"},
			"tokens":{"text.primary":{"$primitive":"a"}}
		}
	}
}`

// FuzzParse ensures [Parse] never panics on arbitrary bytes.
func FuzzParse(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte(""))
	f.Add([]byte("{"))
	f.Add([]byte(minimalParseOKJSON))
	f.Add([]byte(`{"name":"x","variants":{"dark":{"primitives":{"a":"#000000"},"tokens":{"text.primary":{"$primitive":"a"}}}}}{"extra":true}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		require.NotPanics(t, func() {
			_, err := Parse(data)
			_ = err // fuzz contract is panic-freedom, not error semantics
		})
	})
}
