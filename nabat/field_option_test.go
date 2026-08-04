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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldOptionApply(t *testing.T) {
	t.Parallel()

	t.Run("WithHint sets placeholder (string)", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithHint("alice").apply(&pc))
		assert.Equal(t, "alice", pc.placeholder)
	})

	t.Run("WithHint sets placeholder (int)", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithHint(42).apply(&pc))
		assert.Equal(t, "42", pc.placeholder)
	})

	t.Run("WithHint sets placeholder (duration)", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithHint(30*time.Second).apply(&pc))
		assert.Equal(t, "30s", pc.placeholder)
	})

	t.Run("WithMaxChars sets charLimit", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithMaxChars(64).apply(&pc))
		assert.Equal(t, 64, pc.charLimit)
	})

	t.Run("WithMaxChars rejects zero", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		err := WithMaxChars(0).apply(&pc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithMaxChars")
	})

	t.Run("WithPassword sets password", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithPassword().apply(&pc))
		assert.True(t, pc.password)
	})

	t.Run("WithSuggestions sets suggestions", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithSuggestions("a", "b").apply(&pc))
		assert.Equal(t, []string{"a", "b"}, pc.suggestions)
	})

	t.Run("WithAffirmative sets affirmative label", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithAffirmative("Yes").apply(&pc))
		assert.Equal(t, "Yes", pc.affirmative)
	})

	t.Run("WithNegative sets negative label", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithNegative("No").apply(&pc))
		assert.Equal(t, "No", pc.negative)
	})

	t.Run("WithEditor sets editor flag and multiline mode", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithEditor().apply(&pc))
		assert.True(t, pc.editor)
		assert.Equal(t, widgetMultiline, pc.mode)
	})

	t.Run("WithEditorCmd sets editorCmd", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithEditorCmd("vim").apply(&pc))
		assert.Equal(t, "vim", pc.editorCmd)
	})

	t.Run("WithEditorCmd rejects blank", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		err := WithEditorCmd("   ").apply(&pc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithEditorCmd")
	})

	t.Run("WithEditorExtension sets editorExtension", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithEditorExtension(".md").apply(&pc))
		assert.Equal(t, ".md", pc.editorExtension)
	})

	t.Run("WithAllowedTypes sets allowedTypes", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithAllowedTypes(".go").apply(&pc))
		assert.Equal(t, []string{".go"}, pc.allowedTypes)
	})

	t.Run("WithDirAllowed sets dirAllowed", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithDirAllowed().apply(&pc))
		assert.True(t, pc.dirAllowed)
	})

	t.Run("WithCurrentDir sets currentDir", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithCurrentDir("/tmp").apply(&pc))
		assert.Equal(t, "/tmp", pc.currentDir)
	})

	t.Run("WithFiltering sets filtering", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithFiltering(true).applyToSelect(&pc))
		assert.True(t, pc.filtering)
	})

	t.Run("WithHeight sets height", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithHeight(10).applyToSelect(&pc))
		assert.Equal(t, 10, pc.height)
	})

	t.Run("WithHeight rejects zero", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		err := WithHeight(0).applyToSelect(&pc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithHeight")
	})

	t.Run("WithLimit sets limit", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithLimit(3).applyToSelect(&pc))
		assert.Equal(t, 3, pc.limit)
	})

	t.Run("WithLimit rejects zero", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		err := WithLimit(0).applyToSelect(&pc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithLimit")
	})

	t.Run("WithInlineString sets inline", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithInlineString().apply(&pc))
		assert.True(t, pc.inline)
	})

	t.Run("WithInlineBool sets inline", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithInlineBool().apply(&pc))
		assert.True(t, pc.inline)
	})

	t.Run("WithMultiline sets mode to multiline", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithMultiline().apply(&pc))
		assert.Equal(t, widgetMultiline, pc.mode)
	})

	t.Run("WithFilePicker sets mode to file picker", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithFilePicker().apply(&pc))
		assert.Equal(t, widgetFilePicker, pc.mode)
	})
}

func TestWithDefaultTyped(t *testing.T) {
	t.Parallel()

	t.Run("string default", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithDefault("anon").apply(&pc))
		assert.True(t, pc.hasFallback)
		assert.Equal(t, "anon", pc.fallback)
	})

	t.Run("bool default", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithDefault(false).apply(&pc))
		assert.True(t, pc.hasFallback)
		assert.Equal(t, false, pc.fallback)
	})

	t.Run("int default", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithDefault(42).apply(&pc))
		assert.True(t, pc.hasFallback)
		assert.Equal(t, 42, pc.fallback)
	})

	t.Run("duration default", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithDefault(30*time.Second).apply(&pc))
		assert.True(t, pc.hasFallback)
		assert.Equal(t, 30*time.Second, pc.fallback)
	})

	t.Run("typed enum default round-trips", func(t *testing.T) {
		t.Parallel()
		type Environment string
		const EnvProd Environment = "production"
		var pc promptConfig
		require.NoError(t, WithDefault(EnvProd).apply(&pc))
		assert.True(t, pc.hasFallback)
		v, ok := pc.fallback.(Environment)
		assert.True(t, ok)
		assert.Equal(t, EnvProd, v)
	})
}

func TestWithInitialTyped(t *testing.T) {
	t.Parallel()

	t.Run("string initial", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithInitial("alice").apply(&pc))
		assert.True(t, pc.hasInitial)
		assert.Equal(t, "alice", pc.initial)
		assert.False(t, pc.hasFallback)
	})

	t.Run("bool initial true", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithInitial(true).apply(&pc))
		assert.True(t, pc.hasInitial)
		assert.Equal(t, true, pc.initial)
	})

	t.Run("explicit zero is tracked", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithInitial(0).apply(&pc))
		assert.True(t, pc.hasInitial, "explicit zero must set hasInitial")
		assert.Equal(t, 0, pc.initial)
	})

	t.Run("explicit false is tracked", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithInitial(false).apply(&pc))
		assert.True(t, pc.hasInitial, "explicit false must set hasInitial")
		assert.Equal(t, false, pc.initial)
	})
}

func TestWithPrefillTyped(t *testing.T) {
	t.Parallel()

	t.Run("sets both fallback and initial", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithPrefill("2").apply(&pc))
		assert.True(t, pc.hasFallback)
		assert.Equal(t, "2", pc.fallback)
		assert.True(t, pc.hasInitial)
		assert.Equal(t, "2", pc.initial)
	})
}

func TestApplyInitialSeedsTarget(t *testing.T) {
	t.Parallel()

	t.Run("overwrites existing pointer value", func(t *testing.T) {
		t.Parallel()
		out := "old"
		pc := promptConfig{initial: "new", hasInitial: true}
		applyInitial(&out, pc)
		assert.Equal(t, "new", out)
	})

	t.Run("no-op when hasInitial is false", func(t *testing.T) {
		t.Parallel()
		out := "keep"
		applyInitial(&out, promptConfig{})
		assert.Equal(t, "keep", out)
	})

	t.Run("confirm starts on Yes", func(t *testing.T) {
		t.Parallel()
		var out bool
		pc := promptConfig{initial: true, hasInitial: true}
		_ = buildConfirmField(&out, pc)
		assert.True(t, out, "WithInitial(true) must seed confirm target to true")
	})

	t.Run("input starts with seed", func(t *testing.T) {
		t.Parallel()
		var out string
		pc := promptConfig{text: "Name", initial: "alice", hasInitial: true}
		_ = buildInputField(&out, pc)
		assert.Equal(t, "alice", out)
	})
}

func TestWithDefaultStillWorksNonInteractive(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got string
	app.MustCommand("run", WithRun(func(c *Context) error {
		var err error
		got, err = c.Input("name", WithDefault("x"))
		return err
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "x", got)
}

func TestWithPrefillNonInteractiveReturnsFallback(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got string
	app.MustCommand("run", WithRun(func(c *Context) error {
		var err error
		got, err = c.Input("name", WithPrefill("seed"))
		return err
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	assert.Equal(t, "seed", got)
}

func TestWithInitialDoesNotProvideNonInteractiveFallback(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	var got error
	app.MustCommand("run", WithRun(func(c *Context) error {
		_, got = c.Input("name", WithInitial("seed"))
		return nil
	}))
	require.NoError(t, Run(t, app, []string{"run"}))
	require.Error(t, got)
	assert.Contains(t, got.Error(), "requires interactive terminal")
}

func TestWithValidateTypedDispatch(t *testing.T) {
	t.Parallel()

	t.Run("string validate", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		require.NoError(t, WithValidate(func(s string) error {
			return nil
		}).apply(&pc))
		require.NotNil(t, pc.validate)
		assert.NoError(t, pc.validate("ok"))
	})

	t.Run("nil validate rejected", func(t *testing.T) {
		t.Parallel()
		var pc promptConfig
		err := WithValidate[string](nil).apply(&pc)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithValidate")
	})
}

func TestNilOptionAggregation(t *testing.T) {
	t.Parallel()

	var pc promptConfig
	err := applyFieldOptions("test", []FieldOption[string]{nil, WithHint("ok"), nil}, &pc)
	require.Error(t, err)
	var cfgErr *ConfigErrors
	require.ErrorAs(t, err, &cfgErr)
	assert.Equal(t, 2, len(cfgErr.Unwrap()))
}
