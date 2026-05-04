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

// Package logging provides a Nabat extension that installs a styled
// [*slog.Logger] into the App, plus optional --verbose and
// --log-level flag wiring. Use [ParseLevel] to parse the same level names
// (debug, info, warn, error) elsewhere in your CLI.
//
//	import (
//	    "log/slog"
//	    "nabat.dev/nabat"
//	    "nabat.dev/logging"
//	)
//
//	app := nabat.MustNew("myctl",
//	    nabat.WithExtension(logging.New(
//	        logging.WithLevel(slog.LevelInfo),
//	        logging.WithVerboseFlag("verbose"),
//	    )),
//	)
//
// To bring your own logger instead, use [nabat.WithLogger] at construction
// time and skip this extension.
package logging
