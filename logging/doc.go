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

// Package logging installs a styled [*slog.Logger] via [nabat.WithExtension],
// with optional --verbose / --log-level flags. [ParseLevel] accepts the same
// level names. For a custom logger, use [nabat.WithLogger] instead.
//
//	app := nabat.MustNew("myctl",
//	    nabat.WithExtension(logging.New(
//	        logging.WithLevel(slog.LevelInfo),
//	        logging.WithVerboseFlag("verbose"),
//	    )),
//	)
package logging
