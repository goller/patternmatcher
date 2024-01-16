package patternmatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWildcardMatches(t *testing.T) {
	match, _ := matches("fileutils.go", []string{"*"})
	if !match {
		t.Errorf("failed to get a wildcard match, got %v", match)
	}
}

// A simple pattern match should return true.
func TestPatternMatches(t *testing.T) {
	match, _ := matches("fileutils.go", []string{"*.go"})
	if !match {
		t.Errorf("failed to get a match, got %v", match)
	}
}

// An exclusion followed by an inclusion should return true.
func TestExclusionPatternMatchesPatternBefore(t *testing.T) {
	match, _ := matches("fileutils.go", []string{"!fileutils.go", "*.go"})
	if !match {
		t.Errorf("failed to get true match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderExclusions(t *testing.T) {
	match, _ := matches("docs/README.md", []string{"docs", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWithSlashExclusions(t *testing.T) {
	match, _ := matches("docs/README.md", []string{"docs/", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A folder pattern followed by an exception should return false.
func TestPatternMatchesFolderWildcardExclusions(t *testing.T) {
	match, _ := matches("docs/README.md", []string{"docs/*", "!docs/README.md"})
	if match {
		t.Errorf("failed to get a false match on exclusion pattern, got %v", match)
	}
}

// A pattern followed by an exclusion should return false.
func TestExclusionPatternMatchesPatternAfter(t *testing.T) {
	match, _ := matches("fileutils.go", []string{"*.go", "!fileutils.go"})
	if match {
		t.Errorf("failed to get false match on exclusion pattern, got %v", match)
	}
}

// A filename evaluating to . should return false.
func TestExclusionPatternMatchesWholeDirectory(t *testing.T) {
	match, _ := matches(".", []string{"*.go"})
	if match {
		t.Errorf("failed to get false match on ., got %v", match)
	}
}

// A single ! pattern should return an error.
func TestSingleExclamationError(t *testing.T) {
	_, err := matches("fileutils.go", []string{"!"})
	if err == nil {
		t.Errorf("failed to get an error for a single exclamation point, got %v", err)
	}
}

// Matches with no patterns
func TestMatchesWithNoPatterns(t *testing.T) {
	matches, err := matches("/any/path/there", []string{})
	if err != nil {
		t.Fatal(err)
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}

// Matches with malformed patterns
func TestMatchesWithMalformedPatterns(t *testing.T) {
	matches, err := matches("/any/path/there", []string{"["})
	if err == nil {
		t.Fatal("Should have failed because of a malformed syntax in the pattern")
	}
	if matches {
		t.Fatalf("Should not have match anything")
	}
}

type matchesTestCase struct {
	pattern string
	text    string
	pass    bool
}

type multiPatternTestCase struct {
	patterns []string
	text     string
	pass     bool
}

func TestMatches(t *testing.T) {
	tests := []matchesTestCase{
		{"**", "file", true},
		{"**", "file/", true},
		{"**/", "file", true}, // weird one
		{"**/", "file/", true},
		{"**", "/", true},
		{"**/", "/", true},
		{"**", "dir/file", true},
		{"**/", "dir/file", true},
		{"**", "dir/file/", true},
		{"**/", "dir/file/", true},
		{"**/**", "dir/file", true},
		{"**/**", "dir/file/", true},
		{"dir/**", "dir/file", true},
		{"dir/**", "dir/file/", true},
		{"dir/**", "dir/dir2/file", true},
		{"dir/**", "dir/dir2/file/", true},
		{"**/dir", "dir", true},
		{"**/dir", "dir/file", true},
		{"**/dir2/*", "dir/dir2/file", true},
		{"**/dir2/*", "dir/dir2/file/", true},
		{"**/dir2/**", "dir/dir2/dir3/file", true},
		{"**/dir2/**", "dir/dir2/dir3/file/", true},
		{"**file", "file", true},
		{"**file", "dir/file", true},
		{"**/file", "dir/file", true},
		{"**file", "dir/dir/file", true},
		{"**/file", "dir/dir/file", true},
		{"**/file*", "dir/dir/file", true},
		{"**/file*", "dir/dir/file.txt", true},
		{"**/file*txt", "dir/dir/file.txt", true},
		{"**/file*.txt", "dir/dir/file.txt", true},
		{"**/file*.txt*", "dir/dir/file.txt", true},
		{"**/**/*.txt", "dir/dir/file.txt", true},
		{"**/**/*.txt2", "dir/dir/file.txt", false},
		{"**/*.txt", "file.txt", true},
		{"**/**/*.txt", "file.txt", true},
		{"a**/*.txt", "a/file.txt", true},
		{"a**/*.txt", "a/dir/file.txt", true},
		{"a**/*.txt", "a/dir/dir/file.txt", true},
		{"a/*.txt", "a/dir/file.txt", false},
		{"a/*.txt", "a/file.txt", true},
		{"a/*.txt**", "a/file.txt", true},
		{"a[b-d]e", "ae", false},
		{"a[b-d]e", "ace", true},
		{"a[b-d]e", "aae", false},
		{"a[^b-d]e", "aze", true},
		{".*", ".foo", true},
		{".*", "foo", false},
		{"abc.def", "abcdef", false},
		{"abc.def", "abc.def", true},
		{"abc.def", "abcZdef", false},
		{"abc?def", "abcZdef", true},
		{"abc?def", "abcdef", false},
		{"a\\\\", "a\\", true},
		{"**/foo/bar", "foo/bar", true},
		{"**/foo/bar", "dir/foo/bar", true},
		{"**/foo/bar", "dir/dir2/foo/bar", true},
		{"abc/**", "abc", false},
		{"abc/**", "abc/def", true},
		{"abc/**", "abc/def/ghi", true},
		{"**/.foo", ".foo", true},
		{"**/.foo", "bar.foo", false},
		{"a(b)c/def", "a(b)c/def", true},
		{"a(b)c/def", "a(b)c/xyz", false},
		{"a.|)$(}+{bc", "a.|)$(}+{bc", true},
		{"dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", true},
		{"dist/*.whl", "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", true},
	}
	multiPatternTests := []multiPatternTestCase{
		{[]string{"**", "!util/docker/web"}, "util/docker/web/foo", false},
		{[]string{"**", "!util/docker/web", "util/docker/web/foo"}, "util/docker/web/foo", true},
		{[]string{"**", "!dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl"}, "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", false},
		{[]string{"**", "!dist/*.whl"}, "dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", false},
	}

	if runtime.GOOS != "windows" {
		tests = append(tests, []matchesTestCase{
			{"a\\*b", "a*b", true},
		}...)
	}

	t.Run("MatchesOrParentMatches", func(t *testing.T) {
		for _, test := range tests {
			patterns, err := NewPatterns([]string{test.pattern})
			if err != nil {
				t.Fatalf("%v (pattern=%q, text=%q)", err, test.pattern, test.text)
			}
			res, _ := MatchesOrParentMatches(patterns, test.text)
			if test.pass != res {
				t.Fatalf("%v (pattern=%q, text=%q)", err, test.pattern, test.text)
			}
		}

		for _, test := range multiPatternTests {
			patterns, err := NewPatterns(test.patterns)
			if err != nil {
				t.Fatalf("%v (patterns=%q, text=%q)", err, test.patterns, test.text)
			}
			res, _ := MatchesOrParentMatches(patterns, test.text)
			if test.pass != res {
				t.Errorf("expected: %v, got: %v (patterns=%q, text=%q)", test.pass, res, test.patterns, test.text)
			}
		}
	})

	t.Run("MatchesUsingParentResults", func(t *testing.T) {
		check := func(patterns []*Pattern, text string, pass bool, desc string) {
			parentPath := filepath.Dir(filepath.FromSlash(text))
			parentPathDirs := strings.Split(parentPath, string(os.PathSeparator))

			parentMatchInfo := MatchInfo{}
			if parentPath != "." {
				for i := range parentPathDirs {
					_, parentMatchInfo, _ = MatchesUsingParentResults(patterns, strings.Join(parentPathDirs[:i+1], "/"), parentMatchInfo)
				}
			}

			res, _, _ := MatchesUsingParentResults(patterns, text, parentMatchInfo)
			if pass != res {
				t.Errorf("expected: %v, got: %v %s", pass, res, desc)
			}
		}

		for _, test := range tests {
			desc := fmt.Sprintf("(pattern=%q text=%q)", test.pattern, test.text)
			patterns, err := NewPatterns([]string{test.pattern})
			if err != nil {
				t.Fatal(err, desc)
			}

			check(patterns, test.text, test.pass, desc)
		}

		for _, test := range multiPatternTests {
			desc := fmt.Sprintf("pattern=%q text=%q", test.patterns, test.text)
			patterns, err := NewPatterns(test.patterns)
			if err != nil {
				t.Fatal(err, desc)
			}

			check(patterns, test.text, test.pass, desc)
		}
	})

	t.Run("MatchesUsingParentResultsNoContext", func(t *testing.T) {
		check := func(patterns []*Pattern, text string, pass bool, desc string) {
			res, _, _ := MatchesUsingParentResults(patterns, text, MatchInfo{})
			if pass != res {
				t.Errorf("expected: %v, got: %v %s", pass, res, desc)
			}
		}

		for _, test := range tests {
			desc := fmt.Sprintf("(pattern=%q text=%q)", test.pattern, test.text)
			patterns, err := NewPatterns([]string{test.pattern})
			if err != nil {
				t.Fatal(err, desc)
			}

			check(patterns, test.text, test.pass, desc)
		}

		for _, test := range multiPatternTests {
			desc := fmt.Sprintf("(pattern=%q text=%q)", test.patterns, test.text)
			patterns, err := NewPatterns(test.patterns)
			if err != nil {
				t.Fatal(err, desc)
			}

			check(patterns, test.text, test.pass, desc)
		}
	})
}

func TestCleanPatterns(t *testing.T) {
	patterns, err := NewPatterns([]string{"docs", "config"})
	if err != nil {
		t.Fatalf("invalid pattern %v", err)
	}
	if len(patterns) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(patterns))
	}
}

func TestCleanPatternsStripEmptyPatterns(t *testing.T) {
	patterns, err := NewPatterns([]string{"docs", "config", ""})
	if err != nil {
		t.Fatalf("invalid pattern %v", err)
	}
	if len(patterns) != 2 {
		t.Errorf("expected 2 element slice, got %v", len(patterns))
	}
}

func TestCleanPatternsExceptionFlag(t *testing.T) {
	patterns, err := NewPatterns([]string{"docs", "!docs/README.md"})
	if err != nil {
		t.Fatalf("invalid pattern %v", err)
	}
	var exclusions bool
	for _, pattern := range patterns {
		exclusions = exclusions || pattern.Exclusion
	}
	if !exclusions {
		t.Errorf("expected exceptions to be true, got %v", exclusions)
	}
}

func TestCleanPatternsLeadingSpaceTrimmed(t *testing.T) {
	patterns, err := NewPatterns([]string{"docs", "  !docs/README.md"})
	if err != nil {
		t.Fatalf("invalid pattern %v", err)
	}
	var exclusions bool
	for _, pattern := range patterns {
		exclusions = exclusions || pattern.Exclusion
	}
	if !exclusions {
		t.Errorf("expected exceptions to be true, got %v", exclusions)
	}
}

func TestCleanPatternsTrailingSpaceTrimmed(t *testing.T) {
	patterns, err := NewPatterns([]string{"docs", "!docs/README.md  "})
	if err != nil {
		t.Fatalf("invalid pattern %v", patterns)
	}
	var exclusions bool
	for _, pattern := range patterns {
		exclusions = exclusions || pattern.Exclusion
	}
	if !exclusions {
		t.Errorf("expected exceptions to be true, got %v", exclusions)
	}
}

func TestCleanPatternsErrorSingleException(t *testing.T) {
	patterns := []string{"!"}
	_, err := NewPatterns(patterns)
	if err == nil {
		t.Errorf("expected error on single exclamation point, got %v", err)
	}
}

// These matchTests are stolen from go's filepath Match tests.
type matchTest struct {
	pattern, s string
	match      bool
	err        error
}

var matchTests = []matchTest{
	{"abc", "abc", true, nil},
	{"*", "abc", true, nil},
	{"*c", "abc", true, nil},
	{"a*", "a", true, nil},
	{"a*", "abc", true, nil},
	{"a*", "ab/c", true, nil},
	{"a*/b", "abc/b", true, nil},
	{"a*/b", "a/c/b", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/f", true, nil},
	{"a*b*c*d*e*/f", "axbxcxdxe/xxx/f", false, nil},
	{"a*b*c*d*e*/f", "axbxcxdxexxx/fff", false, nil},
	{"a*b?c*x", "abxbbxdbxebxczzx", true, nil},
	{"a*b?c*x", "abxbbxdbxebxczzy", false, nil},
	{"ab[c]", "abc", true, nil},
	{"ab[b-d]", "abc", true, nil},
	{"ab[e-g]", "abc", false, nil},
	{"ab[^c]", "abc", false, nil},
	{"ab[^b-d]", "abc", false, nil},
	{"ab[^e-g]", "abc", true, nil},
	{"a\\*b", "a*b", true, nil},
	{"a\\*b", "ab", false, nil},
	{"a?b", "a☺b", true, nil},
	{"a[^a]b", "a☺b", true, nil},
	{"a???b", "a☺b", false, nil},
	{"a[^a][^a][^a]b", "a☺b", false, nil},
	{"[a-ζ]*", "α", true, nil},
	{"*[a-ζ]", "A", false, nil},
	{"a?b", "a/b", false, nil},
	{"a*b", "a/b", false, nil},
	{"[\\]a]", "]", true, nil},
	{"[\\-]", "-", true, nil},
	{"[x\\-]", "x", true, nil},
	{"[x\\-]", "-", true, nil},
	{"[x\\-]", "z", false, nil},
	{"[\\-x]", "x", true, nil},
	{"[\\-x]", "-", true, nil},
	{"[\\-x]", "a", false, nil},
	{"[]a]", "]", false, filepath.ErrBadPattern},
	{"[-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "x", false, filepath.ErrBadPattern},
	{"[x-]", "-", false, filepath.ErrBadPattern},
	{"[x-]", "z", false, filepath.ErrBadPattern},
	{"[-x]", "x", false, filepath.ErrBadPattern},
	{"[-x]", "-", false, filepath.ErrBadPattern},
	{"[-x]", "a", false, filepath.ErrBadPattern},
	{"\\", "a", false, filepath.ErrBadPattern},
	{"[a-b-c]", "a", false, filepath.ErrBadPattern},
	{"[", "a", false, filepath.ErrBadPattern},
	{"[^", "a", false, filepath.ErrBadPattern},
	{"[^bc", "a", false, filepath.ErrBadPattern},
	{"a[", "a", false, filepath.ErrBadPattern}, // was nil but IMO its wrong
	{"a[", "ab", false, filepath.ErrBadPattern},
	{"*x", "xxx", true, nil},
}

func errp(e error) string {
	if e == nil {
		return "<nil>"
	}
	return e.Error()
}

// TestMatch tests our version of filepath.Match, called Matches.
func TestMatch(t *testing.T) {
	for _, tt := range matchTests {
		pattern := tt.pattern
		s := tt.s
		if runtime.GOOS == "windows" {
			if strings.Contains(pattern, "\\") {
				// no escape allowed on windows.
				continue
			}
			pattern = filepath.Clean(pattern)
			s = filepath.Clean(s)
		}
		ok, err := matches(s, []string{pattern})
		if ok != tt.match || err != tt.err {
			t.Fatalf("Match(%#q, %#q) = %v, %q want %v, %q", pattern, s, ok, errp(err), tt.match, errp(tt.err))
		}
	}
}

type compileTestCase struct {
	pattern               string
	matchType             MatchType
	compiledRegexp        string
	windowsCompiledRegexp string
}

var compileTests = []compileTestCase{
	{"*", RegexpMatch, `^[^/]*$`, `^[^\\]*$`},
	{"file*", RegexpMatch, `^file[^/]*$`, `^file[^\\]*$`},
	{"*file", RegexpMatch, `^[^/]*file$`, `^[^\\]*file$`},
	{"a*/b", RegexpMatch, `^a[^/]*/b$`, `^a[^\\]*\\b$`},
	{"**", SuffixMatch, "", ""},
	{"**/**", RegexpMatch, `^(.*/)?.*$`, `^(.*\\)?.*$`},
	{"dir/**", PrefixMatch, "", ""},
	{"**/dir", SuffixMatch, "", ""},
	{"**/dir2/*", RegexpMatch, `^(.*/)?dir2/[^/]*$`, `^(.*\\)?dir2\\[^\\]*$`},
	{"**/dir2/**", RegexpMatch, `^(.*/)?dir2/.*$`, `^(.*\\)?dir2\\.*$`},
	{"**file", SuffixMatch, "", ""},
	{"**/file*txt", RegexpMatch, `^(.*/)?file[^/]*txt$`, `^(.*\\)?file[^\\]*txt$`},
	{"**/**/*.txt", RegexpMatch, `^(.*/)?(.*/)?[^/]*\.txt$`, `^(.*\\)?(.*\\)?[^\\]*\.txt$`},
	{"a[b-d]e", RegexpMatch, `^a[b-d]e$`, `^a[b-d]e$`},
	{".*", RegexpMatch, `^\.[^/]*$`, `^\.[^\\]*$`},
	{"abc.def", ExactMatch, "", ""},
	{"abc?def", RegexpMatch, `^abc[^/]def$`, `^abc[^\\]def$`},
	{"**/foo/bar", SuffixMatch, "", ""},
	{"a(b)c/def", ExactMatch, "", ""},
	{"a.|)$(}+{bc", ExactMatch, "", ""},
	{"dist/proxy.py-2.4.0rc3.dev36+g08acad9-py3-none-any.whl", ExactMatch, "", ""},
}

// matches returns true if file matches any of the patterns
// and isn't excluded by any of the subsequent patterns.
//
// This implementation is buggy (it only checks a single parent dir against the
// pattern) and will be removed soon. Use MatchesOrParentMatches instead.
func matches(file string, patterns []string) (bool, error) {
	matchPatterns, err := NewPatterns(patterns)
	if err != nil {
		return false, err
	}
	file = filepath.Clean(file)

	if file == "." {
		// Don't let them exclude everything, kind of silly.
		return false, nil
	}

	matched := false
	file = filepath.FromSlash(file)
	parentPath := filepath.Dir(file)
	parentPathDirs := strings.Split(parentPath, string(os.PathSeparator))

	for _, pattern := range matchPatterns {
		// Skip evaluation if this is an inclusion and the filename
		// already matched the pattern, or it's an exclusion and it has
		// not matched the pattern yet.
		if pattern.Exclusion != matched {
			continue
		}

		match, err := pattern.Match(file)
		if err != nil {
			return false, err
		}

		if !match && parentPath != "." {
			// Check to see if the pattern matches one of our parent dirs.
			if len(pattern.Dirs) <= len(parentPathDirs) {
				match, _ = pattern.Match(strings.Join(parentPathDirs[:len(pattern.Dirs)], string(os.PathSeparator)))
			}
		}

		if match {
			matched = !pattern.Exclusion
		}
	}

	return matched, nil
}
