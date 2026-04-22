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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanEnvNamesTrimsAndDropsEmpty(t *testing.T) {
	t.Parallel()

	got := cleanEnvNames([]string{" TARGET ", "", " \t ", "SECOND"})
	assert.Equal(t, []string{"TARGET", "SECOND"}, got)
}

func TestFieldConfigEnvUsageFragmentBuildsPrefixedAndAliases(t *testing.T) {
	t.Parallel()

	cfg := fieldConfig{
		envEnabled:  true,
		envPrefixed: []string{"target-name"},
		envLiteral:  []string{"GITHUB_TOKEN"},
	}
	assert.Equal(t, "(env: MYAPP_TARGET_NAME, GITHUB_TOKEN)", cfg.envUsageFragment("MYAPP_"))
}

func TestApplyArgOptionsAggregatesErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	_, err := applyArgOptions([]ArgOption{
		nil,
		argOptionFn(func(*argSpec) error { return sentinel }),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
	assert.ErrorIs(t, err, sentinel)
}

func TestApplyFlagOptionsAggregatesErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("flag boom")
	_, err := applyFlagOptions([]FlagOption{
		nil,
		flagOptionFn(func(*flagSpec) error { return sentinel }),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
	assert.ErrorIs(t, err, sentinel)
}
