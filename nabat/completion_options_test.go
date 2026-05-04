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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
)

func TestWithCompletionRejectsNilSubOption(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithCompletion(nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrNilOption)
}

func TestWithCompletionNameRejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithCompletion(nabat.WithCompletionName("")))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrInvalidOption)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestWithCompletionShellsRejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithCompletion(nabat.WithCompletionShells()))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrInvalidOption)
	assert.Contains(t, err.Error(), "at least one shell")
}

func TestWithCompletionShellsRejectsUnknown(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithCompletion(nabat.WithCompletionShells("bash", "tcsh")))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrInvalidOption)
	assert.Contains(t, err.Error(), `"tcsh"`)
	assert.Contains(t, err.Error(), "bash|zsh|fish|powershell")
}

func TestWithCompletionShellsAcceptsAllKnown(t *testing.T) {
	t.Parallel()

	for _, sh := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(sh, func(t *testing.T) {
			t.Parallel()
			_, err := nabat.New("test", nabat.WithCompletion(nabat.WithCompletionShells(sh)))
			require.NoError(t, err)
		})
	}
}
