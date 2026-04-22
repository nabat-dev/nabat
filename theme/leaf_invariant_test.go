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
	"go/build"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// forbiddenLeafImports lists package paths that would compromise the
// leaf-no-IO invariant for [nabat.dev/theme]. The package's value
// proposition (extensions can import it without pulling in command-tree
// or stdin/stdout machinery) only holds as long as none of these
// dependencies sneak into the public package via an import.
//
// Subpackages of the theme tree itself (nabat.dev/theme, .../theme/...)
// are exempt — internal helpers cooperate, that's the whole point of
// the merger.
//
// Add an entry here whenever a new framework subpackage ships that the
// theme leaf must stay independent of.
var forbiddenLeafImports = []string{
	"nabat.dev/logging",
	"nabat.dev/manpage",
	"nabat.dev/nabattest",
}

// forbiddenLeafImportPrefix flags any import that pulls in the nabat
// root package itself (nabat.dev with no further path component). The
// check at the call site disambiguates this from nabat.dev/theme/...
// subpackages, which are allowed.
const forbiddenLeafImportPrefix = "nabat.dev/nabat"

// TestLeafHasNoFrameworkIODeps locks the [nabat.dev/theme] package's
// "no imports from nabat.dev" guarantee. The check walks the package's
// own import list (not transitive — Go's standard [build.Import] already
// expands cgo and conditional builds, but stops at one level) and
// fails the build the moment a forbidden dep appears at the top.
//
// The internal/manifest subpackage is checked separately by extension
// because it is allowed to depend on third-party libraries and pure
// stdlib, but must not depend on the framework either — the cycle
// break in P0 relies on it.
func TestLeafHasNoFrameworkIODeps(t *testing.T) {
	t.Parallel()

	for _, pkgPath := range []string{"nabat.dev/theme", "nabat.dev/theme/internal/manifest"} {
		t.Run(pkgPath, func(t *testing.T) {
			t.Parallel()

			pkg, err := build.Default.Import(pkgPath, "", 0)
			require.NoErrorf(t, err, "import %q", pkgPath)

			for _, imp := range pkg.Imports {
				// Subpackages of the theme tree itself are always fine.
				if imp == "nabat.dev/theme" || strings.HasPrefix(imp, "nabat.dev/theme/") {
					continue
				}
				// The bare nabat root package is forbidden everywhere here.
				require.NotEqualf(t, forbiddenLeafImportPrefix, imp,
					"%s imports the nabat root package; theme is supposed to be a leaf", pkgPath)
				for _, forbidden := range forbiddenLeafImports {
					require.Falsef(t, imp == forbidden || strings.HasPrefix(imp, forbidden+"/"),
						"%s imports forbidden framework package %q; the theme leaf must stay free of nabat-root deps", pkgPath, imp)
				}
			}
		})
	}
}
