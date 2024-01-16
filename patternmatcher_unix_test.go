//go:build !windows
// +build !windows

package patternmatcher

import (
	"path"
	"testing"
)

// TestCompile confirms that "compile" assigns the correct match type to a
// variety of test case patterns. If the match type is regexp, it also confirms
// that the compiled regexp matches the expected regexp.
func TestCompile(t *testing.T) {
	for _, tt := range compileTests {
		pattern := path.Clean(tt.pattern)

		newp, err := NewPattern(pattern)
		if err != nil {
			t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
		}

		if newp.MatchType != tt.matchType {
			t.Errorf("pattern %q: matchType = %v, want %v", pattern, newp.MatchType, tt.matchType)
			continue
		}
		if tt.matchType == RegexpMatch {
			if newp.Regexp.String() != tt.compiledRegexp {
				t.Errorf("pattern %q: regexp = %s, want %s", pattern, newp.Regexp, tt.compiledRegexp)
			}
		}
	}
}
