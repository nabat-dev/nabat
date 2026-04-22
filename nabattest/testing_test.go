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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
)

func newApp(t *testing.T) *nabat.App {
	t.Helper()
	io, _, _, _ := NewIO()
	app := nabat.MustNew("test", nabat.WithIO(io))
	app.MustCommand("noop", nabat.WithRun(func(c *nabat.Context) error { return nil }))
	return app
}

func TestRunParallelRejectsWithEnvVars(t *testing.T) {
	t.Parallel()

	err := RunParallel(t, newApp(t), []string{"noop"}, WithEnvVars(map[string]string{
		"NABATTEST_KEY": "value",
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RunParallel does not support WithEnvVars")
}

func TestRunWithNilTBRestoresEnvVars(t *testing.T) {
	const key = "NABATTEST_RESTORE"
	previous, hadPrevious := os.LookupEnv(key)
	t.Cleanup(func() {
		if hadPrevious {
			require.NoError(t, os.Setenv(key, previous))
			return
		}
		require.NoError(t, os.Unsetenv(key))
	})

	err := Run(nil, newApp(t), []string{"noop"}, WithEnvVars(map[string]string{
		key: "temporary",
	}))
	require.NoError(t, err)

	got, has := os.LookupEnv(key)
	if hadPrevious {
		assert.True(t, has)
		assert.Equal(t, previous, got)
		return
	}
	assert.False(t, has)
}

func TestNewTTYIO_reportsAsTTY(t *testing.T) {
	t.Parallel()

	ios, _, _, _ := NewTTYIO()
	assert.True(t, ios.IsStdinTTY())
	assert.True(t, ios.IsStdoutTTY())
	assert.True(t, ios.IsStderrTTY())
	assert.True(t, ios.CanPrompt())
}

func TestNewIO_defaultsToNonTTY(t *testing.T) {
	t.Parallel()

	ios, _, _, _ := NewIO()
	assert.False(t, ios.IsStdinTTY())
	assert.False(t, ios.IsStdoutTTY())
	assert.False(t, ios.IsStderrTTY())
	assert.False(t, ios.CanPrompt())
}
