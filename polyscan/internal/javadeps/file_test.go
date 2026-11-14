package javadeps

import (
	"reflect"
	"testing"
)

func TestParseFile(t *testing.T) {
	file, err := ParseFile([]byte(`
@Deprecated
package com.example.app;

import java.util.List;
import java.util.*;
import com.example.Outer.Inner;
import static com.example.Util.run;
import static com.example.Constants.*;
import com.example
    .Wrapped;
import Plain;

class App {}
`))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	want := &File{
		Package: "com.example.app",
		Imports: []Import{
			{Name: "java.util.List"},
			{Name: "java.util", OnDemand: true},
			{Name: "com.example.Outer.Inner"},
			{Name: "com.example.Util.run"},
			{Name: "com.example.Constants", OnDemand: true},
			{Name: "com.example.Wrapped"},
			{Name: "Plain"},
		},
	}
	if !reflect.DeepEqual(file, want) {
		t.Errorf("file = %+v\nwant   %+v", file, want)
	}
}

func TestParseFileOfTheUnnamedPackage(t *testing.T) {
	file, err := ParseFile([]byte("class Main {}\n"))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if file.Package != "" || file.Imports != nil {
		t.Errorf("file = %+v, want no package and no imports", file)
	}
}
