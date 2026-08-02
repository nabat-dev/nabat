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
	"reflect"
	"testing"
	"unsafe"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nabat.dev/theme"
)

func TestAdhocInputMethodsRequireInteractiveTerminal(t *testing.T) {
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		if _, err := c.Input("name"); err == nil {
			t.Fatalf("expected Input to fail in non-interactive mode")
		}
		// Select and MultiSelect now accept a fallback and succeed in
		// non-interactive mode; verify they return the fallback.
		env, err := c.Select("env", []string{"staging", "production"}, "staging")
		if err != nil {
			t.Fatalf("expected Select to succeed with fallback in non-interactive mode, got: %v", err)
		}
		if env != "staging" {
			t.Fatalf("expected Select to return fallback 'staging', got: %q", env)
		}
		targets, err := c.MultiSelect("targets", []string{"a", "b"}, []string{"a"})
		if err != nil {
			t.Fatalf("expected MultiSelect to succeed with fallback in non-interactive mode, got: %v", err)
		}
		if len(targets) != 1 || targets[0] != "a" {
			t.Fatalf("expected MultiSelect to return fallback [a], got: %v", targets)
		}
		if _, errText := c.TextInput("notes"); errText == nil {
			t.Fatalf("expected TextInput to fail in non-interactive mode")
		}
		if _, errFile := c.FilePicker("config"); errFile == nil {
			t.Fatalf("expected FilePicker to fail in non-interactive mode")
		}
		if _, errConfirm := c.Confirm("continue?"); errConfirm == nil {
			t.Fatalf("expected Confirm to fail in non-interactive mode")
		}
		return nil
	}))

	err := Run(t, app, []string{"run"})
	require.NoError(t, err)
}

// TestPromptOptionValidationAggregatesConfigErrors covers the option-validation
// surface. Each subtest passes a deliberately invalid option (nil callback or
// out-of-range numeric value) and asserts that the prompt call returns a
// *ConfigErrors containing the relevant error substring.
func TestPromptOptionValidationAggregatesConfigErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		run    func(c *Context) error
		substr string
	}{
		{
			name: "Input nil validate and nil option",
			run: func(c *Context) error {
				_, err := c.Input("name", WithValidate[string](nil), nil)
				return err
			},
			substr: "WithValidate",
		},
		{
			name: "Input non-positive char limit",
			run: func(c *Context) error {
				_, err := c.Input("name", WithMaxChars(0))
				return err
			},
			substr: "WithMaxChars",
		},
		{
			name: "Confirm nil validate",
			run: func(c *Context) error {
				_, err := c.Confirm("ok?", WithValidate[bool](nil))
				return err
			},
			substr: "WithValidate",
		},
		{
			name: "Select non-positive height",
			run: func(c *Context) error {
				_, err := c.Select("env", []string{"a", "b"}, "a", WithHeight(-1))
				return err
			},
			substr: "WithHeight",
		},
		{
			name: "MultiSelect non-positive limit",
			run: func(c *Context) error {
				_, err := c.MultiSelect("targets", []string{"a"}, nil, WithLimit(0))
				return err
			},
			substr: "WithLimit",
		},
		{
			name: "TextInput nil validate",
			run: func(c *Context) error {
				_, err := c.TextInput("notes", WithValidate[string](nil))
				return err
			},
			substr: "WithValidate",
		},
		{
			name: "TextInput empty editor cmd",
			run: func(c *Context) error {
				_, err := c.TextInput("notes", WithEditorCmd("   "))
				return err
			},
			substr: "WithEditorCmd",
		},
		{
			name: "FilePicker nil validate",
			run: func(c *Context) error {
				_, err := c.FilePicker("config", WithValidate[string](nil))
				return err
			},
			substr: "WithValidate",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			io, _, _, _ := testIO()
			app := MustNew("test", WithIO(io))
			var got error
			app.MustCommand("run", WithRun(func(c *Context) error {
				got = tt.run(c)
				return nil
			}))

			require.NoError(t, Run(t, app, []string{"run"}))
			require.Error(t, got)

			var cfgErr *ConfigErrors
			require.ErrorAs(t, got, &cfgErr,
				"prompt option validation must return *ConfigErrors so callers can introspect issues")
			assert.True(t, cfgErr.HasIssues())
			assert.Contains(t, got.Error(), tt.substr,
				"aggregated error should mention the failing helper by name")
		})
	}
}

// TestPromptNilOptionWrapsErrInvalidOption locks in the nil-handling contract:
// a nil entry in opts produces an [ErrInvalidOption] wrapping [ErrNilOption].
func TestPromptNilOptionWrapsErrInvalidOption(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Input("name", nil)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.True(t, errors.Is(got, ErrInvalidOption),
		"nil prompt option must wrap ErrInvalidOption")
	assert.True(t, errors.Is(got, ErrNilOption),
		"nil prompt option must wrap ErrNilOption")
}

// formTheme reads huh.Form's unexported theme field; huh provides no getter.
func formTheme(form *huh.Form) huh.Theme {
	rv := reflect.ValueOf(form).Elem().FieldByName("theme")
	ptr := reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr()))
	v := ptr.Elem().Interface()
	if v == nil {
		return nil
	}
	th, ok := v.(huh.Theme)
	if !ok {
		return nil
	}
	return th
}

// TestApplyHuhThemeForAdHocPromptRemovesBorder verifies that all four border
// sides are stripped and PaddingLeft(1) is preserved, for both light and dark.
func TestApplyHuhThemeForAdHocPromptRemovesBorder(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app, err := New("test", WithIO(io), WithTheme(theme.Default))
	require.NoError(t, err)

	for _, isDark := range []bool{false, true} {
		name := map[bool]string{false: "light", true: "dark"}[isDark]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			form := huh.NewForm(huh.NewGroup(huh.NewInput()))
			app.applyHuhThemeForAdHocPrompt(form)

			th := formTheme(form)
			require.NotNil(t, th, "applyHuhThemeForAdHocPrompt must set a theme")
			s := th.Theme(isDark)
			require.NotNil(t, s)

			slots := []struct {
				label        string
				top          bool
				right        bool
				bottom       bool
				left         bool
				paddingLeft  int
				checkPadding bool
			}{
				{
					label:        "Focused.Base",
					top:          s.Focused.Base.GetBorderTop(),
					right:        s.Focused.Base.GetBorderRight(),
					bottom:       s.Focused.Base.GetBorderBottom(),
					left:         s.Focused.Base.GetBorderLeft(),
					paddingLeft:  s.Focused.Base.GetPaddingLeft(),
					checkPadding: true,
				},
				{
					label:  "Focused.Card",
					top:    s.Focused.Card.GetBorderTop(),
					right:  s.Focused.Card.GetBorderRight(),
					bottom: s.Focused.Card.GetBorderBottom(),
					left:   s.Focused.Card.GetBorderLeft(),
				},
				{
					label:  "Blurred.Base",
					top:    s.Blurred.Base.GetBorderTop(),
					right:  s.Blurred.Base.GetBorderRight(),
					bottom: s.Blurred.Base.GetBorderBottom(),
					left:   s.Blurred.Base.GetBorderLeft(),
				},
				{
					label:  "Blurred.Card",
					top:    s.Blurred.Card.GetBorderTop(),
					right:  s.Blurred.Card.GetBorderRight(),
					bottom: s.Blurred.Card.GetBorderBottom(),
					left:   s.Blurred.Card.GetBorderLeft(),
				},
			}
			for _, sl := range slots {
				assert.False(t, sl.top, "%s: border top must be off", sl.label)
				assert.False(t, sl.right, "%s: border right must be off", sl.label)
				assert.False(t, sl.bottom, "%s: border bottom must be off", sl.label)
				assert.False(t, sl.left, "%s: border left must be off", sl.label)
				if sl.checkPadding {
					assert.Equal(t, 1, sl.paddingLeft,
						"%s: PaddingLeft(1) must be preserved after border removal", sl.label)
				}
			}
		})
	}
}

// cachedPtrTheme simulates a Palette.Huh that returns a shared *huh.Styles.
type cachedPtrTheme struct{ styles *huh.Styles }

func (c *cachedPtrTheme) Theme(_ bool) *huh.Styles { return c.styles }

// TestApplyHuhThemeForAdHocPromptDoesNotMutateSourceStyles verifies that a
// Palette.Huh returning a cached *huh.Styles is not permanently mutated.
func TestApplyHuhThemeForAdHocPromptDoesNotMutateSourceStyles(t *testing.T) {
	t.Parallel()

	// Build an app that uses a cached-pointer theme via Palette.Huh.
	base := huh.ThemeFunc(huh.ThemeCharm).Theme(false)
	cached := &cachedPtrTheme{styles: base}
	require.True(t, cached.styles.Focused.Base.GetBorderLeft(),
		"precondition: ThemeCharm must have BorderLeft enabled")

	app, err := New("test",
		WithCustomTheme(theme.Theme{
			Variants: map[theme.Variant]theme.Palette{
				theme.VariantDark: {Huh: cached},
			},
		}),
	)
	require.NoError(t, err)

	// Apply the borderless wrapper.
	form := huh.NewForm(huh.NewGroup(huh.NewInput()))
	app.applyHuhThemeForAdHocPrompt(form)

	// The wrapped theme must clear the border.
	th := formTheme(form)
	require.NotNil(t, th)
	got := th.Theme(false)
	require.NotNil(t, got)
	assert.False(t, got.Focused.Base.GetBorderLeft(),
		"wrapped theme must have left border cleared for ad-hoc prompt")

	// The original cached *Styles pointer must be untouched.
	assert.True(t, cached.styles.Focused.Base.GetBorderLeft(),
		"applyHuhThemeForAdHocPrompt must not mutate the source *huh.Styles")
}
