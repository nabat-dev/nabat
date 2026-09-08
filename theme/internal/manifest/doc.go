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

// Package manifest turns a Nabat theme manifest (DTCG JSON) into a
// [*Compiled] intermediate value.
//
// The [nabat.dev/theme] catalog assembles a Theme from [*Compiled]. This
// package does not import theme, so the dependency direction stays
// theme -> manifest.
package manifest
