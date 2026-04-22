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
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDeprecatedEmptyMessage(t *testing.T) {
	t.Parallel()

	_, err := New("testctl",
		WithFlag("old", "", WithDeprecated("")),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithDeprecated requires a non-empty message")
}

func TestWithDeprecatedWhitespaceOnlyMessage(t *testing.T) {
	t.Parallel()

	_, err := New("testctl",
		WithFlag("old", "", WithDeprecated("   \t")),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithDeprecated requires a non-empty message")
}

func TestWithShorthandDeprecatedWithoutShort(t *testing.T) {
	t.Parallel()

	_, err := New("testctl",
		WithFlag("config", "", WithDeprecatedShorthand("use --config")),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithDeprecatedShorthand requires WithShort")
}

func TestWithShorthandDeprecatedEmptyMessage(t *testing.T) {
	t.Parallel()

	_, err := New("testctl",
		WithFlag("config", "", WithShort('c'), WithDeprecatedShorthand("")),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithDeprecated requires a non-empty message")
}

func TestDeprecatedFlagEmitsWarning(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("testctl", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("legacy", "", WithDeprecated("use --new-flag instead")),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "--legacy", "x"})
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "deprecated")
	assert.Contains(t, s, "use --new-flag instead")
}

func TestShorthandDeprecatedEmitsWarning(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := MustNew("testctl", WithIO(NewIO(strings.NewReader(""), &out, &out)))
	app.MustCommand("deploy",
		WithFlag("config", "", WithShort('c'), WithDeprecatedShorthand("prefer --config")),
		WithRun(func(c *Context) error { return nil }),
	)

	err := Run(t, app, []string{"deploy", "-c", "path"})
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "shorthand")
	assert.Contains(t, s, "deprecated")
	assert.Contains(t, s, "prefer --config")
}
