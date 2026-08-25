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

package nabattest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
)

func TestRunParallelWithDir(t *testing.T) {
	t.Parallel()

	t.Run("dirA", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "marker"), []byte("A"), 0o600))

		io, _, out, _ := NewIO()
		app := nabat.MustNew("test", nabat.WithIO(io),
			nabat.WithCommand("read", nabat.WithRun(func(c *nabat.Context) error {
				data, err := os.ReadFile(c.Abs("marker"))
				if err != nil {
					return err
				}
				c.Print(string(data))
				return nil
			})))
		require.NoError(t, RunParallel(t, app, []string{"read"}, WithDir(dir)))
		assert.Equal(t, "A", out.String())
	})

	t.Run("dirB", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "marker"), []byte("B"), 0o600))

		io, _, out, _ := NewIO()
		app := nabat.MustNew("test", nabat.WithIO(io),
			nabat.WithCommand("read", nabat.WithRun(func(c *nabat.Context) error {
				data, err := os.ReadFile(c.Abs("marker"))
				if err != nil {
					return err
				}
				c.Print(string(data))
				return nil
			})))
		require.NoError(t, RunParallel(t, app, []string{"read"}, WithDir(dir)))
		assert.Equal(t, "B", out.String())
	})
}

func TestDirDefaultsToGetwd(t *testing.T) {
	t.Parallel()
	wd, err := os.Getwd()
	require.NoError(t, err)

	io, _, _, _ := NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io),
		nabat.WithCommand("dir", nabat.WithRun(func(c *nabat.Context) error {
			assert.Equal(t, wd, c.Dir())
			assert.Equal(t, filepath.Join(wd, "foo"), c.Abs("foo"))
			return nil
		})))
	require.NoError(t, RunParallel(t, app, []string{"dir"}))
}

func TestAbsAlreadyAbsolute(t *testing.T) {
	t.Parallel()
	io, _, _, _ := NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io),
		nabat.WithCommand("abs", nabat.WithRun(func(c *nabat.Context) error {
			assert.Equal(t, "/already/abs", c.Abs("/already/abs"))
			assert.Equal(t, "/a/b", c.Abs("/a/../a/b"))
			return nil
		})))
	require.NoError(t, RunParallel(t, app, []string{"abs"}))
}

func TestSetContextNilPreservesDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	io, _, _, _ := NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	require.NoError(t, app.OnPreRun(func(c *nabat.Context) error {
		c.SetContext(nil) //nolint:staticcheck // deliberately testing nil-context guard
		return nil
	}))
	app.MustCommand("check", nabat.WithRun(func(c *nabat.Context) error {
		assert.Equal(t, dir, c.Dir())
		return nil
	}))
	require.NoError(t, RunParallel(t, app, []string{"check"}, WithDir(dir)))
}

func TestWithDirEmptyReturnsError(t *testing.T) {
	t.Parallel()
	err := RunParallel(t, newApp(t), []string{"noop"}, WithDir(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithDir")
}

func TestContextWithDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	io, _, _, _ := NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	c := Context(t, app, WithDir(dir))
	require.NotNil(t, c)
	assert.Equal(t, dir, c.Dir())
	assert.Equal(t, filepath.Join(dir, "marker"), c.Abs("marker"))
}

func TestNilContextDirAndAbs(t *testing.T) {
	t.Parallel()
	var c *nabat.Context
	assert.Equal(t, "", c.Dir())
	assert.Equal(t, "foo", c.Abs("foo"))
	assert.Equal(t, "/already/abs", c.Abs("/already/abs"))
	c.SetDir("/tmp") // no panic
}
