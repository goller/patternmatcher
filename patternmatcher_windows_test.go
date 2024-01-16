package patternmatcher

import (
	"os"
	"path"
	"strings"
	"testing"
)

func TestWindowsCompile(t *testing.T) {
	for _, tt := range compileTests {
		pattern := path.Clean(tt.pattern)
		pathSeparator := string(os.PathSeparator)
		if pathSeparator != "/" {
			pattern = strings.ReplaceAll(pattern, "/", pathSeparator)
		}
		newp, err := NewPattern(pattern)
		if err != nil {
			t.Fatalf("Failed to compile pattern %q: %v", pattern, err)
		}

		newp.Dirs = strings.Split(pattern, pathSeparator)

		if newp.MatchType != tt.matchType {
			t.Errorf("pattern %q: matchType = %v, want %v", pattern, newp.MatchType, tt.matchType)
			continue
		}
		if tt.matchType == RegexpMatch {
			if pathSeparator == `\` {
				if newp.Regexp.String() != tt.windowsCompiledRegexp {
					t.Errorf("pattern %q: regexp = %s, want %s", pattern, newp.Regexp, tt.windowsCompiledRegexp)
				}
			} else if newp.Regexp.String() != tt.compiledRegexp {
				t.Errorf("pattern %q: regexp = %s, want %s", pattern, newp.Regexp, tt.compiledRegexp)
			}
		}
	}
}
