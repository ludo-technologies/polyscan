package analyzer

import (
	"path/filepath"
	"strings"
)

// sourceExtensions lists the file extensions a specifier without one may resolve to.
var sourceExtensions = []string{".ts", ".tsx", ".js", ".jsx", ".mts", ".cts", ".mjs", ".cjs"}

// emittedExtensionSources maps the extension of an emitted JavaScript file to the
// TypeScript sources it is compiled from. Under Node16/NodeNext module resolution a
// TypeScript file is imported by the name of its emitted output, so "./foo.js" refers
// to foo.ts.
var emittedExtensionSources = map[string][]string{
	".js":  {".ts", ".tsx", ".d.ts"},
	".jsx": {".tsx"},
	".mjs": {".mts", ".d.mts"},
	".cjs": {".cts", ".d.cts"},
}

// moduleCandidates lists the slash-separated paths a relative specifier resolved to
// base may denote, in priority order: the path itself, its TypeScript sources when the path names an
// emitted JavaScript file, the path with a source extension appended, then an index
// file inside the path taken as a directory.
func moduleCandidates(base string) []string {
	candidates := []string{base}
	if sources, ok := emittedExtensionSources[filepath.Ext(base)]; ok {
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		for _, ext := range sources {
			candidates = append(candidates, stem+ext)
		}
	}
	for _, ext := range sourceExtensions {
		candidates = append(candidates, base+ext)
	}
	for _, ext := range sourceExtensions {
		candidates = append(candidates, base+"/index"+ext)
	}
	return candidates
}

// resolveModuleCandidate returns the first candidate for base that is a known module,
// or "" when none is.
func resolveModuleCandidate(base string, known map[string]bool) string {
	for _, candidate := range moduleCandidates(base) {
		if known[candidate] {
			return candidate
		}
	}
	return ""
}
