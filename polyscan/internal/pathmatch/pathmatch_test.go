package pathmatch

import "testing"

func TestMatches(t *testing.T) {
	for _, tc := range []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"pkg/sum_test.go", []string{"*_test.go"}, true},
		{"pkg/sum.go", []string{"*_test.go"}, false},
		{"tests/integration.rs", []string{"tests"}, true},
		{"src/utils/distance.ts", []string{"dist"}, false},
		{"dist/bundle.js", []string{"dist"}, true},
		{"src/generated/api/client.ts", []string{"src/generated/**"}, true},
		{"src/generated/api/client.ts", []string{"src/generated"}, true},
		{"lib/src/generated/x.go", []string{"src/generated"}, true},
		{"Widget.TS", []string{"**/*.ts"}, true},
		{"a.go", []string{"", "["}, false},
	} {
		if got := Matches(tc.path, tc.patterns); got != tc.want {
			t.Errorf("Matches(%q, %v) = %v, want %v", tc.path, tc.patterns, got, tc.want)
		}
	}
}
