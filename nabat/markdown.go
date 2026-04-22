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

import "fmt"

// Markdown renders content as markdown and writes it to [Context.IO.Out].
// When stdout is a terminal, glamour applies styles from the active theme's
// glamour accessor (see [theme.ResolvedTheme.Glamour] and
// [theme.ResolvedTheme.GlamourName]). When stdout is not a terminal, the
// raw markdown is written.
//
// Markdown returns a non-nil error only if writing the rendered output fails.
// If glamour fails to initialize, [App.renderMarkdown] falls back to raw content.
func (c *Context) Markdown(content string) error {
	rendered := c.app.renderMarkdown(content)
	_, err := fmt.Fprint(c.io.Out, rendered)
	return err
}
