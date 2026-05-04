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
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"nabat.dev/theme"
)

func (a *App) renderHelp(cmd *cobra.Command, _ []string) {
	out := &writer{w: a.io.Out}
	rt := a.Theme()
	body := rt.Style(theme.TextSecondary)
	section := rt.Style(theme.AccentPrimary)
	muted := rt.Style(theme.TextMuted)
	warn := rt.Style(theme.StatusWarning)
	termWidth := a.io.TerminalWidth()

	// Title + description.
	out.println(rt.Style(theme.TextTitle).Render(cmd.CommandPath()))
	if msg := strings.TrimSpace(cmd.Deprecated); msg != "" {
		out.println(body.Render("Deprecated: " + msg))
	}
	if cmd.Long != "" {
		out.println(body.Render(cmd.Long))
	} else if cmd.Short != "" {
		out.println(body.Render(cmd.Short))
	}

	// Aliases.
	if len(cmd.Aliases) > 0 {
		out.println(muted.Render("Aliases: " + strings.Join(cmd.Aliases, ", ")))
	}

	// Usage.
	out.println()
	out.println(section.Render("Usage:"))
	out.printf("  %s\n", a.styleUsageLine(cmd.UseLine()))

	meta := a.meta[cmd]

	// flagDefByName looks up the Nabat flag definition for a given flag name,
	// walking up the command tree so inherited (persistent) flags are found.
	flagDefByName := func(name string) *flagDef {
		for c := cmd; c != nil; c = c.Parent() {
			if m := a.meta[c]; m != nil {
				for i := range m.flags {
					if m.flags[i].name == name {
						return &m.flags[i]
					}
				}
			}
		}
		return nil
	}

	// Arguments.
	hasPassthroughDesc := meta != nil && meta.passthrough != nil && meta.passthrough.desc != ""
	if meta != nil && (len(meta.args) > 0 || hasPassthroughDesc) {
		out.println()
		out.println(section.Render("Arguments:"))
		for _, in := range meta.args {
			lhsPlain := in.name
			lhsStyled := section.Render(in.name)
			if hint := in.valueType.typeHint(); hint != "" {
				lhsPlain += " [" + hint + "]"
				lhsStyled += " " + muted.Render("["+hint+"]")
			}

			var segs []descSegment
			if in.config.usage != "" {
				for w := range strings.FieldsSeq(in.config.usage) {
					segs = append(segs, descSegment{w, w})
				}
			}
			if frag := in.config.envUsageFragment(a.cfg.envPrefix); frag != "" {
				segs = append(segs, descSegment{frag, muted.Render(frag)})
			}
			if in.config.required {
				segs = append(segs, descSegment{"(required)", section.Render("(required)")})
			}
			if in.config.hasDefault && !isZeroDefault(in.config.defaultValue) {
				s := fmt.Sprintf("(default: %v)", in.config.defaultValue)
				segs = append(segs, descSegment{s, muted.Render(s)})
			}

			descIndent := 2 + len(lhsPlain) + 2
			wrapped := wrapSegments(segs, termWidth-descIndent, descIndent)
			if wrapped != "" {
				out.printf("  %s  %s\n", lhsStyled, wrapped)
			} else {
				out.printf("  %s\n", lhsStyled)
			}
		}
		if hasPassthroughDesc {
			out.printf("  %s  %s\n", section.Render("--"), meta.passthrough.desc)
		}
	}

	// buildFlagDesc returns description segments for a single flag.
	buildFlagDesc := func(f *pflag.Flag, fd *flagDef, includeDefault bool) []descSegment {
		var segs []descSegment
		base := strings.TrimSpace(f.Usage)
		// The env fragment "(env: ...)" is baked into f.Usage at registration.
		if envIdx := strings.Index(base, "(env: "); envIdx >= 0 {
			for w := range strings.FieldsSeq(strings.TrimSpace(base[:envIdx])) {
				segs = append(segs, descSegment{w, w})
			}
			envPart := base[envIdx:]
			segs = append(segs, descSegment{envPart, muted.Render(envPart)})
		} else {
			for w := range strings.FieldsSeq(base) {
				segs = append(segs, descSegment{w, w})
			}
		}
		if fd != nil && fd.config.required {
			segs = append(segs, descSegment{"(required)", section.Render("(required)")})
		}
		if includeDefault && isNonZeroDefault(f.DefValue) {
			s := "(default: " + f.DefValue + ")"
			segs = append(segs, descSegment{s, muted.Render(s)})
		}
		if msg := strings.TrimSpace(f.Deprecated); msg != "" {
			s := "(deprecated: " + msg + ")"
			segs = append(segs, descSegment{s, warn.Render(s)})
		}
		if msg := strings.TrimSpace(f.ShorthandDeprecated); msg != "" {
			s := "(shorthand deprecated: " + msg + ")"
			segs = append(segs, descSegment{s, warn.Render(s)})
		}
		return segs
	}

	// renderFlagSet renders a set of flags with type hints, alignment, and wrapping.
	renderFlagSet := func(flags *pflag.FlagSet, includeDefault bool) {
		type flagInfo struct {
			plain  string // for width measurement
			styled string // for output
		}
		var infos []flagInfo
		maxLen := 0
		flags.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			name := fmt.Sprintf("--%s", f.Name)
			if f.Shorthand != "" {
				name += fmt.Sprintf(", -%s", f.Shorthand)
			}
			plain, styled := name, section.Render(name)
			if fd := flagDefByName(f.Name); fd != nil {
				hint := fd.valueType.typeHint()
				if hint != "" && hint != "bool" && hint != "count" {
					plain += " <" + hint + ">"
					styled += " " + muted.Render("<"+hint+">")
				}
			}
			infos = append(infos, flagInfo{plain, styled})
			if len(plain) > maxLen {
				maxLen = len(plain)
			}
		})

		i := 0
		flags.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			info := infos[i]
			i++
			fd := flagDefByName(f.Name)
			segs := buildFlagDesc(f, fd, includeDefault)
			pad := maxLen - len(info.plain)
			descIndent := 2 + maxLen + 2
			wrapped := wrapSegments(segs, termWidth-descIndent, descIndent)
			if wrapped != "" {
				out.printf("  %s%s  %s\n", info.styled, strings.Repeat(" ", pad), wrapped)
			} else {
				out.printf("  %s\n", info.styled)
			}
		})
	}

	if cmd.NonInheritedFlags().HasAvailableFlags() {
		out.println()
		out.println(section.Render("Flags:"))
		renderFlagSet(cmd.NonInheritedFlags(), true)
	}

	if cmd.HasAvailableInheritedFlags() {
		out.println()
		out.println(section.Render("Global Flags:"))
		renderFlagSet(cmd.InheritedFlags(), false)
	}

	// Commands.
	children := cmd.Commands()
	hasCommands := false
	if len(children) > 0 {
		groups := cmd.Groups()
		grouped := make(map[string][]*cobra.Command, len(groups))
		var ungrouped []*cobra.Command
		for _, child := range children {
			if child.Hidden {
				continue
			}
			hasCommands = true
			if child.GroupID != "" {
				grouped[child.GroupID] = append(grouped[child.GroupID], child)
			} else {
				ungrouped = append(ungrouped, child)
			}
		}

		renderCmdList := func(cmds []*cobra.Command) {
			maxLen := 0
			for _, c := range cmds {
				if n := len(c.Name()); n > maxLen {
					maxLen = n
				}
			}
			for _, c := range cmds {
				styledName := section.Render(c.Name())
				pad := maxLen - len(c.Name())
				if c.Short != "" {
					out.printf("  %s%s  %s\n", styledName, strings.Repeat(" ", pad), c.Short)
				} else {
					out.printf("  %s\n", styledName)
				}
			}
		}

		for _, g := range groups {
			cmds := grouped[g.ID]
			if len(cmds) == 0 {
				continue
			}
			out.println()
			out.println(section.Render(g.Title + ":"))
			renderCmdList(cmds)
		}

		if len(ungrouped) > 0 {
			out.println()
			out.println(section.Render("Commands:"))
			renderCmdList(ungrouped)
		}
	}

	// Examples.
	example := cmd.Example
	if meta != nil && meta.example != "" {
		example = meta.example
	}
	if example != "" {
		out.println()
		out.println(section.Render("Examples:"))
		styled := strings.TrimRight(a.styleShellExample(example), "\n")
		for line := range strings.SplitSeq(styled, "\n") {
			out.printf("  %s\n", line)
		}
	}

	// Footer hint for subcommand discovery.
	if hasCommands {
		out.println()
		out.printf("  %s\n", muted.Render(
			fmt.Sprintf("Use '%s <command> --help' for more information about a command.", cmd.CommandPath()),
		))
	}
}

// styleUsageLine applies per-token styling to a Cobra UseLine string.
// Words that form the command path are styled TextTitle, [flags] and [command]
// TextMuted, <required-arg> AccentPrimary, and [optional-arg] TextSecondary.
// Unknown-shaped tokens fall through unstyled.
func (a *App) styleUsageLine(line string) string {
	rt := a.Theme()
	titleStyle := rt.Style(theme.TextTitle)
	mutedStyle := rt.Style(theme.TextMuted)
	accentStyle := rt.Style(theme.AccentPrimary)
	secondStyle := rt.Style(theme.TextSecondary)

	parts := strings.Fields(line)
	out := make([]string, 0, len(parts))
	onCommandPath := true
	for _, p := range parts {
		switch {
		case onCommandPath && !strings.ContainsAny(p, "<>["):
			out = append(out, titleStyle.Render(p))
		case p == "[flags]" || p == "[command]":
			onCommandPath = false
			out = append(out, mutedStyle.Render(p))
		case strings.HasPrefix(p, "<") && strings.HasSuffix(p, ">"):
			onCommandPath = false
			out = append(out, accentStyle.Render(p))
		case strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]"):
			onCommandPath = false
			out = append(out, secondStyle.Render(p))
		default:
			onCommandPath = false
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// styleShellExample applies shell-aware syntax coloring to an example block.
// Comment lines (# ...) are styled TextMuted. On each command line the first
// bare word is TextTitle (program name); flags (--flag, -f) are AccentPrimary;
// quoted strings are TextSecondary; shell operators (|, &&, >, \) are TextMuted;
// all other words are TextPrimary. Indentation is preserved. A trailing
// backslash marks a line continuation: the first token of the following line is
// not treated as a program name.
func (a *App) styleShellExample(example string) string {
	rt := a.Theme()
	mutedStyle := rt.Style(theme.TextMuted)
	titleStyle := rt.Style(theme.TextTitle)
	accentStyle := rt.Style(theme.AccentPrimary)
	secondStyle := rt.Style(theme.TextSecondary)
	primaryStyle := rt.Style(theme.TextPrimary)

	tokenizeLine := func(line string, isContinuation bool) string {
		var sb strings.Builder
		firstToken := !isContinuation
		pos := 0
		for pos < len(line) {
			ch := line[pos]
			// Whitespace: preserve as-is.
			if ch == ' ' || ch == '\t' {
				sb.WriteByte(ch)
				pos++
				continue
			}
			// Inline comment: rest of line is muted.
			if ch == '#' {
				sb.WriteString(mutedStyle.Render(line[pos:]))
				break
			}
			// Quoted string.
			if ch == '\'' || ch == '"' {
				quote := ch
				end := pos + 1
				for end < len(line) && line[end] != quote {
					if line[end] == '\\' {
						end++ // skip escaped char
					}
					end++
				}
				if end < len(line) {
					end++ // include closing quote
				}
				sb.WriteString(secondStyle.Render(line[pos:end]))
				pos = end
				continue
			}
			// Flag: --foo or -f.
			if ch == '-' && pos+1 < len(line) &&
				(line[pos+1] == '-' ||
					(line[pos+1] >= 'a' && line[pos+1] <= 'z') ||
					(line[pos+1] >= 'A' && line[pos+1] <= 'Z')) {
				end := pos
				for end < len(line) && line[end] != ' ' && line[end] != '\t' {
					end++
				}
				sb.WriteString(accentStyle.Render(line[pos:end]))
				pos = end
				continue
			}
			// Line-continuation backslash at end of line.
			if ch == '\\' && pos == len(line)-1 {
				sb.WriteString(mutedStyle.Render(`\`))
				pos++
				continue
			}
			// 2> redirect.
			if ch == '2' && pos+1 < len(line) && line[pos+1] == '>' {
				sb.WriteString(mutedStyle.Render("2>"))
				pos += 2
				firstToken = true
				continue
			}
			// Pipe, redirect, semicolon — consume up to two chars (||, >>).
			if ch == '|' || ch == '>' || ch == ';' {
				end := pos + 1
				if end < len(line) && (line[end] == '|' || line[end] == '>') {
					end++
				}
				sb.WriteString(mutedStyle.Render(line[pos:end]))
				pos = end
				firstToken = true
				continue
			}
			// &&
			if ch == '&' && pos+1 < len(line) && line[pos+1] == '&' {
				sb.WriteString(mutedStyle.Render("&&"))
				pos += 2
				firstToken = true
				continue
			}
			// Regular word: advance to next whitespace.
			end := pos
			for end < len(line) && line[end] != ' ' && line[end] != '\t' {
				end++
			}
			word := line[pos:end]
			if firstToken {
				sb.WriteString(titleStyle.Render(word))
				firstToken = false
			} else {
				sb.WriteString(primaryStyle.Render(word))
			}
			pos = end
		}
		return sb.String()
	}

	lines := strings.Split(example, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		indent := line[:len(line)-len(trimmed)]
		if strings.HasPrefix(trimmed, "#") {
			out = append(out, indent+mutedStyle.Render(trimmed))
			continue
		}
		isContinuation := i > 0 && strings.HasSuffix(strings.TrimRight(lines[i-1], " \t"), `\`)
		out = append(out, indent+tokenizeLine(trimmed, isContinuation))
	}
	return strings.Join(out, "\n")
}

// descSegment is an atomic unit (word or parenthesized annotation) in a
// help description. plain is for width measurement; styled carries ANSI codes
// for terminal output.
type descSegment struct {
	plain  string
	styled string
}

// wrapSegments joins segments with spaces, breaking to a new indented line
// when the next segment would exceed width. Each segment's styled text is
// self-contained (properly opened and closed ANSI codes), so line breaks
// between segments never split an escape sequence.
func wrapSegments(segs []descSegment, width, indent int) string {
	if len(segs) == 0 {
		return ""
	}
	var out strings.Builder
	lineWidth := 0
	for _, seg := range segs {
		segWidth := len(seg.plain)
		if lineWidth > 0 && width > 0 && lineWidth+1+segWidth > width {
			out.WriteByte('\n')
			out.WriteString(strings.Repeat(" ", indent))
			lineWidth = 0
		}
		if lineWidth > 0 {
			out.WriteByte(' ')
			lineWidth++
		}
		out.WriteString(seg.styled)
		lineWidth += segWidth
	}
	return out.String()
}

// isNonZeroDefault reports whether a flag default value should be displayed.
// It suppresses the zero value for each supported type so help output stays
// concise.
func isNonZeroDefault(defValue string) bool {
	switch defValue {
	case "", "[]", "false", "0", "0.0", "0s":
		return false
	}
	return true
}

// isZeroDefault reports whether v is the zero value of one of the supported
// [ArgValue] kinds. Used to suppress noisy "(default: )", "(default: 0)",
// "(default: false)", and "(default: 0s)" lines in help output for positional
// args. The typed switch is intentionally tied to [ArgValue]: when a new kind
// is added there, this function should grow a matching case. Unknown types
// return false so a newly added [ArgValue] still renders by default until the
// helper is extended.
//
// String args with multiline or file-picker prompt modes are normalized to
// plain string by [normalizeDefaultValue] before reaching this function.
// Count flags ([WithCount]) are flag-only and rejected for args.
func isZeroDefault(v any) bool {
	switch x := v.(type) {
	case string:
		return x == ""
	case bool:
		return !x
	case int:
		return x == 0
	case int64:
		return x == 0
	case uint:
		return x == 0
	case float64:
		return x == 0
	case time.Duration:
		return x == 0
	case []string:
		return len(x) == 0
	default:
		return false
	}
}
