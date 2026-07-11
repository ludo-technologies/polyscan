package source

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary directory structure:
//
//	testdir/
//	├── a.go
//	├── b.py
//	├── c.txt
//	└── sub/
//	    ├── d.go
//	    └── e.py
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, name := range []string{"a.go", "b.py", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("// "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"d.go", "e.py"} {
		if err := os.WriteFile(filepath.Join(sub, name), []byte("// "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

// ---------- MatchesAnyPattern ----------

func TestMatchesAnyPattern_Match(t *testing.T) {
	if !MatchesAnyPattern("foo.go", []string{"*.go", "*.py"}) {
		t.Error("expected match for foo.go against *.go")
	}
}

func TestMatchesAnyPattern_NoMatch(t *testing.T) {
	if MatchesAnyPattern("foo.txt", []string{"*.go", "*.py"}) {
		t.Error("did not expect match for foo.txt")
	}
}

func TestMatchesAnyPattern_EmptyPatterns(t *testing.T) {
	if MatchesAnyPattern("foo.go", nil) {
		t.Error("empty patterns should never match")
	}
}

// ---------- IsDirectory ----------

func TestIsDirectory_Dir(t *testing.T) {
	dir := t.TempDir()
	if !IsDirectory(dir) {
		t.Error("expected true for a directory")
	}
}

func TestIsDirectory_File(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsDirectory(f) {
		t.Error("expected false for a regular file")
	}
}

func TestIsDirectory_Nonexistent(t *testing.T) {
	if IsDirectory("/nonexistent/path/abc123") {
		t.Error("expected false for nonexistent path")
	}
}

// ---------- CollectFiles ----------

func TestCollectFiles_SingleFile(t *testing.T) {
	dir := setupTestDir(t)
	file := filepath.Join(dir, "a.go")

	files, err := CollectFiles([]string{file}, FileFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0] != file {
		t.Errorf("expected %s, got %s", file, files[0])
	}
}

func TestCollectFiles_DirectoryNonRecursive(t *testing.T) {
	dir := setupTestDir(t)

	files, err := CollectFiles([]string{dir}, FileFilter{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	// Should only contain a.go, b.py, c.txt (not sub/ contents)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Dir(f) != dir {
			t.Errorf("file %s should be directly in %s", f, dir)
		}
	}
}

func TestCollectFiles_DirectoryRecursive(t *testing.T) {
	dir := setupTestDir(t)

	files, err := CollectFiles([]string{dir}, FileFilter{Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	// Should contain all 5 files
	if len(files) != 5 {
		t.Fatalf("expected 5 files, got %d: %v", len(files), files)
	}
}

func TestCollectFiles_IncludePatterns(t *testing.T) {
	dir := setupTestDir(t)

	files, err := CollectFiles([]string{dir}, FileFilter{
		IncludePatterns: []string{"*.go"},
		Recursive:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// a.go and sub/d.go
	if len(files) != 2 {
		t.Fatalf("expected 2 .go files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			t.Errorf("unexpected non-.go file: %s", f)
		}
	}
}

func TestCollectFiles_ExcludePatterns(t *testing.T) {
	dir := setupTestDir(t)

	files, err := CollectFiles([]string{dir}, FileFilter{
		ExcludePatterns: []string{"*.txt"},
		Recursive:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// All except c.txt → 4 files
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Ext(f) == ".txt" {
			t.Errorf("should have excluded .txt file: %s", f)
		}
	}
}

func TestCollectFiles_IncludeAndExclude(t *testing.T) {
	dir := setupTestDir(t)

	files, err := CollectFiles([]string{dir}, FileFilter{
		IncludePatterns: []string{"*.go", "*.py"},
		ExcludePatterns: []string{"*.py"},
		Recursive:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Include *.go and *.py, then exclude *.py → only *.go files: a.go, sub/d.go
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if filepath.Ext(f) != ".go" {
			t.Errorf("unexpected file: %s", f)
		}
	}
}

func TestCollectFiles_EmptyPaths(t *testing.T) {
	files, err := CollectFiles(nil, FileFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}

func TestCollectFiles_Deduplication(t *testing.T) {
	dir := setupTestDir(t)
	file := filepath.Join(dir, "a.go")

	// Pass the same file twice, and also pass the directory containing it
	files, err := CollectFiles([]string{file, file, dir}, FileFilter{Recursive: false})
	if err != nil {
		t.Fatal(err)
	}

	// Count how many times a.go appears
	count := 0
	for _, f := range files {
		if filepath.Base(f) == "a.go" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected a.go exactly once, got %d times in %v", count, files)
	}
}
