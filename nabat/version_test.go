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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
)

// TestVersionWithBuildInfo verifies that explicit commit/date overrides from
// [WithVersionCommit] and [WithVersionCommitDate] flow through resolveBuildInfo
// into the rendered text-mode output.
func TestVersionWithBuildInfo(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3",
		nabat.WithVersionCommit("abc1234"),
		nabat.WithVersionCommitDate("2024-03-15T10:00:00Z"),
	))
	require.NoError(t, app.RunArgs(context.Background(), "version"))
	got := stdout.String()
	assert.Contains(t, got, "1.2.3")
	assert.Contains(t, got, "abc1234")
	assert.Contains(t, got, "2024-03-15")
}

// TestVersionShortFormat verifies that --format short renders only the version
// string, with no name prefix or build-info suffix.
func TestVersionShortFormat(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3"))
	require.NoError(t, app.RunArgs(context.Background(), "version", "--format", "short"))
	assert.Equal(t, "1.2.3", strings.TrimSpace(stdout.String()))
}

// TestVersionFormatJSON verifies that --format json emits the buildInfoJSON
// shape with version, goVersion, and platform fields populated.
func TestVersionFormatJSON(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3",
		nabat.WithVersionCommit("abc1234"),
		nabat.WithVersionCommitDate("2024-03-15T10:00:00Z"),
	))
	require.NoError(t, app.RunArgs(context.Background(), "version", "--format", "json"))
	got := stdout.String()
	assert.Contains(t, got, `"version"`)
	assert.Contains(t, got, "1.2.3")
	assert.Contains(t, got, `"goVersion"`)
	assert.Contains(t, got, `"platform"`)
}
