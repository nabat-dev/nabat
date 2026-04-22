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

func TestWithCommandRegistersUnderRoot(t *testing.T) {
	t.Parallel()

	app, err := New("myctl",
		WithCommand("deploy", WithRun(func(c *Context) error { return nil })),
	)
	require.NoError(t, err)
	deploy, _, err := app.root.Find([]string{"deploy"})
	require.NoError(t, err)
	require.NotNil(t, deploy)
	assert.Equal(t, "deploy", deploy.Name())
}

func TestWithCommandNestsChildren(t *testing.T) {
	t.Parallel()

	app, err := New("myctl",
		WithCommand("cluster",
			WithDescription("Cluster management"),
			WithCommand("scale", WithRun(func(c *Context) error { return nil })),
			WithCommand("status", WithRun(func(c *Context) error { return nil })),
		),
	)
	require.NoError(t, err)

	cluster, _, err := app.root.Find([]string{"cluster"})
	require.NoError(t, err)
	require.NotNil(t, cluster)
	assert.Equal(t, "Cluster management", cluster.Short)

	scale, _, err := app.root.Find([]string{"cluster", "scale"})
	require.NoError(t, err)
	require.NotNil(t, scale)

	status, _, err := app.root.Find([]string{"cluster", "status"})
	require.NoError(t, err)
	require.NotNil(t, status)
}

func TestWithCommandDeepNesting(t *testing.T) {
	t.Parallel()

	app, err := New("myctl",
		WithCommand("a",
			WithCommand("b",
				WithCommand("c", WithRun(func(c *Context) error { return nil })),
			),
		),
	)
	require.NoError(t, err)

	c, _, err := app.root.Find([]string{"a", "b", "c"})
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, "c", c.Name())
}

func TestWithCommandAggregatesMultipleErrors(t *testing.T) {
	t.Parallel()

	_, err := New("myctl",
		WithCommand("deploy",
			WithArg("env", ""),
			WithFlag("env", false), // arg/flag collision
		),
		WithCommand("status",
			WithArg("region", ""),
			WithFlag("region", ""), // arg/flag collision
		),
	)
	require.Error(t, err)

	var cfgErrs *ConfigErrors
	require.ErrorAs(t, err, &cfgErrs)
	require.Len(t, cfgErrs.Unwrap(), 2, "both bad commands should surface in one *ConfigErrors")

	for _, e := range cfgErrs.Unwrap() {
		assert.ErrorIs(t, e, ErrArgFlagNameCollision)
	}
}

func TestWithCommandEmptyNameSurfacesError(t *testing.T) {
	t.Parallel()

	_, err := New("myctl",
		WithCommand("", WithRun(func(c *Context) error { return nil })),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command name cannot be empty")
}

func TestWithCommandComposesWithRootOptions(t *testing.T) {
	t.Parallel()

	app, err := New("myctl",
		WithDescription("My CLI"),
		WithFlag("verbose", false, WithPersistent()),
		WithCommand("deploy", WithRun(func(c *Context) error { return nil })),
	)
	require.NoError(t, err)
	assert.Equal(t, "My CLI", app.root.Short)
	require.NotNil(t, app.root.PersistentFlags().Lookup("verbose"))

	deploy, _, err := app.root.Find([]string{"deploy"})
	require.NoError(t, err)
	require.NotNil(t, deploy)
}

func TestWithCommandAcceptsCommandOnlyOptions(t *testing.T) {
	t.Parallel()

	// WithGroup, WithHidden, WithAliases are CommandOption-only; they would be
	// build errors at the New(...) level but must work inside WithCommand.
	app, err := New("myctl",
		WithCommand("hidden",
			WithHidden(),
			WithAliases("h"),
			WithGroup("admin"),
			WithRun(func(c *Context) error { return nil }),
		),
	)
	require.NoError(t, err)

	hidden, _, err := app.root.Find([]string{"hidden"})
	require.NoError(t, err)
	require.NotNil(t, hidden)
	assert.True(t, hidden.Hidden)
	assert.Equal(t, []string{"h"}, hidden.Aliases)
}
