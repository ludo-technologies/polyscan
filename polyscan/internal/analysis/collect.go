package analysis

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ludo-technologies/polyscan/polyscan/internal/lang"
	"github.com/ludo-technologies/polyscan/polyscan/internal/pathmatch"
)

// skippedDirs are the directories a walk never descends into: version
// control, dependencies and build output are not the code under review.
// A path given on the command line is walked whatever its name.
var skippedDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"target":       true,
	"build":        true,
	"dist":         true,
	"third_party":  true,
}

// skipDir reports whether a directory met while walking is left out: the
// named build and dependency directories, and every hidden one such as
// .git or .cache.
func skipDir(name string) bool {
	return skippedDirs[name] || strings.HasPrefix(name, ".")
}

// collectFiles returns, sorted and without duplicates, the absolute paths
// of the files of a supported language under paths. A path that is a file
// is taken as given when its language is supported, unless its name matches
// an exclude pattern. Under a directory, exclude patterns apply to the path
// relative to that directory, so a directory above the root cannot exclude
// the whole tree; a directory that matches is not walked. Unless
// includeTests is set, the files a language names as test files are left
// out under the same relative-path rule.
func collectFiles(paths []string, exclude []string, includeTests bool) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	add := func(path, rel string) {
		language, ok := lang.ByPath(path)
		if !ok || seen[path] || pathmatch.Matches(rel, exclude) {
			return
		}
		if !includeTests && language.IsTestFile(rel) {
			return
		}
		seen[path] = true
		files = append(files, path)
	}
	for _, p := range paths {
		root, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			add(root, filepath.Base(root))
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == root {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if d.IsDir() {
				if skipDir(d.Name()) || pathmatch.Matches(rel, exclude) {
					return fs.SkipDir
				}
				return nil
			}
			add(path, rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}
