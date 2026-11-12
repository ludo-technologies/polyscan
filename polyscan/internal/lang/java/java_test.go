package java

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/ludo-technologies/polyscan/polyscan/internal/engine"
)

func analyze(t *testing.T, source string) map[string]engine.Function {
	t.Helper()
	result, err := Language.Analyze([]byte(source))
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if result.SyntaxError != nil {
		t.Fatalf("syntax error: %v", result.SyntaxError)
	}
	byName := map[string]engine.Function{}
	for _, fn := range result.Functions {
		byName[fn.Name] = fn
	}
	return byName
}

func TestExtractsMethodsAndConstructors(t *testing.T) {
	functions := analyze(t, `
package p;
class Outer {
  Outer() {}
  int method(int x) { return x; }
  static <T> T generic(T t) { return t; }
  abstract void declared();
  native void bound();
  class Inner { void deep() {} }
  void anonymous() {
    new Thread() { public void run() {} };
    new java.util.function.Supplier<String>() { public String get() { return ""; } };
    Runnable r = () -> {};
  }
  void local() { class Local { void inLocal() {} } }
}
interface I { void required(); default void provided() {} static void helper() {} }
enum E { A { void constant() {} }; void shared() {} }
record R(int a) { R { } R(String s) { this(0); } int doubled() { return a * 2; } }
@interface Ann { String value() default ""; }
`)

	want := map[string]string{
		"Outer.Outer":         "constructor",
		"Outer.method":        "method",
		"Outer.generic":       "method",
		"Outer.Inner.deep":    "method",
		"Outer.anonymous":     "method",
		"Outer.Thread.run":    "method",
		"Outer.Supplier.get":  "method",
		"Outer.local":         "method",
		"Outer.Local.inLocal": "method",
		"I.provided":          "method",
		"I.helper":            "method",
		"E.A.constant":        "method",
		"E.shared":            "method",
		"R.R":                 "constructor",
		"R.doubled":           "method",
	}
	// The record's compact and canonical constructors share the name R.
	if len(functions) != len(want) {
		t.Errorf("got %d functions, want %d: %v", len(functions), len(want), names(functions))
	}
	for name, kind := range want {
		fn, ok := functions[name]
		if !ok {
			t.Errorf("missing function %q", name)
			continue
		}
		if fn.Kind != kind {
			t.Errorf("%s: kind = %q, want %q", name, fn.Kind, kind)
		}
	}
}

func TestReceivers(t *testing.T) {
	functions := analyze(t, `
class C { void m() {} void anonymous() { new Runnable() { public void run() {} }; } }
interface I { default void d() {} }
enum E { A { void constant() {} }; void shared() {} }
record R(int a) { int doubled() { return a * 2; } }
`)
	want := map[string]string{
		"C.m":            "C",
		"C.anonymous":    "C",
		"C.Runnable.run": "",
		"I.d":            "",
		"E.A.constant":   "",
		"E.shared":       "E",
		"R.doubled":      "R",
	}
	for name, receiver := range want {
		if got := functions[name].Receiver; got != receiver {
			t.Errorf("%s: receiver = %q, want %q", name, got, receiver)
		}
	}
}

func TestComplexityCountsDecisionPoints(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		want      int
		decisions map[string]int
	}{
		{"none", `return n;`, 1, map[string]int{}},
		{"if", `if (a) return 1; return 0;`, 2, map[string]int{"branch": 1}},
		{"if else", `if (a) { return 1; } else { return 0; }`, 2, map[string]int{"branch": 1}},
		{"else if chain", `if (a) { } else if (b) { } else { } return 0;`, 3, map[string]int{"branch": 2}},
		{"ternary", `return a ? 1 : 0;`, 2, map[string]int{"ternary": 1}},
		{"for", `for (int i = 0; i < 3; i++) { } return 0;`, 2, map[string]int{"loop": 1}},
		{"enhanced for", `for (int x : xs) { } return 0;`, 2, map[string]int{"loop": 1}},
		{"while", `while (a) { } return 0;`, 2, map[string]int{"loop": 1}},
		{"do while", `do { } while (a); return 0;`, 2, map[string]int{"loop": 1}},
		{"switch cases without default", `switch (n) { case 1: return 1; case 2: case 3: return 2; default: return 0; }`, 4, map[string]int{"case": 3}},
		{"arrow switch", `switch (n) { case 1 -> { return 1; } case 2, 3 -> { return 2; } default -> { return 0; } }`, 3, map[string]int{"case": 2}},
		{"switch expression", `return switch (n) { case 1 -> 1; case 2 -> 2; default -> 0; };`, 3, map[string]int{"case": 2}},
		{"pattern switch with a guard", `return switch (o) { case Integer i when i > 0 -> 1; case Integer i -> 2; default -> 0; };`, 4, map[string]int{"case": 2, "branch": 1}},
		{"try catch", `try { return 1; } catch (IllegalStateException e) { return 2; } catch (RuntimeException e) { return 3; } finally { n++; }`, 3, map[string]int{"exception": 2}},
		{"multi-catch counts once", `try { return 1; } catch (IllegalStateException | IllegalArgumentException e) { return 2; }`, 2, map[string]int{"exception": 1}},
		{"try with resources", `try (var r = open()) { return 1; } catch (Exception e) { return 2; }`, 2, map[string]int{"exception": 1}},
		{"logical operators", `if (a && b || c) { } return 0;`, 4, map[string]int{"branch": 1, "logical_operator": 2}},
		{"bitwise operators do not count", `return n & 1 | 2;`, 1, map[string]int{}},
		{"throw does not count", `throw new IllegalStateException();`, 1, map[string]int{}},
		{"lambda counts toward its enclosing method", `java.util.function.IntUnaryOperator f = z -> { if (z > 0) return z; return 0; }; return f.applyAsInt(n);`, 2, map[string]int{"branch": 1}},
		{"anonymous class initializer counts toward its enclosing method", `return new java.util.HashMap<String, Integer>() {{ if (a) put("x", 1); }}.size();`, 2, map[string]int{"branch": 1}},
		{"local class initializer counts toward its enclosing method", `class L { int v = a ? 1 : 0; } return new L().v;`, 2, map[string]int{"ternary": 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := analyze(t, "class C {\n  int f(boolean a, boolean b, boolean c, int n, int[] xs, Object o) {\n\t"+tc.body+"\n  }\n}\n")["C.f"]
			if fn.Complexity != tc.want {
				t.Errorf("complexity = %d, want %d", fn.Complexity, tc.want)
			}
			if len(fn.Decisions) != len(tc.decisions) {
				t.Errorf("decisions = %v, want %v", fn.Decisions, tc.decisions)
			}
			for kind, count := range tc.decisions {
				if fn.Decisions[kind] != count {
					t.Errorf("decisions[%q] = %d, want %d", kind, fn.Decisions[kind], count)
				}
			}
		})
	}
}

func TestAnonymousClassMethodsCountOnTheirOwn(t *testing.T) {
	functions := analyze(t, `
class C {
  void outer(boolean a) {
    if (a) {}
    new Runnable() { public void run() { if (a) {} while (a) {} } };
  }
}
`)
	if got := functions["C.outer"].Complexity; got != 2 {
		t.Errorf("outer complexity = %d, want 2", got)
	}
	if got := functions["C.Runnable.run"].Complexity; got != 3 {
		t.Errorf("run complexity = %d, want 3", got)
	}
}

func TestMembers(t *testing.T) {
	functions := analyze(t, `
class C {
  private int a, b = 1;
  int size;
  static int count;
  private static final String NAME = "c";
  C(int a) { this.a = a; size = 0; }
  static void helper() { count++; }
  @SuppressWarnings("static") void annotated() { a++; }
  void explicit() { this.a = this.b; this.helper(); this.count++; this.inherited = 1; }
  void bare() { b = a + count; foo(); NAME.length(); }
  void notFields(Other other, List<Integer> list) {
    other.a = 1;
    list.size();
    int n = size();
  }
  void references() { use(this::bare); use(C::helper); use(items::add); }
  int arrays() { return items[a]; }
  int[] items;
}
`)
	cases := []struct {
		name    string
		hasSelf bool
		fields  []string
		calls   []string
	}{
		{"C.C", false, nil, nil},
		{"C.helper", false, nil, nil},
		{"C.annotated", true, []string{"a"}, []string{}},
		{"C.explicit", true, []string{"a", "b"}, []string{"helper"}},
		{"C.bare", true, []string{"a", "b"}, []string{"foo"}},
		{"C.notFields", true, []string{}, []string{"size"}},
		{"C.references", true, []string{"items"}, []string{"bare", "use"}},
		{"C.arrays", true, []string{"a", "items"}, []string{}},
	}
	for _, tc := range cases {
		fn, ok := functions[tc.name]
		if !ok {
			t.Errorf("missing function %q", tc.name)
			continue
		}
		if fn.Receiver != "C" || fn.HasSelf != tc.hasSelf {
			t.Errorf("%s: receiver %q hasSelf %v, want C %v", tc.name, fn.Receiver, fn.HasSelf, tc.hasSelf)
		}
		if got := slices.Sorted(maps.Keys(fn.Fields)); !slices.Equal(got, tc.fields) {
			t.Errorf("%s: fields = %v, want %v", tc.name, got, tc.fields)
		}
		if got := slices.Sorted(maps.Keys(fn.Calls)); !slices.Equal(got, tc.calls) {
			t.Errorf("%s: calls = %v, want %v", tc.name, got, tc.calls)
		}
	}
}

func TestMembersOfNestedRecordAndEnum(t *testing.T) {
	functions := analyze(t, `
class Outer {
  int shared;
  class Inner {
    int own;
    void touch() { own++; shared++; Outer.this.shared++; }
  }
  void run() { new Runnable() { public void run() { shared++; } }; }
  void lambda() { Runnable r = () -> shared++; }
}
record Point(int x, int y) {
  Point { if (x < 0) throw new IllegalArgumentException(); }
  int sum() { return x + y; }
  static Point origin() { return new Point(0, 0); }
}
record Bag(String... items) { int size() { return items.length; } }
enum Level {
  LOW(1), HIGH(2);
  private final int weight;
  private static int created;
  Level(int weight) { this.weight = weight; }
  int weight() { return weight + created; }
  boolean low() { return this == LOW; }
}
interface I { int LIMIT = 3; default int limit() { return LIMIT; } }
`)
	cases := []struct {
		name     string
		receiver string
		hasSelf  bool
		fields   []string
	}{
		{"Outer.Inner.touch", "Outer.Inner", true, []string{"own"}},
		{"Outer.run", "Outer", true, []string{}},
		{"Outer.Runnable.run", "", false, nil},
		{"Outer.lambda", "Outer", true, []string{"shared"}},
		{"Point.Point", "Point", false, nil},
		{"Point.sum", "Point", true, []string{"x", "y"}},
		{"Point.origin", "Point", false, nil},
		{"Bag.size", "Bag", true, []string{"items"}},
		{"Level.Level", "Level", false, nil},
		{"Level.weight", "Level", true, []string{"weight"}},
		{"Level.low", "Level", true, []string{}},
		{"I.limit", "", false, nil},
	}
	for _, tc := range cases {
		fn, ok := functions[tc.name]
		if !ok {
			t.Errorf("missing function %q", tc.name)
			continue
		}
		if fn.Receiver != tc.receiver || fn.HasSelf != tc.hasSelf {
			t.Errorf("%s: receiver %q hasSelf %v, want %q %v", tc.name, fn.Receiver, fn.HasSelf, tc.receiver, tc.hasSelf)
		}
		if got := slices.Sorted(maps.Keys(fn.Fields)); !slices.Equal(got, tc.fields) {
			t.Errorf("%s: fields = %v, want %v", tc.name, got, tc.fields)
		}
	}
}

func TestMembersIgnoreShadowedFields(t *testing.T) {
	functions := analyze(t, `
class C {
  int a, b, c, d, e, f, g, h;
  void param(int a) { a++; b++; }
  void varargs(int... a) { use(a); }
  void local() { c++; int c = 1; c++; }
  void block() { { int d = 1; d++; } d++; }
  void loop() { for (int e = 0; e < 3; e++) {} e++; }
  void each() { for (int f : f(this)) { f++; } }
  void group(int k) { switch (k) { case 1: int g = 1; break; case 2: g = 2; break; } }
  void caught() { try { } catch (RuntimeException h) { h.printStackTrace(); } }
  void lambdas() { java.util.function.IntUnaryOperator l = a -> a + 1; Fn m = (b, c) -> b; Fn n = (int d) -> d; }
  void resource() { try (var e = open()) { e.read(); } }
  void pattern(Object o) { if (o instanceof String f) { f.length(); } }
  void typePattern(Object o) { switch (o) { case String g -> g.length(); default -> {} } }
  void recordPattern(Object o) { if (o instanceof Pair(int h, int ignored)) { h++; } }
  void explicit(int a) { this.a = a; }
}
`)
	want := map[string][]string{
		"C.param":         {"b"},
		"C.varargs":       {},
		"C.local":         {"c"},
		"C.block":         {"d"},
		"C.loop":          {"e"},
		"C.each":          {},
		"C.group":         {},
		"C.caught":        {},
		"C.lambdas":       {},
		"C.resource":      {},
		"C.pattern":       {},
		"C.typePattern":   {},
		"C.recordPattern": {},
		"C.explicit":      {"a"},
	}
	for name, fields := range want {
		if got := slices.Sorted(maps.Keys(functions[name].Fields)); !slices.Equal(got, fields) {
			t.Errorf("%s: fields = %v, want %v", name, got, fields)
		}
	}
}

func TestContentDropsComments(t *testing.T) {
	fn := analyze(t, "class C {\n  /** Doc. */\n  int f() {\n\t// line\n\treturn/* block */1;\n  }\n}\n")["C.f"]
	if strings.Contains(fn.Content, "line") || strings.Contains(fn.Content, "block") {
		t.Errorf("content keeps comments: %q", fn.Content)
	}
	if !strings.Contains(fn.Content, "return 1") || fn.CodeLines != 3 {
		t.Errorf("content = %q, code lines = %d", fn.Content, fn.CodeLines)
	}
}

func TestTestFiles(t *testing.T) {
	for path, want := range map[string]bool{
		"src/main/java/p/Foo.java":            false,
		"src/main/java/p/Testcontainers.java": false,
		"src/main/java/p/Latest.java":         false,
		"src/main/java/p/FooTest.java":        true,
		"src/main/java/p/FooTests.java":       true,
		"src/main/java/p/FooTestCase.java":    true,
		"src/main/java/p/FooIT.java":          true,
		"src/test/java/p/Fixtures.java":       true,
		"tests/p/Fixtures.java":               true,
		"javatests/com/p/Fixtures.java":       true,
		"app/src/androidTest/java/p/Foo.java": true,
		"src/main/java/p/latest/Foo.java":     false,
		"src/main/java/p/testing/Helper.java": false,
	} {
		if got := Language.IsTestFile(path); got != want {
			t.Errorf("IsTestFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSyntaxErrorKeepsOtherMethods(t *testing.T) {
	result, err := Language.Analyze([]byte(`
class C {
  int first(boolean a) { if (a) return 1; return 0; }
  int broken(boolean a) { if ( return 1; }
  int last() { return 2; }
}
`))
	if err != nil {
		t.Fatal(err)
	}
	if result.SyntaxError == nil {
		t.Fatal("expected a syntax error")
	}
	complexity := map[string]int{}
	for _, fn := range result.Functions {
		complexity[fn.Name] = fn.Complexity
	}
	if complexity["C.first"] != 2 || complexity["C.last"] != 1 {
		t.Errorf("functions = %v, want first and last", complexity)
	}
	if _, ok := complexity["C.broken"]; ok {
		t.Error("a method containing the error must not be analyzed")
	}
}

func TestNestingDepth(t *testing.T) {
	functions := analyze(t, `
class C {
  void flat(boolean a) {
    if (a) {}
    while (a) {}
  }

  void elseIfContinuesTheChain(boolean a, boolean b) {
    if (a) {
    } else if (b) {
      for (;;) {}
    } else {
      if (a) { if (b) {} }
    }
  }

  void deep(boolean a, int[] xs) {
    if (a) {
      for (int x : xs) {
        switch (x) {
          case 1:
            try { } catch (Exception e) { do {} while (a); }
        }
      }
    }
  }

  int switchExpression(int n, boolean a) {
    return switch (n) {
      case 1 -> { if (a) yield 1; yield 2; }
      default -> 0;
    };
  }

  void resources(boolean a) {
    try (var r = open()) { if (a) {} }
  }

  void lambda(boolean a, boolean b) {
    Runnable l = () -> { if (a) { if (b) {} } };
  }
}
`)
	want := map[string]int{
		"C.flat":                    1,
		"C.elseIfContinuesTheChain": 3,
		"C.deep":                    5,
		"C.switchExpression":        2,
		"C.resources":               2,
		"C.lambda":                  2,
	}
	for name, depth := range want {
		fn, ok := functions[name]
		if !ok {
			t.Errorf("missing function %q", name)
			continue
		}
		if fn.NestingDepth != depth {
			t.Errorf("%s: nesting depth = %d, want %d", name, fn.NestingDepth, depth)
		}
	}
}

func TestEffectiveComplexityCollapsesFlatDispatch(t *testing.T) {
	cases := []struct {
		name string
		body string
		// want is the complexity, effective the value the risk level uses.
		want, effective int
	}{
		{"flat switch", `switch (n) { case 1: return 1; case 2: return 2; case 3: return 3; } return 0;`, 4, 2},
		{"flat switch with default", `switch (n) { case 1: return 1; case 2: return 2; default: return 0; }`, 3, 2},
		{"fallthrough arms stay flat", `switch (n) { case 1: case 2: return 1; case 3: return 3; } return 0;`, 4, 2},
		{"flat arrow switch", `return switch (n) { case 1 -> 1; case 2 -> 2; case 3 -> 3; default -> 0; };`, 4, 2},
		{"a branching arm keeps every arm", `switch (n) { case 1: return 1; case 2: if (a) return 2; return 0; case 3: return 3; } return 0;`, 5, 5},
		{"a guarded arm keeps every arm", `return switch (o) { case Integer i when i > 0 -> 1; case String s -> 2; default -> 0; };`, 4, 4},
		{"an arm with a catch keeps every arm", `switch (n) { case 1: return 1; case 2: try { return 2; } catch (Exception e) { return 0; } case 3: return 3; } return 0;`, 5, 5},
		{"a branching default keeps every arm", `switch (n) { case 1: return 1; case 2: return 2; default: if (a) return 9; return 0; }`, 4, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := analyze(t, "class C {\n  int f(boolean a, int n, Object o) {\n\t"+tc.body+"\n  }\n}\n")["C.f"]
			if fn.Complexity != tc.want {
				t.Errorf("complexity = %d, want %d", fn.Complexity, tc.want)
			}
			if fn.EffectiveComplexity != tc.effective {
				t.Errorf("effective complexity = %d, want %d", fn.EffectiveComplexity, tc.effective)
			}
		})
	}
}

func TestEffectiveComplexityCollapsesReturnedPredicate(t *testing.T) {
	cases := []struct {
		name string
		body string
		// want is the complexity, effective the value the risk level uses.
		want, effective int
	}{
		{"a returned chain counts once", `return a && b && n > 0 && n < 9;`, 4, 2},
		{"a guard and a returned chain", `if (!a || !b) return false; return a && n > 0 && n < 9;`, 5, 4},
		{"operators in a condition still count", `if (a && b) return true; return false;`, 3, 3},
		{"operators outside a return still count", `boolean c = a && b && n > 0; return c;`, 3, 3},
		{"an expression lambda's chain collapses on its own", `java.util.function.BooleanSupplier f = () -> a && b && n > 0; return f.getAsBoolean();`, 3, 2},
		{"a block lambda's returned chain collapses on its own", `java.util.function.BooleanSupplier f = () -> { return a && b && n > 0; }; return f.getAsBoolean();`, 3, 2},
		{"a ternary in the returned expression keeps every operator", `return a && b && (n > 0 ? a : b);`, 4, 4},
		{"a lambda's own statements stay outside the returned expression", `java.util.function.BooleanSupplier g = () -> { boolean c = a && b && n > 0; return c; }; return g.getAsBoolean() && a;`, 4, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := analyze(t, "class C {\n  boolean f(boolean a, boolean b, int n) {\n\t"+tc.body+"\n  }\n}\n")["C.f"]
			if fn.Complexity != tc.want {
				t.Errorf("complexity = %d, want %d", fn.Complexity, tc.want)
			}
			if fn.EffectiveComplexity != tc.effective {
				t.Errorf("effective complexity = %d, want %d", fn.EffectiveComplexity, tc.effective)
			}
		})
	}
}

func names(functions map[string]engine.Function) []string {
	var out []string
	for name := range functions {
		out = append(out, name)
	}
	return out
}
