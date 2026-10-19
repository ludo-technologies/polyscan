package rustdeps

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseUses(t *testing.T) {
	uses, err := ParseUses([]byte(`
use std::sync::Arc;
use wtransport::{Endpoint, ServerConfig as WtConfig};
use crate::net::{self, tcp::{Listener}, *};
use std::io::Write as _;
use Local;
mod inner {
    use super::config::ServerConfig;
    use rustls::ServerConfig;
}
fn f() {
    use std::fmt::{
        // the formatter
        Display,
    };
}
`))
	if err != nil {
		t.Fatalf("ParseUses: %v", err)
	}
	want := map[string][]string{
		"Arc":          {"std::sync::Arc"},
		"Endpoint":     {"wtransport::Endpoint"},
		"WtConfig":     {"wtransport::ServerConfig"},
		"net":          {"crate::net"},
		"Listener":     {"crate::net::tcp::Listener"},
		"Local":        {"Local"},
		"ServerConfig": {"super::config::ServerConfig", "rustls::ServerConfig"},
		"Display":      {"std::fmt::Display"},
	}
	if !reflect.DeepEqual(uses, want) {
		t.Errorf("uses = %v\nwant   %v", uses, want)
	}
}

func TestCrate(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		// A virtual workspace manifest names no crate.
		"Cargo.toml": "[workspace]\nmembers = [\"crates/*\"]\n",
		"crates/net-core/Cargo.toml": `[package]
name = "net-core"

[dependencies]
serde = "1"
tokio = { version = "1", features = ["full"] }
settings = { package = "app-config", path = "../config" }

[dev-dependencies]
test-util = { path = "../test-util" }

[target.'cfg(unix)'.dependencies]
sys = { package = "net-sys", path = "../sys" }
`,
		"crates/renamed/Cargo.toml":      "[package]\nname = \"renamed\"\n\n[lib]\nname = \"other\"\n",
		"crates/net-core/src/lib.rs":     "",
		"crates/renamed/src/bin/main.rs": "",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := NewResolver()
	for dir, want := range map[string]*Crate{
		"crates/net-core/src": {Package: "net-core", Name: "net_core", Dependencies: map[string]string{
			"serde": "serde", "tokio": "tokio", "settings": "app-config", "test_util": "test-util", "sys": "net-sys",
		}},
		"crates/renamed/src/bin": {Package: "renamed", Name: "other", Dependencies: map[string]string{}},
		"crates":                 nil,
	} {
		got, err := r.Crate(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("Crate(%s): %v", dir, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Crate(%s) = %+v, want %+v", dir, got, want)
		}
	}

	for _, manifest := range []string{"[package\n", "[package]\nversion = \"0.1.0\"\n"} {
		if err := os.WriteFile(filepath.Join(root, "crates/net-core/Cargo.toml"), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := NewResolver().Crate(filepath.Join(root, "crates/net-core/src")); err == nil {
			t.Errorf("Crate with Cargo.toml %q: want an error", manifest)
		}
	}
}
