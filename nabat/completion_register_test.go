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

package nabat_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func newCompletionApp(t *testing.T, opts ...nabat.Option) (*nabat.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	io, _, stdout, stderr := nabattest.NewIO()
	all := append([]nabat.Option{nabat.WithIO(io)}, opts...)
	app, err := nabat.New("myctl", all...)
	require.NoError(t, err)
	return app, stdout, stderr
}

func TestCompletionDisabledByDefault(t *testing.T) {
	t.Parallel()

	app, _, _ := newCompletionApp(t)
	for _, sub := range app.UnsafeRoot().Commands() {
		assert.NotEqual(t, "completion", sub.Name(), "completion subcommand must be opt-in")
	}
}

func TestCompletionBash(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion())
	require.NoError(t, app.RunArgs(context.Background(), "completion", "bash"))
	assert.Contains(t, stdout.String(), "bash")
}

func TestCompletionZsh(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion())
	require.NoError(t, app.RunArgs(context.Background(), "completion", "zsh"))
	assert.Contains(t, stdout.String(), "zsh")
}

func TestCompletionFish(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion())
	require.NoError(t, app.RunArgs(context.Background(), "completion", "fish"))
	assert.Contains(t, stdout.String(), "fish")
}

func TestCompletionPowerShell(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion())
	require.NoError(t, app.RunArgs(context.Background(), "completion", "powershell"))
	assert.NotEmpty(t, stdout.String())
}

func TestCompletionCustomCommandName(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion(nabat.WithCompletionName("shell")))
	require.NoError(t, app.RunArgs(context.Background(), "shell", "bash"))
	assert.Contains(t, stdout.String(), "bash")
}

func TestCompletionHidden(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion(nabat.WithCompletionHidden()))
	require.NoError(t, app.RunArgs(context.Background(), "--help"))
	assert.NotContains(t, stdout.String(), "completion")
}

func TestCompletionVisibleByDefault(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newCompletionApp(t, nabat.WithCompletion())
	require.NoError(t, app.RunArgs(context.Background(), "--help"))
	assert.Contains(t, stdout.String(), "completion")
}

func TestCompletionLongHelpHasInstructions(t *testing.T) {
	t.Parallel()

	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			t.Parallel()
			app, stdout, _ := newCompletionApp(t, nabat.WithCompletion())
			require.NoError(t, app.RunArgs(context.Background(), "completion", shell, "--help"))
			assert.NotEmpty(t, stdout.String())
		})
	}
}

func TestCompletionShellsRestriction(t *testing.T) {
	t.Parallel()

	app, _, _ := newCompletionApp(t, nabat.WithCompletion(nabat.WithCompletionShells("bash", "zsh")))
	var leafNames []string
	for _, sub := range app.UnsafeRoot().Commands() {
		if sub.Name() == "completion" {
			for _, leaf := range sub.Commands() {
				leafNames = append(leafNames, leaf.Name())
			}
			break
		}
	}
	require.NotEmpty(t, leafNames, "completion subcommand should be installed")
	assert.ElementsMatch(t, []string{"bash", "zsh"}, leafNames)
}

func TestCompletionWritesToOutputFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	out := filepath.Join(dir, "myctl.bash")

	app, _, _ := newCompletionApp(t, nabat.WithCompletion())
	require.NoError(t, app.RunArgs(context.Background(), "completion", "bash", "--output", out))

	// #nosec G304 -- out is a file we just created inside t.TempDir().
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "bash")
}
