// Package config loads the polyscan configuration file, .polyscan.toml.
//
// One file configures every language. Load reads the file named by --config,
// or else the nearest .polyscan.toml found by walking up from the analyzed
// path to the filesystem root. Without either, the defaults apply.
//
// The file is strict: an unknown key and an invalid value both fail the run,
// so a setting is never accepted and then ignored. A configuration file that
// jscan read, such as jscan.config.json, also fails the run when discovery
// reaches it before any .polyscan.toml, because its settings would otherwise
// stop applying without notice.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ludo-technologies/polyscan/core/domain"
	"github.com/pelletier/go-toml/v2"
)

// FileName is the name discovery looks for.
const FileName = ".polyscan.toml"

// DocsURL documents the file.
const DocsURL = "https://docs.codescan.dev/polyscan/configuration/"

// legacyFileNames are the configuration files jscan read. pyscn's own names,
// such as .pyscn.toml, are not among them: in a repository that pyscn also
// analyzes, that file belongs to pyscn.
var legacyFileNames = []string{
	"jscan.config.json",
	".jscanrc.json",
	"jscan.yaml",
	"jscan.yml",
	".jscan.toml",
	".jscan.yml",
	"jscan.json",
	".jscan.json",
}

// legacyEnvVar named a configuration file for jscan.
const legacyEnvVar = "JSCAN_CONFIG"

// Config is the content of a .polyscan.toml file.
type Config struct {
	Analysis   Analysis   `toml:"analysis"`
	Complexity Complexity `toml:"complexity"`
	Check      Check      `toml:"check"`
}

// Analysis chooses the files every analysis covers.
type Analysis struct {
	// Exclude lists further files and directories to leave out, in the
	// syntax of --exclude. The patterns are added to the built-in skips.
	Exclude []string `toml:"exclude"`
	// IncludeTests keeps test files and test code in the analysis.
	IncludeTests bool `toml:"include_tests"`
}

// Complexity sets the risk bands of cyclomatic complexity for every
// language.
type Complexity struct {
	// LowThreshold is the highest complexity of the low risk band.
	LowThreshold int `toml:"low_threshold"`
	// MediumThreshold is the highest complexity of the medium risk band.
	MediumThreshold int `toml:"medium_threshold"`
}

// Check sets the thresholds of polyscan check. Each has the flag of the same
// name, which takes precedence.
type Check struct {
	MaxComplexity     int  `toml:"max_complexity"`
	MaxCycles         int  `toml:"max_cycles"`
	AllowDeadCode     bool `toml:"allow_dead_code"`
	AllowCircularDeps bool `toml:"allow_circular_deps"`
	AllowParseErrors  bool `toml:"allow_parse_errors"`
}

// Default returns the configuration that applies without a file.
func Default() *Config {
	return &Config{
		Complexity: Complexity{
			LowThreshold:    domain.DefaultComplexityLowThreshold,
			MediumThreshold: domain.DefaultComplexityMediumThreshold,
		},
		Check: Check{
			MaxComplexity: 10,
		},
	}
}

// Validate reports the first value out of range.
func (c *Config) Validate() error {
	if c.Complexity.LowThreshold < 1 {
		return fmt.Errorf("complexity.low_threshold must be at least 1, got %d", c.Complexity.LowThreshold)
	}
	if c.Complexity.MediumThreshold <= c.Complexity.LowThreshold {
		return fmt.Errorf("complexity.medium_threshold (%d) must be greater than complexity.low_threshold (%d)",
			c.Complexity.MediumThreshold, c.Complexity.LowThreshold)
	}
	if c.Check.MaxComplexity < 1 {
		return fmt.Errorf("check.max_complexity must be at least 1, got %d", c.Check.MaxComplexity)
	}
	if c.Check.MaxCycles < 0 {
		return fmt.Errorf("check.max_cycles must not be negative, got %d", c.Check.MaxCycles)
	}
	return nil
}

// Load returns the configuration for an analysis of target. A non-empty
// explicit path names the file to read; an empty one discovers it.
func Load(explicit, target string) (*Config, error) {
	if os.Getenv(legacyEnvVar) != "" {
		return nil, fmt.Errorf("%s is no longer read: move its settings to %s and pass it with --config (see %s)",
			legacyEnvVar, FileName, DocsURL)
	}
	path := explicit
	if path == "" {
		var err error
		path, err = discover(target)
		if err != nil {
			return nil, err
		}
		if path == "" {
			return Default(), nil
		}
	}
	return read(path)
}

// discover returns the nearest .polyscan.toml at or above target, or an
// empty path when there is none. A jscan configuration file met first is an
// error.
func discover(target string) (string, error) {
	dir, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		for _, name := range legacyFileNames {
			legacy := filepath.Join(dir, name)
			if _, err := os.Stat(legacy); err == nil {
				return "", fmt.Errorf("%s is no longer read: move its settings to %s in the same directory (see %s)",
					legacy, FileName, DocsURL)
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// read parses and validates one file over the defaults, so a key the file
// leaves out keeps its default.
func read(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration: %w", err)
	}
	cfg := Default()
	decoder := toml.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(cfg); err != nil {
		var strict *toml.StrictMissingError
		if errors.As(err, &strict) {
			keys := make([]string, len(strict.Errors))
			for i, keyErr := range strict.Errors {
				keys[i] = strings.Join(keyErr.Key(), ".")
			}
			return nil, fmt.Errorf("%s: unknown key(s): %s (see %s)", path, strings.Join(keys, ", "), DocsURL)
		}
		var decodeErr *toml.DecodeError
		if errors.As(err, &decodeErr) {
			row, column := decodeErr.Position()
			return nil, fmt.Errorf("%s:%d:%d: %w", path, row, column, err)
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}
