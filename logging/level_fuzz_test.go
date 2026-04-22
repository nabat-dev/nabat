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

package logging

import (
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzParseLevel(f *testing.F) {
	f.Add("debug")
	f.Add("INFO")
	f.Add("warn")
	f.Add("error")
	f.Add("")
	f.Add("not-a-level")

	f.Fuzz(func(t *testing.T, s string) {
		got, err := ParseLevel(s)
		switch strings.ToLower(s) {
		case "debug":
			require.NoError(t, err)
			assert.Equal(t, slog.LevelDebug, got)
		case "info":
			require.NoError(t, err)
			assert.Equal(t, slog.LevelInfo, got)
		case "warn":
			require.NoError(t, err)
			assert.Equal(t, slog.LevelWarn, got)
		case "error":
			require.NoError(t, err)
			assert.Equal(t, slog.LevelError, got)
		default:
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrUnknownLevel))
			assert.Equal(t, slog.Level(0), got)
		}
	})
}
