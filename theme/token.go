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
// dotted lowercase strings (for example "status.success") that identify
// role, not appearance. The constants below are Nabat's well-known set;
// token names are an open set, not an enum.
type Token string

// Well-known semantic tokens used by Nabat core consumers.
const (
	// StatusSuccess marks affirmative output (Context.Success, ok badges).
	StatusSuccess Token = "status.success"

	// StatusWarning marks warnings (Context.Warn, degraded operation).
	StatusWarning Token = "status.warning"

	// StatusError marks failure output (Context.Error, rejected commands).
	StatusError Token = "status.error"

	// StatusInfo marks neutral status narrative (Context.Info, retries).
	StatusInfo Token = "status.info"

	// TextPrimary styles primary body text and table/list/tree item text.
	TextPrimary Token = "text.primary"

	// TextSecondary styles descriptive text and prose.
	TextSecondary Token = "text.secondary"

	// TextTitle styles help titles, table headers, and section titles.
	TextTitle Token = "text.title"

	// TextLink styles hyperlinks.
	TextLink Token = "text.link"

	// AccentPrimary styles labels and key chrome accents.
	AccentPrimary Token = "accent.primary"

	// TextMuted styles de-emphasized chrome (borders, enumerators).
	TextMuted Token = "text.muted"

	// CodeSurface styles code block backgrounds.
	CodeSurface Token = "code.surface"

	// TableBorder styles characters drawn between table cells.
	TableBorder Token = "table.border"

	// TableHeader styles cells in a table header row.
	TableHeader Token = "table.header"

	// TableCell styles cells in a table data row.
	TableCell Token = "table.cell"

	// ListItem styles list item text.
	ListItem Token = "list.item"

	// ListEnumerator styles list enumerator markers.
	ListEnumerator Token = "list.enumerator"

	// TreeItem styles tree item text.
	TreeItem Token = "tree.item"

	// TreeEnumerator styles tree enumerator markers.
	TreeEnumerator Token = "tree.enumerator"

	// SpinnerActive styles the live [Spinner] icon. Unset themes fall
	// through to [StatusInfo] via [DefaultAliases].
	SpinnerActive Token = "spinner.active"

	// StatusActive styles the spinner icon on active [Status] rows.
	// Unset themes fall through to [StatusInfo] via [DefaultAliases].
	StatusActive Token = "status.active"
)
