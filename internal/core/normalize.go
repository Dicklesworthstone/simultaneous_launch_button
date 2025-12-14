// Package core provides command normalization for pattern matching.
package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
)

// NormalizedCommand represents a parsed and normalized command.
type NormalizedCommand struct {
	// Original is the original command string.
	Original string
	// Primary is the primary command after stripping wrappers.
	Primary string
	// Segments contains individual command segments for compound commands.
	Segments []string
	// IsCompound indicates if this is a compound command.
	IsCompound bool
	// HasSubshell indicates if the command contains subshells.
	HasSubshell bool
	// StrippedWrappers lists the wrappers that were stripped.
	StrippedWrappers []string
	// ParseError indicates if parsing failed (triggers tier upgrade).
	ParseError bool
}

// Command wrapper prefixes to strip
var wrapperPrefixes = []string{
	"sudo",
	"doas",
	"env",
	"command",
	"builtin",
	"time",
	"nice",
	"ionice",
	"nohup",
	"strace",
	"ltrace",
}

// Compound command separators
var compoundSeparators = regexp.MustCompile(`\s*(?:;|&&|\|\||&)\s*`)

// Pipe detection
var pipePattern = regexp.MustCompile(`\s*\|\s*`)

// Subshell patterns: $(...) or `...` or (...)
var subshellPattern = regexp.MustCompile("\\$\\([^)]+\\)|`[^`]+`|\\([^)]+\\)")

var envAssignPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// NormalizeCommand parses and normalizes a command for pattern matching.
func NormalizeCommand(cmd string) *NormalizedCommand {
	result := &NormalizedCommand{
		Original:   cmd,
		Segments:   []string{},
		ParseError: false,
	}

	// Trim whitespace
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return result
	}

	// Check for subshells
	result.HasSubshell = subshellPattern.MatchString(cmd)

	// Split on compound separators
	segments := compoundSeparators.Split(cmd, -1)
	if len(segments) > 1 {
		// If separators are inside quoted SQL strings (e.g., psql -c "DELETE ...;")
		// keep as single segment to preserve context.
		if strings.Count(cmd, "\"") >= 2 {
			segments = []string{cmd}
			result.IsCompound = false
		} else {
			result.IsCompound = true
		}
	}

	// Also check for pipes (not technically compound, but multiple commands)
	for _, seg := range segments {
		if pipePattern.MatchString(seg) {
			result.IsCompound = true
			// Split on pipes and add each segment
			pipeParts := pipePattern.Split(seg, -1)
			for _, part := range pipeParts {
				part = strings.TrimSpace(part)
				if part != "" {
					result.Segments = append(result.Segments, part)
				}
			}
		} else {
			seg = strings.TrimSpace(seg)
			if seg != "" {
				result.Segments = append(result.Segments, seg)
			}
		}
	}

	// Normalize each segment (strip wrappers with shell-aware parsing)
	normalizedSegments := make([]string, 0, len(result.Segments))
	for _, seg := range result.Segments {
		normalized, wrappers, parseErr := normalizeSegment(seg)
		if parseErr {
			result.ParseError = true
		}
		if normalized != "" {
			normalizedSegments = append(normalizedSegments, normalized)
		}
		result.StrippedWrappers = append(result.StrippedWrappers, wrappers...)
	}
	result.Segments = normalizedSegments

	// Primary command is the first segment after normalization
	if len(result.Segments) > 0 {
		result.Primary = result.Segments[0]
	}

	return result
}

// normalizeSegment strips wrappers using a shell-aware tokenizer.
func normalizeSegment(seg string) (string, []string, bool) {
	parser := shellwords.NewParser()
	tokens, err := parser.Parse(seg)
	parseErr := err != nil
	if parseErr {
		// Fallback to simple split to avoid losing data
		tokens = strings.Fields(seg)
	}

	stripped := []string{}

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		// env with assignments
		if tok == "env" {
			stripped = append(stripped, "env")
			i++
			for i < len(tokens) && isEnvAssignment(tokens[i]) {
				i++
			}
			continue
		}

		if isWrapper(tok) {
			stripped = append(stripped, tok)
			i++
			continue
		}
		break
	}

	if i >= len(tokens) {
		return "", stripped, parseErr
	}

	normalized := strings.TrimSpace(strings.Join(tokens[i:], " "))
	return normalized, stripped, parseErr
}

func isWrapper(tok string) bool {
	for _, w := range wrapperPrefixes {
		if tok == w {
			return true
		}
	}
	return false
}

func isEnvAssignment(tok string) bool {
	return envAssignPattern.MatchString(tok)
}

// ResolvePathsInCommand expands relative paths to absolute paths.
func ResolvePathsInCommand(cmd, cwd string) string {
	// Expand ~ to home directory even when cwd is empty.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		tildePattern := regexp.MustCompile(`(^|\s)~(/[^\s]*)?`)
		cmd = tildePattern.ReplaceAllStringFunc(cmd, func(match string) string {
			prefix := ""
			if len(match) > 0 && (match[0] == ' ' || match[0] == '\t') {
				prefix = match[:1]
				match = match[1:]
			}

			suffix := strings.TrimPrefix(match, "~")
			suffix = strings.TrimPrefix(suffix, "/")
			suffix = strings.TrimPrefix(suffix, "\\")

			resolved := home
			if suffix != "" {
				resolved = filepath.Join(home, suffix)
			}

			return prefix + resolved
		})
	}

	if cwd == "" {
		return cmd
	}

	// Simple path resolution - replace ./ and ../ patterns
	// More sophisticated parsing could be done with shell tokenization

	// Replace ./ at word boundaries
	dotSlashPattern := regexp.MustCompile(`(^|\s)\.(/[^\s]*)`)
	cmd = dotSlashPattern.ReplaceAllStringFunc(cmd, func(match string) string {
		prefix := ""
		if len(match) > 0 && (match[0] == ' ' || match[0] == '\t') {
			prefix = string(match[0])
			match = match[1:]
		}
		resolved := filepath.Join(cwd, match)
		return prefix + resolved
	})

	// Replace ../ patterns
	dotDotPattern := regexp.MustCompile(`(^|\s)\.\.(/[^\s]*)`)
	cmd = dotDotPattern.ReplaceAllStringFunc(cmd, func(match string) string {
		prefix := ""
		if len(match) > 0 && (match[0] == ' ' || match[0] == '\t') {
			prefix = string(match[0])
			match = match[1:]
		}
		resolved := filepath.Clean(filepath.Join(cwd, match))
		return prefix + resolved
	})

	return cmd
}

// ExtractCommandName extracts just the command name (first word).
func ExtractCommandName(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	// Return just the base command name, without path
	return filepath.Base(fields[0])
}
