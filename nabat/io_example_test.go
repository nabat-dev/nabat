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

package nabat_test

import (
	"fmt"
	"io"
	"strings"

	"nabat.dev/nabat"
	"nabat.dev/nabattest"
)

func ExampleNewIO() {
	out := &strings.Builder{}
	s := nabat.NewIO(strings.NewReader("input"), out, &strings.Builder{})
	body, err := io.ReadAll(s.In)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body))
	// Output:
	// input
}

func ExampleNewIO_buffers() {
	in := &strings.Builder{}
	out := &strings.Builder{}
	errOut := &strings.Builder{}
	s := nabat.NewIO(strings.NewReader(""), out, errOut)
	if _, err := fmt.Fprint(s.Out, "primary\n"); err != nil {
		panic(err)
	}
	if _, err := fmt.Fprint(s.ErrOut, "diagnostics\n"); err != nil {
		panic(err)
	}
	_ = in
	fmt.Print(out.String())
	fmt.Print(errOut.String())
	// Output:
	// primary
	// diagnostics
}

func ExampleNewIO_testBuffers() {
	s, in, out, errOut := nabattest.NewIO()
	if _, err := in.WriteString("typed\n"); err != nil {
		panic(err)
	}
	if _, err := fmt.Fprint(s.Out, "stdout\n"); err != nil {
		panic(err)
	}
	if _, err := fmt.Fprint(s.ErrOut, "stderr\n"); err != nil {
		panic(err)
	}
	fmt.Print(in.String())
	fmt.Print(out.String())
	fmt.Print(errOut.String())
	// Output:
	// typed
	// stdout
	// stderr
}
