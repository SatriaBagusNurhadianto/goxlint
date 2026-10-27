package config

import (
	"path"
	"strings"
)

func matchAny(patterns []string, rel string) bool {
	for _, p := range patterns {
		if Match(p, rel) {
			return true
		}
	}
	return false
}

// Match reports whether the slash-separated path rel matches a glob pattern.
// "**" matches any number of path segments. A pattern without a slash matches
// the file name at any depth. A trailing "/**" also matches the directory itself.
func Match(pattern, rel string) bool {
	pattern = strings.TrimPrefix(pattern, "./")
	if !strings.Contains(pattern, "/") {
		pattern = "**/" + pattern
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(strings.TrimSuffix(rel, "/"), "/"))
}

func matchSegments(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			rest := pat[1:]
			for i := 0; i <= len(segs); i++ {
				if matchSegments(rest, segs[i:]) {
					return true
				}
			}
			return len(rest) == 0
		}
		if len(segs) == 0 {
			return false
		}
		// A malformed pattern reports ErrBadPattern and simply matches nothing.
		if ok, _ := path.Match(pat[0], segs[0]); !ok {
			return false
		}
		pat, segs = pat[1:], segs[1:]
	}
	return len(segs) == 0
}
