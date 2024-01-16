package patternmatcher

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/scanner"
	"unicode/utf8"
)

// escapeBytes is a bitmap used to check whether a character should be escaped when creating the regex.
var escapeBytes [8]byte

// shouldEscape reports whether a rune should be escaped as part of the regex.
//
// This only includes characters that require escaping in regex but are also NOT valid filepath pattern characters.
// Additionally, '\' is not excluded because there is specific logic to properly handle this, as it's a path separator
// on Windows.
//
// Adapted from regexp::QuoteMeta in go stdlib.
// See https://cs.opensource.google/go/go/+/refs/tags/go1.17.2:src/regexp/regexp.go;l=703-715;drc=refs%2Ftags%2Fgo1.17.2
func shouldEscape(b rune) bool {
	return b < utf8.RuneSelf && escapeBytes[b%8]&(1<<(b/8)) != 0
}

func init() {
	for _, b := range []byte(`.+()|{}$`) {
		escapeBytes[b%8] |= 1 << (b / 8)
	}
}

// MatchesUsingParentResults returns true if "file" matches any of the patterns
// and isn't excluded by any of the subsequent patterns. The functionality is
// the same as Matches, but as an optimization, the caller passes in
// intermediate results from matching the parent directory.
//
// parentMatched tracks the results of matching a file against a set of patterns.
// Position in this array corresponds to the position in the patterns.
// This is used as state for children of parent directories to avoid re-checking patterns.
//
// The "file" argument should be a slash-delimited path.
func MatchesUsingParentResults(patterns []*Pattern, file string, parentMatched []bool) (bool, []bool, error) {
	if len(parentMatched) != 0 && len(parentMatched) != len(patterns) {
		return false, nil, errors.New("wrong number of values in parentMatched")
	}

	file = filepath.FromSlash(file)
	matched := false

	matchInfo := make([]bool, len(patterns))
	for i, pattern := range patterns {
		match := false
		// If the parent matched this pattern, we don't need to recheck.
		if len(parentMatched) != 0 {
			match = parentMatched[i]
		}

		if !match {
			// Skip evaluation if this is an inclusion and the filename
			// already matched the pattern, or it's an exclusion and it has
			// not matched the pattern yet.
			if pattern.Exclusion != matched {
				continue
			}

			match = pattern.Match(file)

			// If the zero value of MatchInfo was passed in, we don't have
			// any information about the parent dir's match results, and we
			// apply the same logic as MatchesOrParentMatches.
			if !match && len(parentMatched) == 0 {
				if parentPath := filepath.Dir(file); parentPath != "." {
					parentPathDirs := strings.Split(parentPath, string(os.PathSeparator))
					// Check to see if the pattern matches one of our parent dirs.
					for i := range parentPathDirs {
						match = pattern.Match(strings.Join(parentPathDirs[:i+1], string(os.PathSeparator)))
						if match {
							break
						}
					}
				}
			}
		}
		matchInfo[i] = match

		if match {
			matched = !pattern.Exclusion
		}
	}
	return matched, matchInfo, nil
}

// MatchesOrParentMatches returns true if file matches any of the patterns
// and isn't excluded by any of the subsequent patterns.
//
// The "file" argument should be a slash-delimited path.
func MatchesOrParentMatches(patterns []*Pattern, file string) (bool, error) {
	file = filepath.Clean(file)

	if file == "." {
		// Don't let them exclude everything, kind of silly.
		return false, nil
	}

	matched := false
	file = filepath.FromSlash(file)
	parentPath := filepath.Dir(file)
	parentPathDirs := strings.Split(parentPath, string(os.PathSeparator))

	for _, pattern := range patterns {
		// Skip evaluation if this is an inclusion and the filename
		// already matched the pattern, or it's an exclusion and it has
		// not matched the pattern yet.
		if pattern.Exclusion != matched {
			continue
		}

		match := pattern.Match(file)
		if !match && parentPath != "." {
			// Check to see if the pattern matches one of our parent dirs.
			for i := range parentPathDirs {
				match = pattern.Match(strings.Join(parentPathDirs[:i+1], string(os.PathSeparator)))
				if match {
					break
				}
			}
		}

		if match {
			matched = !pattern.Exclusion
		}
	}

	return matched, nil
}

// NewPatterns creates patterns that match against paths.
func NewPatterns(patterns []string) ([]*Pattern, error) {
	matchPatters := make([]*Pattern, 0, len(patterns))
	for _, p := range patterns {
		// Eliminate leading and trailing whitespace.
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)

		// Do some syntax checking on the pattern.
		// filepath's Match() has some really weird rules that are inconsistent
		// so instead of trying to dup their logic, just call Match() for its
		// error state and if there is an error in the pattern return it.
		// If this becomes an issue we can remove this since its really only
		// needed in the error (syntax) case - which isn't really critical.
		if _, err := filepath.Match(p, "."); err != nil {
			return nil, err
		}

		newp, err := NewPattern(p)
		if err != nil {
			return nil, err
		}
		matchPatters = append(matchPatters, newp)
	}
	return matchPatters, nil
}

type MatchType int

const (
	UnknownMatch MatchType = iota
	ExactMatch
	PrefixMatch
	SuffixMatch
	RegexpMatch
)

// Pattern defines a single regexp used to filter file paths.
type Pattern struct {
	MatchType      MatchType
	CleanedPattern string
	Dirs           []string
	Regexp         *regexp.Regexp
	// Exclusion returns true if this pattern defines Exclusion
	Exclusion bool
}

func NewPattern(pattern string) (*Pattern, error) {
	var exclusion bool
	if pattern[0] == '!' {
		if len(pattern) == 1 {
			return nil, errors.New("illegal exclusion pattern: \"!\"")
		}
		exclusion = true
		pattern = pattern[1:]
	}

	matchType, regexp, err := Compile(pattern)
	if err != nil {
		return nil, err
	}
	p := &Pattern{
		MatchType:      matchType,
		CleanedPattern: pattern,
		Dirs:           strings.Split(pattern, string(os.PathSeparator)),
		Regexp:         regexp,
		Exclusion:      exclusion,
	}

	return p, nil
}

func (p *Pattern) Match(path string) bool {
	switch p.MatchType {
	case ExactMatch:
		return path == p.CleanedPattern
	case PrefixMatch:
		// strip trailing **
		return strings.HasPrefix(path, p.CleanedPattern[:len(p.CleanedPattern)-2])
	case SuffixMatch:
		// strip leading **
		suffix := p.CleanedPattern[2:]
		if strings.HasSuffix(path, suffix) {
			return true
		}
		// **/foo matches "foo"
		return suffix[0] == os.PathSeparator && path == suffix[1:]
	case RegexpMatch:
		return p.Regexp.MatchString(path)
	}

	return false
}

func Compile(pattern string) (MatchType, *regexp.Regexp, error) {
	pathSeparator := string(os.PathSeparator)
	regStr := "^"
	// Go through the pattern and convert it to a regexp.
	// We use a scanner so we can support utf-8 chars.
	var scan scanner.Scanner
	scan.Init(strings.NewReader(pattern))

	escapedPathSeparator := pathSeparator
	if pathSeparator == `\` {
		escapedPathSeparator += `\`
	}

	matchType := ExactMatch
	for i := 0; scan.Peek() != scanner.EOF; i++ {
		ch := scan.Next()

		if ch == '*' {
			if scan.Peek() == '*' {
				// is some flavor of "**"
				scan.Next()

				// Treat **/ as ** so eat the "/"
				if string(scan.Peek()) == pathSeparator {
					scan.Next()
				}

				if scan.Peek() == scanner.EOF {
					// is "**EOF" - to align with .gitignore just accept all
					if matchType == ExactMatch {
						matchType = PrefixMatch
					} else {
						regStr += ".*"
						matchType = RegexpMatch
					}
				} else {
					// is "**"
					// Note that this allows for any # of /'s (even 0) because
					// the .* will eat everything, even /'s
					regStr += "(.*" + escapedPathSeparator + ")?"
					matchType = RegexpMatch
				}

				if i == 0 {
					matchType = SuffixMatch
				}
			} else {
				// is "*" so map it to anything but "/"
				regStr += "[^" + escapedPathSeparator + "]*"
				matchType = RegexpMatch
			}
		} else if ch == '?' {
			// "?" is any char except "/"
			regStr += "[^" + escapedPathSeparator + "]"
			matchType = RegexpMatch
		} else if shouldEscape(ch) {
			// Escape some regexp special chars that have no meaning
			// in golang's filepath.Match
			regStr += `\` + string(ch)
		} else if ch == '\\' {
			// escape next char. Note that a trailing \ in the pattern
			// will be left alone (but need to escape it)
			if pathSeparator == `\` {
				// On windows map "\" to "\\", meaning an escaped backslash,
				// and then just continue because filepath.Match on
				// Windows doesn't allow escaping at all
				regStr += escapedPathSeparator
				continue
			}
			if scan.Peek() != scanner.EOF {
				regStr += `\` + string(scan.Next())
				matchType = RegexpMatch
			} else {
				regStr += `\`
			}
		} else if ch == '[' || ch == ']' {
			regStr += string(ch)
			matchType = RegexpMatch
		} else {
			regStr += string(ch)
		}
	}

	if matchType != RegexpMatch {
		return matchType, nil, nil
	}

	regStr += "$"

	re, err := regexp.Compile(regStr)
	if err != nil {
		return UnknownMatch, nil, err
	}

	return matchType, re, nil
}
