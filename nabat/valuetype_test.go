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
)

func TestParseStringToType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		vt      valueType
		wantErr bool
		check   func(t *testing.T, v any)
	}{
		{
			name:  "int parses decimal string",
			input: "42",
			vt:    vtInt(),
			check: func(t *testing.T, v any) {
				t.Helper()
				assert.Equal(t, 42, v)
			},
		},
		{
			name:  "bool parses true",
			input: "true",
			vt:    vtBool(),
			check: func(t *testing.T, v any) {
				t.Helper()
				assert.Equal(t, true, v)
			},
		},
		{
			name:  "float parses decimal",
			input: "3.14",
			vt:    vtFloat(),
			check: func(t *testing.T, v any) {
				t.Helper()
				assert.Equal(t, 3.14, v)
			},
		},
		{
			name:  "string slice parses comma separated",
			input: "a,b,c",
			vt:    vtStringSlice(),
			check: func(t *testing.T, v any) {
				t.Helper()
				vs, ok := v.([]string)
				require.True(t, ok, "want []string, got %T", v)
				assert.Equal(t, []string{"a", "b", "c"}, vs)
			},
		},
		{
			name:  "empty string for string slice yields empty slice",
			input: "",
			vt:    vtStringSlice(),
			check: func(t *testing.T, v any) {
				t.Helper()
				vs, ok := v.([]string)
				require.True(t, ok, "want []string, got %T", v)
				assert.Empty(t, vs)
			},
		},
		{
			name:  "string preserves newlines (was: text)",
			input: "hello\nworld",
			vt:    vtString(),
			check: func(t *testing.T, v any) {
				t.Helper()
				assert.Equal(t, "hello\nworld", v)
			},
		},
		{
			name:  "string preserves path (was: file)",
			input: "/tmp/config.yaml",
			vt:    vtString(),
			check: func(t *testing.T, v any) {
				t.Helper()
				assert.Equal(t, "/tmp/config.yaml", v)
			},
		},
		{
			name:    "invalid int returns error",
			input:   "abc",
			vt:      vtInt(),
			wantErr: true,
		},
		{
			name:    "invalid bool returns error",
			input:   "maybe",
			vt:      vtBool(),
			wantErr: true,
		},
		{
			name:    "invalid float returns error",
			input:   "xyz",
			vt:      vtFloat(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := parseStringToType(tt.input, tt.vt)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			tt.check(t, v)
		})
	}
}

func TestValidateChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		vt      valueType
		value   any
		wantErr bool
	}{
		{
			name:    "select rejects value not in choices",
			vt:      vtSelect("a", "b"),
			value:   "c",
			wantErr: true,
		},
		{
			name:    "select accepts value in choices",
			vt:      vtSelect("a", "b"),
			value:   "a",
			wantErr: false,
		},
		{
			name:    "multi-select rejects choice not in list",
			vt:      vtMultiSelect("a", "b"),
			value:   []string{"a", "c"},
			wantErr: true,
		},
		{
			name:    "multi-select accepts subset of choices",
			vt:      vtMultiSelect("a", "b"),
			value:   []string{"a", "b"},
			wantErr: false,
		},
		{
			name:    "select with empty choices accepts any string",
			vt:      vtSelect(),
			value:   "anything",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateChoice(tt.vt, tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValueTypeHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vt   valueType
		want string
	}{
		{name: "string kind shows string hint", vt: vtString(), want: "string"},
		{name: "bool kind shows bool hint", vt: vtBool(), want: "bool"},
		{name: "int kind shows int hint", vt: vtInt(), want: "int"},
		{name: "float kind shows float hint", vt: vtFloat(), want: "float"},
		{name: "string slice shows ellipsis hint", vt: vtStringSlice(), want: "string..."},
		{name: "select joins choices with pipe", vt: vtSelect("a", "b"), want: "a|b"},
		{name: "multi-select joins choices with pipe and ellipsis", vt: vtMultiSelect("x", "y"), want: "x|y..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.vt.typeHint()
			assert.Equalf(t, tt.want, got, "typeHint() for kind %d", tt.vt.kind)
		})
	}
}
