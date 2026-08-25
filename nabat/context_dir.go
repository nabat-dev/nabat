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
)

type dirKey struct{}

// WithDir returns a child of parent that carries dir as the virtual working
// directory for the next [App.RunArgs] call. dir is stored as-is;
// nabattest.WithDir absolutizes first.
func WithDir(parent context.Context, dir string) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, dirKey{}, dir)
}

func dirFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	v, ok := ctx.Value(dirKey{}).(string)
	return v, ok && v != ""
}

// Dir returns the virtual working directory for this invocation.
// [WithDir] on the Go context sets it; otherwise it is the process
// working directory snapshot taken when the invocation started.
// A nil Context returns "".
func (c *Context) Dir() string {
	if c == nil {
		return ""
	}
	return c.dir
}

// Abs resolves path against the invocation directory. Absolute paths are
// cleaned. Relative paths are joined to [Context.Dir].
func (c *Context) Abs(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(c.Dir(), path)
}

// SetDir sets the virtual working directory. An empty dir is a no-op.
// A relative dir is made absolute against [os.Getwd] on this goroutine.
// If Getwd fails, Dir is unchanged. An absolute dir is cleaned.
// [Context.SetContext] leaves Dir unchanged.
func (c *Context) SetDir(dir string) {
	if c == nil || dir == "" {
		return
	}
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return
		}
		dir = filepath.Join(wd, dir)
	}
	c.dir = filepath.Clean(dir)
}
