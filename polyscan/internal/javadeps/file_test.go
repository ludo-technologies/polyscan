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
		Types: []Type{{Name: "App"}},
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

func TestParseFileTypes(t *testing.T) {
	file, err := ParseFile([]byte(`
package com.example;

public interface Shape {
    class Nested {}
}
public abstract class Base {}
public final class Circle extends Base implements Shape {}
abstract class Hidden {}
@Deprecated public @interface Marker {}
public enum Color { RED }
record Point(int x, int y) {}
`))
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	// Nested is a member of Shape, not a type of the package.
	want := []Type{
		{Name: "Shape", Public: true, Abstract: true},
		{Name: "Base", Public: true, Abstract: true},
		{Name: "Circle", Public: true},
		{Name: "Hidden", Abstract: true},
		{Name: "Marker", Public: true, Abstract: true},
		{Name: "Color", Public: true},
		{Name: "Point"},
	}
	if !reflect.DeepEqual(file.Types, want) {
		t.Errorf("types = %+v\nwant    %+v", file.Types, want)
	}
}
