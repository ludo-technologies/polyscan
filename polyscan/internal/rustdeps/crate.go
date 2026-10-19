package rustdeps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// manifest is the part of a Cargo.toml that names the library crate.
type manifest struct {
	Package *struct {
		Name string `toml:"name"`
	} `toml:"package"`
	Lib struct {
		Name string `toml:"name"`
	} `toml:"lib"`
}

// Resolver maps directories to the crate their Cargo.toml declares and
// remembers every answer, since a crate's files share one Cargo.toml.
type Resolver struct {
	byDir map[string]string
}

// NewResolver returns an empty Resolver.
func NewResolver() *Resolver {
	return &Resolver{byDir: map[string]string{}}
}

// CrateName returns the name the code of other crates and of the package's
// own binaries, examples and tests uses for the library crate of the
// nearest Cargo.toml at or above dir: its [lib] name, or else its package
// name with hyphens turned into underscores. The name is empty when no
// directory up to the root has a Cargo.toml, or when the nearest one is a
// workspace manifest without a [package].
func (r *Resolver) CrateName(dir string) (string, error) {
	if name, ok := r.byDir[dir]; ok {
		return name, nil
	}
	var name string
	path := filepath.Join(dir, "Cargo.toml")
	if content, err := os.ReadFile(path); err == nil {
		var m manifest
		if err := toml.Unmarshal(content, &m); err != nil {
			return "", fmt.Errorf("%s: %w", path, err)
		}
		switch {
		case m.Lib.Name != "":
			name = m.Lib.Name
		case m.Package != nil:
			name = strings.ReplaceAll(m.Package.Name, "-", "_")
		}
	} else if !os.IsNotExist(err) {
		return "", err
	} else if parent := filepath.Dir(dir); parent != dir {
		name, err = r.CrateName(parent)
		if err != nil {
			return "", err
		}
	}
	r.byDir[dir] = name
	return name, nil
}
