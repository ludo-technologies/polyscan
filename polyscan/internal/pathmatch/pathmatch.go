// Package pathmatch matches paths against the exclude and include patterns
// polyscan accepts on the command line and in jscan.config.json.
package pathmatch

import (
	"path/filepath"
	"strings"
)

// Matches reports whether a path matches at least one of the patterns.
//
// Patterns without a slash are matched against the file name and against each
// directory segment of the path, so "dist" matches "dist/bundle.js" but not
// "src/utils/distance.ts". Patterns with a slash are matched against the path
// as a whole, with "**" matching any number of segments.
//
// Matching ignores case, on both sides, so that the default include pattern
// "**/*.ts" selects Widget.TS exactly as the extension check accepts it.
// Treating the two differently would drop such a file with nothing said
// about it.
//
// Note: filepath.Match errors are ignored throughout (invalid patterns simply
// don't match) so that the remaining valid patterns still apply.
func Matches(path string, patterns []string) bool {
	segments := strings.Split(strings.ToLower(filepath.ToSlash(path)), "/")
	baseName := segments[len(segments)-1]
	dirSegments := segments[:len(segments)-1]

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.Trim(filepath.ToSlash(pattern), "/"))
		if pattern == "" {
			continue
		}

		if strings.Contains(pattern, "/") {
			if matchesPathPattern(pattern, segments) {
				return true
			}
			continue
		}

		if matched, err := filepath.Match(pattern, baseName); err == nil && matched {
			return true
		}
		for _, segment := range dirSegments {
			if segment == pattern {
				return true
			}
			if matched, err := filepath.Match(pattern, segment); err == nil && matched {
				return true
			}
		}
	}
	return false
}

// matchesPathPattern reports whether a multi-segment pattern such as
// "src/generated/**" matches the given path segments. The pattern may start at
// any segment boundary, so it behaves like a relative path fragment.
func matchesPathPattern(pattern string, segments []string) bool {
	patternSegments := strings.Split(pattern, "/")
	for i := range segments {
		if matchSegments(patternSegments, segments[i:]) {
			return true
		}
	}
	return false
}

// matchSegments matches pattern segments against a leading run of path
// segments, where "**" matches zero or more segments and every other segment is
// a glob. Matching a prefix is enough, so a pattern naming a directory such as
// "src/generated" also matches everything below it.
func matchSegments(pattern, segments []string) bool {
	if len(pattern) == 0 {
		return true
	}
	if pattern[0] == "**" {
		for i := 0; i <= len(segments); i++ {
			if matchSegments(pattern[1:], segments[i:]) {
				return true
			}
		}
		return false
	}
	if len(segments) == 0 {
		return false
	}
	if matched, err := filepath.Match(pattern[0], segments[0]); err != nil || !matched {
		return false
	}
	return matchSegments(pattern[1:], segments[1:])
}
