package config

import "testing"

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"*_test.go", "a/b/c_test.go", true},
		{"*_test.go", "c.go", false},
		{"**/*_test.go", "c_test.go", true},
		{"ref/**", "ref", true},
		{"ref/**", "ref/oxc/x.go", true},
		{"ref/**", "src/ref/x.go", false},
		{"gen/*.go", "gen/a.go", true},
		{"gen/*.go", "gen/sub/a.go", false},
		{"./gen/*.go", "gen/a.go", true},
		{"a/**/z.go", "a/z.go", true},
		{"a/**/z.go", "a/b/c/z.go", true},
	}
	for _, tt := range tests {
		if got := Match(tt.pattern, tt.path); got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}
