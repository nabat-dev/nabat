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
// On a TTY, glamour styles come from the active theme; otherwise raw
// markdown is written. Returns an error only on write failure; glamour
// init failure falls back to raw content.
func (c *Context) Markdown(content string) error {
	rendered := c.app.renderMarkdown(content)
	_, err := fmt.Fprint(c.io.Out, rendered)
	return err
}
