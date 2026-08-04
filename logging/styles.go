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

package logging

import (
	"charm.land/lipgloss/v2"

	"nabat.dev/theme"
)

// Styles holds lipgloss styles for level badges and key/value pairs in
// structured log output. Build with [FromTheme] or by hand. Once built,
// Styles is immutable and safe to share across goroutines.
type Styles struct {
	// Debug styles the "DEBU" level badge.
	Debug lipgloss.Style

	// Info styles the "INFO" level badge.
	Info lipgloss.Style

	// Warn styles the "WARN" level badge.
	Warn lipgloss.Style

	// Error styles the "ERRO" level badge.
	Error lipgloss.Style

	// Key styles structured-log attribute keys.
	Key lipgloss.Style

	// Value styles structured-log attribute values.
	Value lipgloss.Style
}

// FromTheme derives [Styles] from a [theme.ResolvedTheme] using
// [theme.StatusInfo], [theme.StatusWarning], and [theme.StatusError] for
// badges, and [theme.AccentPrimary] / [theme.TextPrimary] for key=value
// pairs. Badges are fixed-width "DEBU" / "INFO" / "WARN" / "ERRO" labels
// rendered bold. Callers that need a different shape can build [Styles]
// directly.
func FromTheme(rt theme.ResolvedTheme) Styles {
	info := rt.Style(theme.StatusInfo)
	warn := rt.Style(theme.StatusWarning)
	er := rt.Style(theme.StatusError)
	return Styles{
		Debug: info.Faint(true).
			SetString("DEBU").
			Bold(true).
			MaxWidth(4),
		Info: info.
			SetString("INFO").
			Bold(true).
			MaxWidth(4),
		Warn: warn.
			SetString("WARN").
			Bold(true).
			MaxWidth(4),
		Error: er.
			SetString("ERRO").
			Bold(true).
			MaxWidth(4),
		Key:   rt.Style(theme.AccentPrimary),
		Value: rt.Style(theme.TextPrimary),
	}
}
