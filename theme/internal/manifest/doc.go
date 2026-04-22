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

// Package manifest holds the parser machinery that turns a Nabat theme
// manifest (DTCG JSON, schema/v1.json) into a [theme.Theme] closure.
//
// It exists so the public [nabat.dev/theme] package surface stays
// minimal: only [Parse] is exported. Every other type or function in
// this package — the rawTheme intermediate, the styleResolver, the
// chroma / glamour / huh sub-parsers — is an implementation detail
// that can change between releases without breaking downstream callers.
//
// This package depends on [nabat.dev/theme] for the [theme.Theme] type
// it returns. Tests live alongside the implementation as
// `package manifest` so they can keep poking at internal types directly.
package manifest
