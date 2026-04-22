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

package theme

// Token names a semantic style slot in a [ResolvedTheme]. Tokens are
// dotted lowercase strings (for example "status.success", "text.primary")
// that identify the role a style plays rather than its appearance. The
// constants in this file are Nabat's well-known set; third-party themes
// and extensions may define and consume additional tokens — token names
// are an open set, not an enum.
//
// The named string type catches arbitrary-string misuse at [Builder.Set]
// and [ResolvedTheme.Style] call sites without preventing dynamic lookup
// by callers that already hold a string (for example a user manifest or
// a flag value).
type Token string

// Well-known semantic tokens used by the Nabat core consumers.
const (
	// StatusSuccess marks affirmative output: completed deploys,
	// "ok" badges, the leading symbol on Context.Success.
	StatusSuccess Token = "status.success"

	// StatusWarning marks warnings: degraded operation, deprecated APIs,
	// the leading symbol on Context.Warn.
	StatusWarning Token = "status.warning"

	// StatusError marks failure output: rejected commands, the leading
	// symbol on Context.Error, the "error:" prefix on uncaught errors.
	StatusError Token = "status.error"

	// StatusInfo marks neutral status narrative: retrying, connecting,
	// the leading symbol on Context.Info, version-line text.
	StatusInfo Token = "status.info"

	// TextPrimary styles primary body text, table cell values, and
	// list/tree item text.
	TextPrimary Token = "text.primary"

	// TextSecondary styles descriptive text and prose.
	TextSecondary Token = "text.secondary"

	// TextTitle styles help titles, table headers, and other prominent
	// section titles.
	TextTitle Token = "text.title"

	// TextLink styles hyperlinks.
	TextLink Token = "text.link"

	// AccentPrimary styles labels and key chrome accents.
	AccentPrimary Token = "accent.primary"

	// TextMuted styles de-emphasized chrome such as table borders,
	// list enumerators, and tree connectors.
	TextMuted Token = "text.muted"

	// CodeSurface styles code block backgrounds.
	CodeSurface Token = "code.surface"

	// TableBorder styles the characters drawn between table cells.
	TableBorder Token = "table.border"

	// TableHeader styles the cells in a table's header row.
	TableHeader Token = "table.header"

	// TableCell styles the cells in a table's data rows.
	TableCell Token = "table.cell"

	// ListItem styles list item text.
	ListItem Token = "list.item"

	// ListEnumerator styles list enumerator markers (•, -, 1., …).
	ListEnumerator Token = "list.enumerator"

	// TreeItem styles tree item text.
	TreeItem Token = "tree.item"

	// TreeEnumerator styles tree enumerator markers (├──, └──, …).
	TreeEnumerator Token = "tree.enumerator"
)
