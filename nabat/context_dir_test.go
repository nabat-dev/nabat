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
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{name: "nil parent", ctx: WithDir(nil, dir), want: dir}, //nolint:staticcheck // WithDir nil-parent guard
		{name: "empty falls back to getwd", ctx: WithDir(t.Context(), ""), want: wd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			io, _, _, _ := testIO()
			app := MustNew("test", WithIO(io))
			app.MustCommand("check", WithRun(func(c *Context) error {
				assert.Equal(t, tt.want, c.Dir())
				return nil
			}))
			require.NoError(t, app.RunArgs(tt.ctx, "check"))
		})
	}
}

func TestSetDir(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name    string
		dir     string
		wantDir string
		wantAbs string
	}{
		{
			name:    "empty is no-op",
			wantDir: wd,
			wantAbs: filepath.Join(wd, "file"),
		},
		{
			name:    "relative joins getwd",
			dir:     "subdir",
			wantDir: filepath.Clean(filepath.Join(wd, "subdir")),
			wantAbs: filepath.Clean(filepath.Join(wd, "subdir", "file")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			io, _, _, _ := testIO()
			app := MustNew("test", WithIO(io))
			c := app.NewBareContext()
			require.NotNil(t, c)
			c.SetDir(tt.dir)
			assert.Equal(t, tt.wantDir, c.Dir())
			assert.Equal(t, tt.wantAbs, c.Abs("file"))
		})
	}
}

func TestDirFromContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		ctx  context.Context
		want string
		ok   bool
	}{
		{name: "nil context"},
		{name: "empty dir", ctx: WithDir(t.Context(), "")},
		{name: "set dir", ctx: WithDir(t.Context(), "/tmp/project"), want: "/tmp/project", ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := dirFromContext(tt.ctx)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDirAndAbsNilContext(t *testing.T) {
	t.Parallel()
	var c *Context
	assert.Equal(t, "", c.Dir())

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "relative", path: "foo", want: "foo"},
		{name: "absolute", path: "/already/abs", want: "/already/abs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, c.Abs(tt.path))
		})
	}
}
