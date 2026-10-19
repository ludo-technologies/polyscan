package rustdeps

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Crate is the package a Cargo.toml declares.
type Crate struct {
	// Package is the package name as written.
	Package string
	// Name is the name the package's own binaries, examples and tests use
	// for its library: the [lib] name, or else the package name with
	// hyphens turned into underscores.
	Name string
	// Dependencies maps the name the crate's code uses for each dependency
	// to the dependency's package name. A dependency renamed with
	// package = "..." is used under its key, so settings = { package =
	// "my-proxy" } maps settings to my-proxy.
	Dependencies map[string]string
}

// dependencies is one dependency table, whose values are a version string or
// a table with an optional package key.
type dependencies map[string]any

// manifest is the part of a Cargo.toml that names the crate and its
// dependencies.
type manifest struct {
	Package *struct {
		Name string `toml:"name"`
	} `toml:"package"`
	Lib struct {
		Name string `toml:"name"`
	} `toml:"lib"`
	Dependencies      dependencies `toml:"dependencies"`
	DevDependencies   dependencies `toml:"dev-dependencies"`
	BuildDependencies dependencies `toml:"build-dependencies"`
	Target            map[string]struct {
		Dependencies      dependencies `toml:"dependencies"`
		DevDependencies   dependencies `toml:"dev-dependencies"`
		BuildDependencies dependencies `toml:"build-dependencies"`
	} `toml:"target"`
}

// Resolver maps directories to the crate their Cargo.toml declares and
// remembers every answer, since a crate's files share one Cargo.toml.
type Resolver struct {
	byDir map[string]*Crate
}

// NewResolver returns an empty Resolver.
func NewResolver() *Resolver {
	return &Resolver{byDir: map[string]*Crate{}}
}

// Crate returns the crate of the nearest Cargo.toml at or above dir. It is
// nil when no directory up to the root has a Cargo.toml, or when the
// nearest one is a workspace manifest without a [package].
func (r *Resolver) Crate(dir string) (*Crate, error) {
	if crate, ok := r.byDir[dir]; ok {
		return crate, nil
	}
	var crate *Crate
	path := filepath.Join(dir, "Cargo.toml")
	if content, err := os.ReadFile(path); err == nil {
		if crate, err = parseManifest(content); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	} else if parent := filepath.Dir(dir); parent != dir {
		if crate, err = r.Crate(parent); err != nil {
			return nil, err
		}
	}
	r.byDir[dir] = crate
	return crate, nil
}

func parseManifest(content []byte) (*Crate, error) {
	var m manifest
	if err := toml.Unmarshal(content, &m); err != nil {
		return nil, err
	}
	if m.Package == nil {
		return nil, nil
	}
	if m.Package.Name == "" {
		return nil, fmt.Errorf("[package] has no name")
	}
	crate := &Crate{
		Package:      m.Package.Name,
		Name:         m.Lib.Name,
		Dependencies: map[string]string{},
	}
	if crate.Name == "" {
		crate.Name = strings.ReplaceAll(m.Package.Name, "-", "_")
	}
	tables := []dependencies{m.Dependencies, m.DevDependencies, m.BuildDependencies}
	for _, target := range m.Target {
		tables = append(tables, target.Dependencies, target.DevDependencies, target.BuildDependencies)
	}
	for _, table := range tables {
		for key, value := range table {
			pkg := key
			if spec, ok := value.(map[string]any); ok {
				if renamed, ok := spec["package"].(string); ok {
					pkg = renamed
				}
			}
			crate.Dependencies[strings.ReplaceAll(key, "-", "_")] = pkg
		}
	}
	return crate, nil
}
