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

import "github.com/spf13/cobra"

// CompletionDirective signals to the shell how to handle a [CompletionFunc]
// result. The constants mirror Cobra's directives so behavior is identical;
// the alias type lets the public option signatures stay free of cobra imports.
type CompletionDirective = cobra.ShellCompDirective

const (
	// CompletionDefault delegates to the shell's default completion.
	CompletionDefault CompletionDirective = cobra.ShellCompDirectiveDefault
	// CompletionError indicates an error while computing completions.
	CompletionError CompletionDirective = cobra.ShellCompDirectiveError
	// CompletionNoSpace prevents the shell from appending a trailing space.
	CompletionNoSpace CompletionDirective = cobra.ShellCompDirectiveNoSpace
	// CompletionNoFileComp suppresses file completion when no candidates match.
	CompletionNoFileComp CompletionDirective = cobra.ShellCompDirectiveNoFileComp
	// CompletionFilterFileExt restricts file completion to the listed extensions.
	CompletionFilterFileExt CompletionDirective = cobra.ShellCompDirectiveFilterFileExt
	// CompletionFilterDirs restricts file completion to directories.
	CompletionFilterDirs CompletionDirective = cobra.ShellCompDirectiveFilterDirs
	// CompletionKeepOrder preserves the order in which completions are returned.
	CompletionKeepOrder CompletionDirective = cobra.ShellCompDirectiveKeepOrder
)

// CompletionFunc returns shell completion candidates for an arg or flag value.
// args contains the command-line tokens before the cursor (excluding the partial
// token); toComplete is the partial token under the cursor.
type CompletionFunc func(args []string, toComplete string) ([]string, CompletionDirective)

// completionFunc is the internal cobra-typed shape passed to pflag/cobra.
type completionFunc = cobra.CompletionFunc

// adaptCompletion wraps a [CompletionFunc] in the cobra-typed shape Cobra
// expects. Returns nil when fn is nil so callers can pass it directly to
// pflag/cobra without checks.
func adaptCompletion(fn CompletionFunc) completionFunc {
	if fn == nil {
		return nil
	}
	return func(_ *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		out, dir := fn(args, toComplete)
		comps := make([]cobra.Completion, 0, len(out))
		for _, s := range out {
			comps = append(comps, cobra.Completion(s))
		}
		return comps, dir
	}
}
