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

// collectedFile is one source file found under a CLI path.
type collectedFile struct {
	// path is the file as the caller named it: the argument joined with the
	// walk-relative remainder. Reports use this spelling so Go and JS agree
	// when the target is outside cwd.
	path string
	// abs is the absolute path used to read the file. Dedup keys on abs.
	abs string
	// rel is the path relative to that walk root. Test-file gating uses rel
	// so a root whose parent is named tests/ does not empty Rust/C++ results.
	rel string
}

// collectFiles returns, sorted and without duplicates, the files of a
// supported language under paths. A path that is a file is taken as given
// when its language is supported, unless its name matches an exclude
// pattern. Under a directory, exclude patterns apply to the path relative
// to that directory, so a directory above the root cannot exclude the whole
// tree; a directory that matches is not walked. Unless includeTests is set,
// the files a language names as test files are left out under the same
// relative-path rule. Walk and stat use the absolute path; the stored path
// keeps the caller's spelling.
func collectFiles(paths []string, exclude []string, includeTests bool) ([]collectedFile, error) {
	seen := map[string]bool{}
	var files []collectedFile
	add := func(stored, abs, rel string) {
		language, ok := lang.ByPath(abs)
		if !ok || seen[abs] || pathmatch.Matches(rel, exclude) {
			return
		}
		if !includeTests && language.IsTestFile(rel) {
			return
		}
		seen[abs] = true
		files = append(files, collectedFile{path: stored, abs: abs, rel: rel})
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
			add(p, root, filepath.Base(p))
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
			add(filepath.Join(p, rel), path, rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, nil
}

func collectPaths(files []collectedFile) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.path
	}
	return out
}
