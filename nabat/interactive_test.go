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

func TestAdhocInputMethodsRequireInteractiveTerminal(t *testing.T) {
	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	app.MustCommand("run", WithRun(func(c *Context) error {
		if _, err := c.Input("name"); err == nil {
			t.Fatalf("expected Input to fail in non-interactive mode")
		}
		// Select and MultiSelect now accept a fallback and succeed in
		// non-interactive mode; verify they return the fallback.
		env, err := Select(c, "env", []string{"staging", "production"}, "staging")
		if err != nil {
			t.Fatalf("expected Select to succeed with fallback in non-interactive mode, got: %v", err)
		}
		if env != "staging" {
			t.Fatalf("expected Select to return fallback 'staging', got: %q", env)
		}
		targets, err := MultiSelect(c, "targets", []string{"a", "b"}, []string{"a"})
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
				_, err := Select(c, "env", []string{"a", "b"}, "a", WithHeight(-1))
				return err
			},
			substr: "WithHeight",
		},
		{
			name: "MultiSelect non-positive limit",
			run: func(c *Context) error {
				_, err := MultiSelect(c, "targets", []string{"a"}, nil, WithLimit(0))
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
