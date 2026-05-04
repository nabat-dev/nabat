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

// Styles holds lipgloss styles used by the logging extension to color
// level badges and key/value pairs in structured log output.
//
// Construct one from a [theme.ResolvedTheme] via [FromTheme] (the default),
// or build one by hand for custom handlers that want different visuals.
// Once built, Styles is immutable and safe to share across goroutines.
type Styles struct {
	// Debug styles the "DEBU" level badge for debug records.
	Debug lipgloss.Style

	// Info styles the "INFO" level badge for informational records.
	Info lipgloss.Style

	// Warn styles the "WARN" level badge for warning records.
	Warn lipgloss.Style

	// Error styles the "ERRO" level badge for error records.
	Error lipgloss.Style

	// Key styles structured-log attribute keys (the left side of `key=value`).
	Key lipgloss.Style

	// Value styles structured-log attribute values (the right side of `key=value`).
	Value lipgloss.Style
}

// FromTheme derives the logging extension's level-badge and key/value
// styles from a [theme.ResolvedTheme] using the well-known semantic
// tokens — [theme.StatusInfo], [theme.StatusWarning], [theme.StatusError]
// for level badges and [theme.AccentPrimary], [theme.TextPrimary] for the
// key=value pairs. Each badge is a fixed-width "DEBU" / "INFO" / "WARN" /
// "ERRO" label rendered bold.
//
// FromTheme keeps theme derivation a logging-package concern so the
// nabat root package does not have to know about logger concepts; the
// nabat/logging extension owns its own visual contract end to end.
//
// Callers that want a different shape (different badge text, different
// width, extra fields) can build a [Styles] value directly instead of
// going through FromTheme.
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
