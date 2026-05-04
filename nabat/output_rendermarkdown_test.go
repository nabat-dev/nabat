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
)

// Tests for (*App).renderMarkdown stay in package nabat because the method is
// unexported; [output_test.go] is package nabat_test for public API contract checks.

func TestRenderMarkdownEmptyReturnsEmpty(t *testing.T) {
	t.Parallel()

	app := MustNew("test")
	result := app.renderMarkdown("")
	assert.Empty(t, result)
}

func TestRenderMarkdownNonTTYReturnsContent(t *testing.T) {
	t.Parallel()

	io, _, _, _ := testIO()
	app := MustNew("test", WithIO(io))
	result := app.renderMarkdown("# Hello\n\nWorld\n")
	assert.Contains(t, result, "Hello")
}
