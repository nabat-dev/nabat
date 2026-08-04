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

// WithPrompt attaches an interactive prompt to a declarative arg. T is inferred
// from the arg bind type. Pass "" for description to omit the subtitle.
// Kind-specific sub-options are constrained by T so misuse fails at compile time.
//
// Example:
//
//	WithArg("name", "", WithPrompt("Your name", "", WithDefault("anon")))
func WithPrompt[T any](title, description string, opts ...FieldOption[T]) ArgOption {
	return argOptionFn(func(s *argSpec) error {
		s.prompt.text = title
		s.prompt.description = description
		if err := applyFieldOptions("prompt", opts, &s.prompt); err != nil {
			return err
		}
		return nil
	})
}
