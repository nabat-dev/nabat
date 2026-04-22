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

package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"nabat.dev/theme"
)

// TestUnicodeLevelsAreOrdered locks the monotonic ordering contract:
// higher tiers include the lower ones, so consumers can compare with
// less-than / greater-than for "at least N".
func TestUnicodeLevelsAreOrdered(t *testing.T) {
	t.Parallel()

	assert.True(t, theme.UnicodeASCII < theme.UnicodeWide,
		"UnicodeASCII < UnicodeWide invariant broke")
	assert.True(t, theme.UnicodeWide < theme.UnicodeEmoji,
		"UnicodeWide < UnicodeEmoji invariant broke")
}

// TestCapabilitiesZeroValueIsConservative documents the zero-value
// behavior: every field defaults to the safer (less-feature) value
// so consumers branching on Capabilities never get surprised by
// undetected facts.
func TestCapabilitiesZeroValueIsConservative(t *testing.T) {
	t.Parallel()

	var c theme.Capabilities
	assert.False(t, c.Dark, "Dark zero-value should be false (safer for unknown background)")
	assert.False(t, c.Interactive, "Interactive zero-value should be false")
	assert.False(t, c.Hyperlinks, "Hyperlinks zero-value should be false")
	assert.False(t, c.ReducedMotion, "ReducedMotion zero-value should be false")
	assert.Equal(t, theme.UnicodeASCII, c.Unicode, "Unicode zero-value should be UnicodeASCII (no UTF-8 assumed)")
	assert.Equal(t, 0, c.Width, "Width zero-value should be 0 (unknown)")
	assert.Empty(t, c.BackgroundHex, "BackgroundHex zero-value should be empty (unknown)")
}
