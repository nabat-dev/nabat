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

// Package theme defines the open-ended, capability-aware theming primitives
// used by Nabat and its extensions.
//
// A [Theme] is a struct that carries one or more [Palette] entries (one per
// declared variant), a default variant selector, and cross-variant defaults
// (list enumerator, table border). [Theme.Resolve] picks a variant based on
// [Capabilities] and returns an immutable [ResolvedTheme] that consumers
// query by [Token] or by accessor.
//
// The package has no dependency on the nabat root package or on
// [nabat.IOStreams]. Extensions can import it directly to read styles from a
// resolved theme without pulling in command, IO, or option machinery.
//
// See the package-level docs and docs/themes.md for the full design.
package theme
