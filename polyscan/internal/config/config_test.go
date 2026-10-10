package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadWithoutFile(t *testing.T) {
	cfg, err := Load("", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("cfg = %+v, want the defaults", cfg)
	}
}

func TestLoadDiscoversUpward(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, FileName), "[analysis]\nexclude = [\"gen\"]\n\n[check]\nmax_cycles = 2\n")
	write(t, filepath.Join(root, "pkg", "src", "a.go"), "package a\n")

	for _, target := range []string{
		filepath.Join(root, "pkg", "src"),
		filepath.Join(root, "pkg", "src", "a.go"),
	} {
		cfg, err := Load("", target)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cfg.Analysis.Exclude, []string{"gen"}) || cfg.Check.MaxCycles != 2 {
			t.Errorf("%s: cfg = %+v, want the root file's settings", target, cfg)
		}
		// Keys the file leaves out keep their defaults.
		if cfg.Complexity != Default().Complexity || cfg.Check.MaxComplexity != Default().Check.MaxComplexity {
			t.Errorf("%s: cfg = %+v, want defaults for the keys left out", target, cfg)
		}
	}
}

func TestLoadNearestFileWins(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "jscan.config.json"), "{}")
	write(t, filepath.Join(root, "pkg", FileName), "[check]\nmax_complexity = 15\n")

	cfg, err := Load("", filepath.Join(root, "pkg"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Check.MaxComplexity != 15 {
		t.Errorf("max_complexity = %d, want 15", cfg.Check.MaxComplexity)
	}
}

func TestLoadRejectsJscanFile(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".jscan.toml"), "")
	write(t, filepath.Join(root, "pkg", ".pyscn.toml"), "")

	_, err := Load("", filepath.Join(root, "pkg"))
	if err == nil || !strings.Contains(err.Error(), ".jscan.toml is no longer read") {
		t.Fatalf("err = %v, want the jscan file named", err)
	}
}

func TestLoadIgnoresPyscnFile(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".pyscn.toml"), "[complexity]\nlow_threshold = 1\n")

	cfg, err := Load("", root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, Default()) {
		t.Errorf("cfg = %+v, want the defaults", cfg)
	}
}

func TestLoadRejectsJscanEnvVar(t *testing.T) {
	t.Setenv("JSCAN_CONFIG", "jscan.config.json")
	if _, err := Load("", t.TempDir()); err == nil || !strings.Contains(err.Error(), "JSCAN_CONFIG") {
		t.Fatalf("err = %v, want JSCAN_CONFIG named", err)
	}
}

func TestLoadExplicitPath(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, FileName), "[check]\nmax_complexity = 15\n")
	explicit := filepath.Join(t.TempDir(), "ci.toml")
	write(t, explicit, "[check]\nmax_complexity = 30\n")

	cfg, err := Load(explicit, root)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Check.MaxComplexity != 30 {
		t.Errorf("max_complexity = %d, want the explicit file's 30", cfg.Check.MaxComplexity)
	}

	if _, err := Load(filepath.Join(root, "missing.toml"), root); err == nil {
		t.Error("a missing explicit file loaded")
	}
}

func TestLoadRejectsInvalidFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		content string
		want    string
	}{
		"unknown key":      {"[analysis]\nexclude = []\nexlude = []\n", "unknown key(s): analysis.exlude"},
		"unknown section":  {"[output]\nmin_complexity = 2\n", "unknown key(s): output"},
		"wrong type":       {"[check]\nmax_cycles = \"2\"\n", FileName + ":2:14: toml: cannot decode TOML string"},
		"syntax error":     {"[check\n", FileName + ":1:7: toml: expected character ]"},
		"low threshold":    {"[complexity]\nlow_threshold = 0\n", "complexity.low_threshold must be at least 1"},
		"medium threshold": {"[complexity]\nlow_threshold = 20\n", "complexity.medium_threshold (19) must be greater"},
		"max complexity":   {"[check]\nmax_complexity = 0\n", "check.max_complexity must be at least 1"},
		"max cycles":       {"[check]\nmax_cycles = -1\n", "check.max_cycles must not be negative"},
		"exclude pattern":  {"[analysis]\nexclude = [\"gen\", \"src/[\"]\n", `analysis.exclude: invalid pattern "src/["`},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			write(t, filepath.Join(root, FileName), tc.content)
			_, err := Load("", root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}
