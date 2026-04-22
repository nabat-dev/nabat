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

package theme_test

import (
	"fmt"

	"nabat.dev/theme"
)

// ExampleGet shows the typical lookup path: resolve a built-in theme
// by name, then call [Theme.Resolve] against a Capabilities snapshot
// to read the resulting ResolvedTheme. Most consumers reach this
// through nabat.WithTheme rather than calling Get directly.
func ExampleGet() {
	t, err := theme.Get(theme.Default)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	r := t.Resolve(theme.Capabilities{Dark: true, Interactive: true})
	fmt.Println(r.Name())
	// Output: default
}

// ExampleGet_unknown shows the actionable error returned when a theme
// name does not exist in the catalog. The error lists every available
// name so users can fix typos without consulting docs.
func ExampleGet_unknown() {
	_, err := theme.Get("draculaa")
	fmt.Println(err != nil)
	// Output: true
}

// ExampleNames demonstrates that [Names] returns themes in lexical order,
// which is what callers typically want for completion lists.
func ExampleNames() {
	names := theme.Names()
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			fmt.Println("unsorted")
			return
		}
	}
	fmt.Println("sorted")
	// Output: sorted
}
