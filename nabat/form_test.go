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
	"time"

	"charm.land/huh/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormFallbackWalk(t *testing.T) {
	t.Parallel()

	var (
		name    string
		proceed bool
		count   int
		timeout time.Duration
	)
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&name, "Name", "", WithDefault("alice")),
			WithFormField(&proceed, "Proceed?", "", WithDefault(true)),
			WithFormField(&count, "Count", "", WithDefault(5)),
			WithFormField(&timeout, "Timeout", "", WithDefault(30*time.Second)),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "alice", name)
	assert.True(t, proceed)
	assert.Equal(t, 5, count)
	assert.Equal(t, 30*time.Second, timeout)
}

func TestFormMissingFallbackAggregates(t *testing.T) {
	t.Parallel()

	var name string
	var count int
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&name, "Name", ""),
			WithFormField(&count, "Count", ""),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, formErr)

	var cfgErr *ConfigErrors
	require.ErrorAs(t, formErr, &cfgErr)
	assert.Equal(t, 2, len(cfgErr.Unwrap()))
	assert.Contains(t, formErr.Error(), "Name")
	assert.Contains(t, formErr.Error(), "Count")
}

func TestFormSelectStringSingle(t *testing.T) {
	t.Parallel()

	var env string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithSelectField(&env, "Environment", "",
				[]string{"staging", "production"},
				"staging",
				WithFiltering(true),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "staging", env)
}

func TestFormSelectStringMulti(t *testing.T) {
	t.Parallel()

	var tags []string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithMultiSelectField(&tags, "Tags", "",
				[]string{"go", "rust", "ts"},
				[]string{"go", "rust"},
				WithLimit(2),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, []string{"go", "rust"}, tags)
}

func TestFormSelectTypedEnumSingle(t *testing.T) {
	t.Parallel()

	type Severity int
	const (
		SevLow    Severity = 1
		SevMedium Severity = 3
		SevHigh   Severity = 5
	)

	var sev Severity
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithSelectField(&sev, "Severity", "",
				[]Severity{SevLow, SevMedium, SevHigh},
				SevLow,
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, SevLow, sev)
}

func TestFormSelectTypedEnumMulti(t *testing.T) {
	t.Parallel()

	type Tag string
	var tags []Tag
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithMultiSelectField(&tags, "Tags", "",
				[]Tag{"go", "rust", "ts"},
				[]Tag{"go"},
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, []Tag{"go"}, tags)
}

func TestFormSelectIntElement(t *testing.T) {
	t.Parallel()

	var priority int
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithSelectField(&priority, "Priority", "",
				[]int{1, 2, 3},
				2,
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, 2, priority)
}

func TestFormNonInteractiveWritesFallback(t *testing.T) {
	t.Parallel()

	var name string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&name, "Name", "", WithDefault("fallback")),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "fallback", name)
}

func TestFormNilOptionReportsConfigErrors(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(nil, WithFormTitle("x"))
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, formErr)
	var cfgErr *ConfigErrors
	require.ErrorAs(t, formErr, &cfgErr)
}

func TestFormFieldNilTargetReportsError(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(WithFormField[string](nil, "Name", ""))
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, formErr)
	assert.Contains(t, formErr.Error(), "target cannot be nil")
}

func TestFormChrome(t *testing.T) {
	t.Parallel()

	var fc formConfig
	require.NoError(t, WithFormTitle("Deploy").apply(&fc))
	require.NoError(t, WithFormDescription("Configure deployment.").apply(&fc))
	require.NoError(t, WithFormSubmit("Go").apply(&fc))
	require.NoError(t, WithFormCancel("Abort").apply(&fc))

	assert.Equal(t, "Deploy", fc.title)
	assert.Equal(t, "Configure deployment.", fc.description)
	assert.Equal(t, "Go", fc.submit)
	assert.Equal(t, "Abort", fc.cancel)
}

func TestFormFieldStringWithMultilineMode(t *testing.T) {
	t.Parallel()

	var notes string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&notes, "Notes", "Release notes",
				WithMultiline(),
				WithDefault(""),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "", notes)
}

func TestFormFieldStringWithFilePicker(t *testing.T) {
	t.Parallel()

	var cert string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&cert, "Certificate", "",
				WithFilePicker(),
				WithAllowedTypes(".pem", ".crt"),
				WithDefault(""),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "", cert)
}

func TestFormNoteNonInteractiveSkips(t *testing.T) {
	t.Parallel()

	var name string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormNote("Checklist", "- Step 1\n- Step 2"),
			WithFormField(&name, "Name", "", WithDefault("alice")),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "alice", name)
}

// WithFormNote satisfies GroupOption so it can be placed inside WithFormGroup.
func TestFormNoteInGroup(t *testing.T) {
	t.Parallel()

	var name string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormGroup(
				WithGroupTitle("Step 1"),
				WithFormNote("Info", "Read this."),
				WithFormField(&name, "Name", "", WithDefault("bob")),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "bob", name)
}

func TestFormAccessibleStored(t *testing.T) {
	t.Parallel()

	var fc formConfig
	require.NoError(t, WithFormAccessible().apply(&fc))
	assert.True(t, fc.accessible)
}

func TestFormGroupMultiPageFallbackWalk(t *testing.T) {
	t.Parallel()

	var name, email string
	var env string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormGroup(
				WithGroupTitle("Identity"),
				WithFormField(&name, "Name", "", WithDefault("alice")),
				WithFormField(&email, "Email", "", WithDefault("alice@example.com")),
			),
			WithFormGroup(
				WithGroupTitle("Deployment"),
				WithSelectField(&env, "Environment", "",
					[]string{"staging", "production"},
					"staging",
				),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "alice", name)
	assert.Equal(t, "alice@example.com", email)
	assert.Equal(t, "staging", env)
}

func TestFormGroupChromeStoredOnGroup(t *testing.T) {
	t.Parallel()

	var fg formGroup
	require.NoError(t, WithGroupTitle("Step A").applyToGroup(&fg))
	require.NoError(t, WithGroupDescription("Configure A").applyToGroup(&fg))
	assert.Equal(t, "Step A", fg.title)
	assert.Equal(t, "Configure A", fg.description)
}

// Chrome precedence: WithFormTitle falls through to first group only when the
// first group has no WithGroupTitle.
func TestFormChromeFirstGroupPrecedence(t *testing.T) {
	t.Parallel()

	// Form-level title falls through when group has no title.
	var fc formConfig
	require.NoError(t, WithFormTitle("Form Title").apply(&fc))
	require.NoError(t, WithFormField[string](new(string), "F", "", WithDefault("")).apply(&fc))
	require.Equal(t, "Form Title", fc.title)
	require.Equal(t, "", fc.groups[0].title)

	// Form-level title does NOT override group title.
	var fc2 formConfig
	require.NoError(t, WithFormTitle("Form Title").apply(&fc2))
	require.NoError(t, WithFormGroup(
		WithGroupTitle("Group Title"),
		WithFormField[string](new(string), "F", "", WithDefault("")),
	).apply(&fc2))
	require.Equal(t, "Group Title", fc2.groups[0].title)
}

// Fields added directly to Form and inside a group land in the right group.
func TestFormMixDirectAndGroupFields(t *testing.T) {
	t.Parallel()

	var direct, grouped string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&direct, "Direct", "", WithDefault("d")),
			WithFormGroup(
				WithGroupTitle("Page 2"),
				WithFormField(&grouped, "Grouped", "", WithDefault("g")),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "d", direct)
	assert.Equal(t, "g", grouped)
}

func TestFormGroupNilOptionReportsError(t *testing.T) {
	t.Parallel()

	var fc formConfig
	err := WithFormGroup(nil).apply(&fc)
	require.Error(t, err)
	var cfgErr *ConfigErrors
	require.ErrorAs(t, err, &cfgErr)
}

func TestFormOptionsFuncStoredInConfig(t *testing.T) {
	t.Parallel()

	var pc promptConfig
	fn := func() []string { return []string{"a", "b"} }
	require.NoError(t, WithOptionsFunc(fn, nil).applyToSelect(&pc))
	assert.NotNil(t, pc.optionsFunc)
}

func TestFormOptionsFuncNonInteractiveFallsBackToDefault(t *testing.T) {
	t.Parallel()

	var state string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		country := "US"
		states := map[string][]string{
			"US": {"California", "Texas"},
			"CA": {"Ontario", "Quebec"},
		}
		formErr = c.Form(
			WithSelectField(&state, "State", "",
				nil,
				"California",
				WithOptionsFunc(func() []string { return states[country] }, &country),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "California", state)
}

func TestFormKeyMapStored(t *testing.T) {
	t.Parallel()

	km := huh.NewDefaultKeyMap()
	var fc formConfig
	require.NoError(t, WithFormKeyMap(km).apply(&fc))
	assert.Same(t, km, fc.keymap)
}

func TestFormTimeoutStored(t *testing.T) {
	t.Parallel()

	var fc formConfig
	require.NoError(t, WithFormTimeout(30*time.Second).apply(&fc))
	assert.Equal(t, 30*time.Second, fc.timeout)
}

func TestFormFilePickerShowFlagsStored(t *testing.T) {
	t.Parallel()

	var pc promptConfig
	pc.mode = widgetFilePicker
	require.NoError(t, WithShowHidden().apply(&pc))
	require.NoError(t, WithShowSize().apply(&pc))
	require.NoError(t, WithShowPermissions().apply(&pc))
	assert.True(t, pc.showHidden)
	assert.True(t, pc.showSize)
	assert.True(t, pc.showPermissions)
}

func TestFormShowHiddenRequiresFilePicker(t *testing.T) {
	t.Parallel()

	var pc promptConfig
	err := WithShowHidden().apply(&pc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithShowHidden requires WithFilePicker")
}

func TestFormShowSizeRequiresFilePicker(t *testing.T) {
	t.Parallel()

	var pc promptConfig
	err := WithShowSize().apply(&pc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithShowSize requires WithFilePicker")
}

func TestFormShowPermissionsRequiresFilePicker(t *testing.T) {
	t.Parallel()

	var pc promptConfig
	err := WithShowPermissions().apply(&pc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WithShowPermissions requires WithFilePicker")
}

func TestFormFilePickerShowFlagsViaFormField(t *testing.T) {
	t.Parallel()

	var cert string
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form(
			WithFormField(&cert, "Cert", "",
				WithFilePicker(),
				WithShowHidden(),
				WithShowSize(),
				WithShowPermissions(),
				WithDefault(""),
			),
		)
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
	assert.Equal(t, "", cert)
}

func TestParseFormFieldValue(t *testing.T) {
	t.Parallel()

	t.Run("int valid", func(t *testing.T) {
		t.Parallel()
		v, err := parseFormFieldValue[int]("42")
		require.NoError(t, err)
		assert.Equal(t, 42, v)
	})
	t.Run("int invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseFormFieldValue[int]("abc")
		assert.Error(t, err)
	})
	t.Run("int64 valid", func(t *testing.T) {
		t.Parallel()
		v, err := parseFormFieldValue[int64]("9999999999")
		require.NoError(t, err)
		assert.Equal(t, int64(9999999999), v)
	})
	t.Run("int64 invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseFormFieldValue[int64]("xyz")
		assert.Error(t, err)
	})
	t.Run("uint valid", func(t *testing.T) {
		t.Parallel()
		v, err := parseFormFieldValue[uint]("100")
		require.NoError(t, err)
		assert.Equal(t, uint(100), v)
	})
	t.Run("uint invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseFormFieldValue[uint]("-5")
		assert.Error(t, err)
	})
	t.Run("float64 valid", func(t *testing.T) {
		t.Parallel()
		v, err := parseFormFieldValue[float64]("3.14")
		require.NoError(t, err)
		assert.InDelta(t, 3.14, v, 1e-9)
	})
	t.Run("float64 invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseFormFieldValue[float64]("not-a-float")
		assert.Error(t, err)
	})
	t.Run("duration valid", func(t *testing.T) {
		t.Parallel()
		v, err := parseFormFieldValue[time.Duration]("30s")
		require.NoError(t, err)
		assert.Equal(t, 30*time.Second, v)
	})
	t.Run("duration invalid", func(t *testing.T) {
		t.Parallel()
		_, err := parseFormFieldValue[time.Duration]("bad")
		assert.Error(t, err)
	})
}

// Empty form (no fields) returns nil without error.
func TestFormEmptyReturnsNil(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var formErr error
	app.MustCommand("run", WithRun(func(c *Context) error {
		formErr = c.Form()
		return nil
	}))

	require.NoError(t, Run(t, app, []string{"run"}))
	require.NoError(t, formErr)
}
