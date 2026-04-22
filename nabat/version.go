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

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// buildInfo is the resolved version and build metadata rendered by the
// built-in version feature.
type buildInfo struct {
	version   string
	commit    string
	date      string
	dirty     bool
	goVersion string
	platform  string
}

// resolveBuildInfo combines explicit overrides from [WithVersion] with values
// read from [runtime/debug.ReadBuildInfo]. Explicit overrides win; missing
// commit/date fields are filled from VCS settings when available.
// dateTimeFormat is a Go time layout applied to the vcs.time value; empty
// means use the raw RFC3339 string. It has no effect on an explicit date set
// via [WithVersionCommitDate].
func resolveBuildInfo(version, commit, date, dateTimeFormat string) buildInfo {
	bi := buildInfo{
		version:   version,
		commit:    commit,
		date:      date,
		goVersion: runtime.Version(),
		platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if bi.commit == "" && len(s.Value) >= 7 {
					bi.commit = s.Value[:7]
				}
			case "vcs.time":
				if bi.date == "" {
					bi.date = formatVCSTime(s.Value, dateTimeFormat)
				}
			case "vcs.modified":
				bi.dirty = s.Value == "true"
			}
		}
	}
	return bi
}

// VersionFromBuildInfo returns the module version stamped by the Go toolchain
// when the binary was installed via go install (e.g. "v1.2.3"). Falls back to
// "(devel)" for local builds where no module version is available.
//
// Pass the result directly to [WithVersion]:
//
//	nabat.WithVersion(nabat.VersionFromBuildInfo())
func VersionFromBuildInfo() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "(devel)"
}

// formatVCSTime applies layout to the RFC3339 vcs.time value. Returns raw
// unchanged when layout is empty or the value cannot be parsed.
func formatVCSTime(raw, layout string) string {
	if layout == "" {
		return raw
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return t.Format(layout)
}

type versioner struct {
	name       string
	bi         buildInfo
	infoStyle  lipgloss.Style
	cachedLine string
}

func (v versioner) line() string {
	line := v.name + " " + v.bi.version
	var meta []string
	if v.bi.commit != "" {
		c := v.bi.commit
		if v.bi.dirty {
			c += "+dirty"
		}
		meta = append(meta, c)
	}
	if v.bi.date != "" {
		meta = append(meta, v.bi.date)
	}
	if len(meta) > 0 {
		line += " (" + strings.Join(meta, ", ") + ")"
	}
	return v.infoStyle.Render(line)
}

func (v versioner) print(w io.Writer, mode string) error {
	switch mode {
	case "json":
		view := buildInfoJSON{
			Version:   v.bi.version,
			Commit:    v.bi.commit,
			Date:      v.bi.date,
			Dirty:     v.bi.dirty,
			GoVersion: v.bi.goVersion,
			Platform:  v.bi.platform,
		}
		out, err := json.MarshalIndent(view, "", "  ")
		if err != nil {
			return fmt.Errorf("nabat: marshal version: %w", err)
		}
		_, err = fmt.Fprintln(w, string(out))
		return err
	case "short":
		_, err := fmt.Fprintln(w, v.bi.version)
		return err
	default:
		_, err := fmt.Fprintln(w, v.cachedLine)
		return err
	}
}

type buildInfoJSON struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	Date      string `json:"date,omitempty"`
	Dirty     bool   `json:"dirty,omitempty"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}
