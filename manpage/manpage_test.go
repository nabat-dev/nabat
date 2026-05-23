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

package manpage_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/manpage"
	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func newApp(t *testing.T, opts ...nabat.Option) (*nabat.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	io, _, stdout, stderr := nabattest.NewIO()
	all := append([]nabat.Option{nabat.WithIO(io)}, opts...)
	app := nabat.MustNew("myctl", all...)
	return app, stdout, stderr
}

func TestManSubcommandWritesRoffToStdout(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newApp(t, nabat.WithExtension(manpage.New()))
	require.NoError(t, nabattest.Run(t, app, []string{"man"}))
	assert.NotEmpty(t, stdout.String())
}

func TestManSubcommandListedInHelpByDefault(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newApp(t, nabat.WithExtension(manpage.New()))
	require.NoError(t, nabattest.Run(t, app, []string{"--help"}))
	assert.Contains(t, stdout.String(), "Generate man page")
}

func TestManSubcommandHiddenFromHelpWhenConfigured(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newApp(t, nabat.WithExtension(manpage.New(manpage.WithHidden())))
	require.NoError(t, nabattest.Run(t, app, []string{"--help"}))
	assert.NotContains(t, stdout.String(), "Generate man page")
}

func TestManSubcommandUsesCustomName(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newApp(t, nabat.WithExtension(manpage.New(manpage.WithCommandName("docs"))))
	require.NoError(t, nabattest.Run(t, app, []string{"docs"}))
	assert.NotEmpty(t, stdout.String())
}

func TestManSubcommandDefaultSectionFlag(t *testing.T) {
	t.Parallel()

	app := nabat.MustNew("myctl", nabat.WithExtension(manpage.New()))

	manCmd, _, err := app.UnsafeRoot().Find([]string{"man"})
	require.NoError(t, err)
	require.NotNil(t, manCmd)
	f := manCmd.Flags().Lookup("section")
	require.NotNil(t, f)
	assert.Equal(t, "1", f.DefValue)
}

func TestManSubcommandCustomDefaultSection(t *testing.T) {
	t.Parallel()

	app := nabat.MustNew("myctl", nabat.WithExtension(manpage.New(manpage.WithSection(8))))

	manCmd, _, err := app.UnsafeRoot().Find([]string{"man"})
	require.NoError(t, err)
	f := manCmd.Flags().Lookup("section")
	require.NotNil(t, f)
	assert.Equal(t, "8", f.DefValue)
}

func TestManSubcommandSectionFlagOverridesDefault(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newApp(t, nabat.WithExtension(manpage.New()))
	require.NoError(t, nabattest.Run(t, app, []string{"man", "--section", "5"}))
	assert.NotEmpty(t, stdout.String())
}

func TestManSubcommandWritesToOutputFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "myctl.1")

	app, _, _ := newApp(t, nabat.WithExtension(manpage.New()))
	require.NoError(t, nabattest.Run(t, app, []string{"man", "-o", outPath}))

	// #nosec G304 -- path is a file we just wrote inside t.TempDir().
	b, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}

func TestNewRejectsNilOption(t *testing.T) {
	t.Parallel()

	_, err := manpage.New(nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "option at index 0 is nil")
}

func TestNewRejectsEmptyCommandName(t *testing.T) {
	t.Parallel()

	_, err := manpage.New(manpage.WithCommandName(""))
	require.Error(t, err)
	assert.ErrorContains(t, err, "command name cannot be empty")
}

func TestNewRejectsSectionBelowRange(t *testing.T) {
	t.Parallel()

	_, err := manpage.New(manpage.WithSection(0))
	require.Error(t, err)
	assert.ErrorContains(t, err, "section must be between 1 and 9")
}

func TestNewRejectsSectionAboveRange(t *testing.T) {
	t.Parallel()

	_, err := manpage.New(manpage.WithSection(10))
	require.Error(t, err)
	assert.ErrorContains(t, err, "section must be between 1 and 9")
}
