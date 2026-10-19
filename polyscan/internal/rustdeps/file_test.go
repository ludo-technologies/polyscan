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

func TestCrateName(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		// A virtual workspace manifest names no crate.
		"Cargo.toml":                     "[workspace]\nmembers = [\"crates/*\"]\n",
		"crates/net-core/Cargo.toml":     "[package]\nname = \"net-core\"\n",
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
	for dir, want := range map[string]string{
		"crates/net-core/src":    "net_core",
		"crates/renamed/src/bin": "other",
		"crates":                 "",
	} {
		got, err := r.CrateName(filepath.Join(root, dir))
		if err != nil {
			t.Fatalf("CrateName(%s): %v", dir, err)
		}
		if got != want {
			t.Errorf("CrateName(%s) = %q, want %q", dir, got, want)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "crates/net-core/Cargo.toml"), []byte("[package\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewResolver().CrateName(filepath.Join(root, "crates/net-core/src")); err == nil {
		t.Error("CrateName with a malformed Cargo.toml: want an error")
	}
}
