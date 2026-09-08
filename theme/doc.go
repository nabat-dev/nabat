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

// Package theme defines capability-aware theming primitives for Nabat
// and its extensions.
//
// A [Theme] carries one or more [Palette] entries, a default variant,
// and cross-variant defaults. [Theme.Resolve] picks a variant from
// [Capabilities] and returns an immutable [ResolvedTheme] queried by
// [Token] or accessor.
//
// The package does not depend on the nabat root package or IOStreams;
// extensions can import it to read styles without pulling in command
// or IO machinery.
package theme
