package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ludo-technologies/polyscan/polyscan/internal/pathmatch"
	ignore "github.com/sabhiram/go-gitignore"
)

// FileHelper provides file operation utilities
type FileHelper struct{}

// NewFileHelper creates a new FileHelper
func NewFileHelper() *FileHelper {
	return &FileHelper{}
}

// CollectJSFiles collects JavaScript/TypeScript files from paths
func (h *FileHelper) CollectJSFiles(paths []string, recursive bool, includePatterns, excludePatterns []string) ([]string, error) {
	var files []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			// A file named directly is only matched on its own name: the
			// directories it happens to live under are not part of the request.
			// Include patterns are skipped entirely here, since dropping a file
			// the user named explicitly would be the wrong answer.
			if h.isJSFile(path) && !h.isExcluded(filepath.Base(path), excludePatterns) {
				files = append(files, path)
			}
			continue
		}

		// Directory handling
		if recursive {
			// Load .gitignore from root directory
			gi := loadGitIgnore(path)

			err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Exclusions apply to the path relative to the analysis root, so
				// directories above the root never exclude the whole tree.
				relPath, relErr := filepath.Rel(path, filePath)
				if relErr != nil {
					relPath = filePath
				}

				if gi != nil && relErr == nil && gi.MatchesPath(relPath) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Skip excluded directories early
				if info.IsDir() {
					// The walk root itself is never excluded: the user asked for it.
					if relPath == "." {
						return nil
					}
					if h.isExcluded(relPath, excludePatterns) {
						return filepath.SkipDir
					}
					return nil
				}

				if h.isJSFile(filePath) && h.isIncluded(relPath, includePatterns) && !h.isExcluded(relPath, excludePatterns) {
					files = append(files, filePath)
				}

				return nil
			})
		} else {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, err
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					filePath := filepath.Join(path, entry.Name())
					if h.isJSFile(filePath) && h.isIncluded(entry.Name(), includePatterns) && !h.isExcluded(entry.Name(), excludePatterns) {
						files = append(files, filePath)
					}
				}
			}
		}

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// CollectPythonFiles is a compatibility wrapper for legacy domain.FileReader.
func (h *FileHelper) CollectPythonFiles(paths []string, recursive bool, includePatterns, excludePatterns []string) ([]string, error) {
	return h.CollectJSFiles(paths, recursive, includePatterns, excludePatterns)
}

// IsValidJSFile checks if a file is a valid JavaScript/TypeScript file
func (h *FileHelper) IsValidJSFile(path string) bool {
	return h.isJSFile(path)
}

// IsValidPythonFile is a compatibility wrapper for legacy domain.FileReader.
func (h *FileHelper) IsValidPythonFile(path string) bool {
	return h.IsValidJSFile(path)
}

// FileExists checks if a file exists
func (h *FileHelper) FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// ReadFile reads file content
func (h *FileHelper) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// isJSFile checks if a file is JavaScript/TypeScript based on extension
func (h *FileHelper) isJSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".js" || ext == ".ts" || ext == ".jsx" || ext == ".tsx" ||
		ext == ".mjs" || ext == ".cjs" || ext == ".mts" || ext == ".cts"
}

// isIncluded reports whether a path is selected by the include patterns.
//
// An empty pattern list selects everything, so a caller that does not care
// about include patterns can pass nil. Otherwise a path has to match at least
// one pattern, under the same rules exclude patterns use.
//
// Include patterns narrow the extensions jscan already understands; they cannot
// widen them, because a file it cannot parse is of no use to an analysis.
func (h *FileHelper) isIncluded(path string, includePatterns []string) bool {
	if len(includePatterns) == 0 {
		return true
	}
	return pathmatch.Matches(path, includePatterns)
}

// isExcluded checks if a path matches any exclude pattern.
func (h *FileHelper) isExcluded(path string, excludePatterns []string) bool {
	return pathmatch.Matches(path, excludePatterns)
}

// loadGitIgnore loads a .gitignore file from the root directory.
// Returns nil if the file does not exist or cannot be read.
func loadGitIgnore(root string) *ignore.GitIgnore {
	gitignorePath := filepath.Join(root, ".gitignore")
	gi, err := ignore.CompileIgnoreFile(gitignorePath)
	if err != nil {
		return nil
	}
	return gi
}

// ResolveFilePaths resolves file paths, returning existing files directly
// or collecting files from directories
func ResolveFilePaths(
	fileHelper *FileHelper,
	paths []string,
	recursive bool,
	includePatterns []string,
	excludePatterns []string,
) ([]string, error) {
	// Check if all paths are already files
	allFiles := true
	for _, path := range paths {
		exists, err := fileHelper.FileExists(path)
		if err != nil || !exists {
			allFiles = false
			break
		}
	}

	// If all paths are already files, no need to collect again
	if allFiles {
		return paths, nil
	}

	// Collect files from directories
	return fileHelper.CollectJSFiles(paths, recursive, includePatterns, excludePatterns)
}
