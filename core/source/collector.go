package source

import (
	"os"
	"path/filepath"
	"sort"
)

// FileFilter configures which files to include or exclude during collection.
type FileFilter struct {
	IncludePatterns []string // Glob patterns to include (e.g. "*.py", "*.js")
	ExcludePatterns []string // Glob patterns to exclude (e.g. "*.pyc", "*_test.go")
	Recursive       bool     // Whether to recurse into subdirectories
}

// CollectFiles walks the given paths and returns matching files based on the filter.
// Each path can be a file or directory. Directories are walked according to filter settings.
func CollectFiles(paths []string, filter FileFilter) ([]string, error) {
	seen := make(map[string]bool)
	var result []string

	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			// Single file: check patterns
			if matchesFilter(filepath.Base(abs), filter) && !seen[abs] {
				seen[abs] = true
				result = append(result, abs)
			}
			continue
		}

		// Directory: walk
		if filter.Recursive {
			err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					return nil
				}
				if matchesFilter(d.Name(), filter) && !seen[path] {
					seen[path] = true
					result = append(result, path)
				}
				return nil
			})
		} else {
			entries, err2 := os.ReadDir(abs)
			if err2 != nil {
				return nil, err2
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				full := filepath.Join(abs, entry.Name())
				if matchesFilter(entry.Name(), filter) && !seen[full] {
					seen[full] = true
					result = append(result, full)
				}
			}
		}
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(result)
	return result, nil
}

// matchesFilter checks whether a filename passes the include/exclude patterns.
func matchesFilter(name string, filter FileFilter) bool {
	// If include patterns are specified, file must match at least one
	if len(filter.IncludePatterns) > 0 {
		if !MatchesAnyPattern(name, filter.IncludePatterns) {
			return false
		}
	}
	// If exclude patterns are specified, file must not match any
	if len(filter.ExcludePatterns) > 0 {
		if MatchesAnyPattern(name, filter.ExcludePatterns) {
			return false
		}
	}
	return true
}

// MatchesAnyPattern returns true if the name matches any of the given glob patterns.
func MatchesAnyPattern(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// IsDirectory returns true if the given path is a directory.
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
