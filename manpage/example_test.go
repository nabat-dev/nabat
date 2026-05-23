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

package manpage_test

import (
	"context"
	"fmt"
	"strings"

	"nabat.dev/manpage"
	"nabat.dev/nabat"
	"nabat.dev/nabat/nabattest"
)

func ExampleNew() {
	io, _, stdout, _ := nabattest.NewIO()
	app := nabat.MustNew("myctl",
		nabat.WithIO(io),
		nabat.WithExtension(manpage.New()),
	)
	if err := app.RunArgs(context.Background(), "man"); err != nil {
		fmt.Println("error:", err)
		return
	}
	doc := strings.TrimSpace(stdout.String())
	if strings.HasPrefix(doc, ".TH") {
		fmt.Println("roff")
	} else {
		fmt.Println("unexpected format")
	}
	// Output:
	// roff
}
