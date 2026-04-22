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

func TestArgOptionsNilRejected(t *testing.T) {
	t.Parallel()

	_, err := applyArgOptions([]ArgOption{
		ArgOptions(WithUsage("first")),
		nil,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestArgOptionsAppliesInOrder(t *testing.T) {
	t.Parallel()

	s, err := applyArgOptions([]ArgOption{
		ArgOptions(
			WithUsage("first"),
			WithUsage("second"),
		),
	})
	require.NoError(t, err)
	assert.Equal(t, "second", s.field.usage)
}

func TestFlagOptionsNilRejected(t *testing.T) {
	t.Parallel()

	_, err := applyFlagOptions([]FlagOption{
		FlagOptions(WithShort('a')),
		nil,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidOption)
}

func TestFlagOptionsAppliesInOrder(t *testing.T) {
	t.Parallel()

	s, err := applyFlagOptions([]FlagOption{
		FlagOptions(
			WithShort('a'),
			WithShort('b'),
		),
	})
	require.NoError(t, err)
	assert.Equal(t, 'b', s.field.short)
	assert.True(t, s.field.hasShort)
}

func TestAppOptionsNilRejected(t *testing.T) {
	t.Parallel()

	cfg, err := defaultConfig()
	require.NoError(t, err)
	cfg.name = "x"
	err = AppOptions(
		WithEnvPrefix("P_"),
		nil,
	).applyToConfig(cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestAppOptionsSecondOverridesPrefix(t *testing.T) {
	t.Parallel()

	cfg, err := defaultConfig()
	require.NoError(t, err)
	cfg.name = "app"
	require.NoError(t, AppOptions(
		WithEnvPrefix("FIRST_"),
		WithEnvPrefix("SECOND_"),
	).applyToConfig(cfg))
	assert.Equal(t, "SECOND_", cfg.envPrefix)
}

func TestCommandOptionsNilRejected(t *testing.T) {
	t.Parallel()

	_, err := applyCommandOptions([]CommandOption{
		CommandOptions(WithDescription("x"), nil),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestRootOptionsNilRejected(t *testing.T) {
	t.Parallel()

	cfg, err := defaultConfig()
	require.NoError(t, err)
	cfg.name = "x"
	err = RootOptions(
		WithDescription("root"),
		nil,
	).applyToConfig(cfg)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}

func TestWithCommandInitNilRejected(t *testing.T) {
	t.Parallel()

	_, err := applyCommandOptions([]CommandOption{WithCommandInit(nil)})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNilOption)
}
