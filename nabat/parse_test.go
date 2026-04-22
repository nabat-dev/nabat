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

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyParseOptionsAppliesAllFlags(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	applyParseOptions(cmd, []ParseOption{
		WithAllowUnknownFlags(),
		WithTraverseChildren(true),
		WithDisableFlagParsing(),
	})

	assert.True(t, cmd.FParseErrWhitelist.UnknownFlags)
	assert.True(t, cmd.TraverseChildren)
	assert.True(t, cmd.DisableFlagParsing)
}

func TestApplyParseOptionsLeavesDefaultsWithoutOptions(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	applyParseOptions(cmd, nil)

	assert.False(t, cmd.FParseErrWhitelist.UnknownFlags)
	assert.False(t, cmd.TraverseChildren)
	assert.False(t, cmd.DisableFlagParsing)
}

func TestWithParseOptionsRejectsNilNestedOption(t *testing.T) {
	t.Parallel()

	_, err := New("myctl",
		WithCommand("proxy",
			WithParseOptions(nil),
			WithRun(func(c *Context) error { return nil }),
		),
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
	assert.ErrorContains(t, err, "parse option index 0")
}
