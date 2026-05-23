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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func newVersionApp(t *testing.T, opts ...nabat.Option) (*nabat.App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	io, _, stdout, stderr := nabattest.NewIO()
	all := append([]nabat.Option{nabat.WithIO(io)}, opts...)
	app, err := nabat.New("myctl", all...)
	require.NoError(t, err)
	return app, stdout, stderr
}

func TestVersionCommandPrintsVersion(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3"))
	require.NoError(t, app.RunArgs(context.Background(), "version"))
	assert.Contains(t, stdout.String(), "myctl")
	assert.Contains(t, stdout.String(), "1.2.3")
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3"))
	require.NoError(t, app.RunArgs(context.Background(), "--version"))
	assert.Contains(t, stdout.String(), "1.2.3")
}

func TestVersionCustomCommandName(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3", nabat.WithVersionCommandName("ver")))
	require.NoError(t, app.RunArgs(context.Background(), "ver"))
	assert.Contains(t, stdout.String(), "1.2.3")
}

func TestVersionCustomFlagName(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3",
		nabat.WithVersionFlagName("ver"),
		nabat.WithVersionShorthand('V'),
	))
	require.NoError(t, app.RunArgs(context.Background(), "--ver"))
	assert.Contains(t, stdout.String(), "1.2.3")
}

func TestVersionWithoutShorthand(t *testing.T) {
	t.Parallel()

	app, _, _ := newVersionApp(t, nabat.WithVersion("1.2.3", nabat.WithoutVersionShorthand()))
	f := app.UnsafeRoot().Flags().ShorthandLookup("v")
	assert.Nil(t, f)
}

func TestWithVersionRejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion(""))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrInvalidOption)
	assert.Contains(t, err.Error(), "version string cannot be empty")
}

func TestWithVersionRejectsNilSubOption(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion("1.0.0", nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrNilOption)
}

func TestWithVersionCommandNameRejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion("1.0.0", nabat.WithVersionCommandName("")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestWithVersionFlagNameRejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion("1.0.0", nabat.WithVersionFlagName("")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestWithVersionShorthandRejectsNonASCII(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion("1.0.0", nabat.WithVersionShorthand('é')))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "printable ASCII")
}

func TestWithVersionRejectsBothDisabled(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion("1.0.0",
		nabat.WithoutVersionCommand(),
		nabat.WithoutVersionFlag(),
	))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrInvalidOption)
	assert.Contains(t, err.Error(), "both flag and command")
}

func TestWithVersionShorthandConflictsWithoutShorthand(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithVersion("1.0.0",
		nabat.WithVersionShorthand('V'),
		nabat.WithoutVersionShorthand(),
	))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrInvalidOption)
}

func TestVersionWithoutCommandKeepsFlag(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3", nabat.WithoutVersionCommand()))
	for _, sub := range app.UnsafeRoot().Commands() {
		assert.NotEqual(t, "version", sub.Name(), "version subcommand must not be installed")
	}
	require.NoError(t, app.RunArgs(context.Background(), "--version"))
	assert.Contains(t, stdout.String(), "1.2.3")
}

func TestVersionWithoutFlagKeepsCommand(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3", nabat.WithoutVersionFlag()))
	assert.Nil(t, app.UnsafeRoot().Flags().Lookup("version"))
	require.NoError(t, app.RunArgs(context.Background(), "version"))
	assert.Contains(t, stdout.String(), "1.2.3")
}

func TestVersionRegisterRejectsDuplicateFlagName(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test",
		nabat.WithFlag("release", false),
		nabat.WithVersion("1.0.0", nabat.WithVersionFlagName("release")),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `flag --release collides`)
}

func TestVersionCustomFlagNameIdempotent(t *testing.T) {
	t.Parallel()

	app, stdout, _ := newVersionApp(t, nabat.WithVersion("1.2.3", nabat.WithVersionFlagName("ver")))
	require.NoError(t, app.RunArgs(context.Background(), "--ver"))
	assert.Contains(t, stdout.String(), "1.2.3")

	stdout.Reset()
	require.NoError(t, app.RunArgs(context.Background(), "--ver"))
	assert.Contains(t, stdout.String(), "1.2.3")
}

type fakeExtension struct {
	name    string
	initErr error
	called  bool
	initFn  func(nabat.AppSurface) error
}

func (f *fakeExtension) String() string { return f.name }

func (f *fakeExtension) Init(app nabat.AppSurface) error {
	f.called = true
	if f.initFn != nil {
		return f.initFn(app)
	}
	return f.initErr
}

func TestWithExtensionRejectsNil(t *testing.T) {
	t.Parallel()

	_, err := nabat.New("test", nabat.WithExtension(nil, nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, nabat.ErrNilOption)
}

func TestWithExtensionRunsInsideNew(t *testing.T) {
	t.Parallel()

	ext := &fakeExtension{name: "fake"}
	_, err := nabat.New("test", nabat.WithExtension(ext, nil))
	require.NoError(t, err)
	assert.True(t, ext.called, "Init should run during New")
}

func TestWithExtensionInitErrorSurfacesFromNew(t *testing.T) {
	t.Parallel()

	boom := errors.New("kaboom")
	ext := &fakeExtension{name: "boomer", initErr: boom}
	_, err := nabat.New("test", nabat.WithExtension(ext, nil))
	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "extension boomer")
}

func TestWithExtensionRunsInDeclarationOrder(t *testing.T) {
	t.Parallel()

	var order []string
	first := &fakeExtension{name: "first", initFn: func(nabat.AppSurface) error {
		order = append(order, "first")
		return nil
	}}
	second := &fakeExtension{name: "second", initFn: func(nabat.AppSurface) error {
		order = append(order, "second")
		return nil
	}}
	_, err := nabat.New("test",
		nabat.WithExtension(first, nil),
		nabat.WithExtension(second, nil),
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, order)
}

func TestWithExtensionCanRegisterSubcommand(t *testing.T) {
	t.Parallel()

	ext := &fakeExtension{name: "subadder", initFn: func(app nabat.AppSurface) error {
		app.MustCommand("hello", nabat.WithRun(func(*nabat.Context) error { return nil }))
		return nil
	}}
	app, err := nabat.New("test", nabat.WithExtension(ext, nil))
	require.NoError(t, err)
	found := false
	for _, sub := range app.UnsafeRoot().Commands() {
		if sub.Name() == "hello" {
			found = true
		}
	}
	assert.True(t, found, "extension subcommand should be registered")
}
