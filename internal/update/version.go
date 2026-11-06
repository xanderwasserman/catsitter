// Package update checks GitHub Releases for a newer Catsitter and installs it.
package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a semantic version. Pre is the pre-release suffix without the dash.
type Version struct {
	Major, Minor, Patch int
	Pre                 string
}

// ParseVersion reads "1.2.3", "v1.2.3" or "1.2.3-rc.1". Build metadata after
// "+" is checked and then ignored. Pre-release and build identifiers may only
// hold letters, digits and hyphens, so a parsed version is safe in paths,
// URLs and arguments.
func ParseVersion(s string) (Version, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	s, build, hasBuild := strings.Cut(s, "+")
	if hasBuild && !identifiers(build) {
		return Version{}, false
	}
	core, pre, hasPre := strings.Cut(s, "-")
	if hasPre && !identifiers(pre) {
		return Version{}, false
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, false
	}
	var nums [3]int
	for i, p := range parts {
		if p == "" || strings.Trim(p, "0123456789") != "" || (len(p) > 1 && p[0] == '0') {
			return Version{}, false
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return Version{}, false
		}
		nums[i] = n
	}
	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2], Pre: pre}, true
}

// identifiers reports whether s is dot-separated, non-empty runs of
// [0-9A-Za-z-], as semver requires for pre-release and build parts.
func identifiers(s string) bool {
	for _, id := range strings.Split(s, ".") {
		if id == "" {
			return false
		}
		for _, r := range id {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '-') {
				return false
			}
		}
	}
	return true
}

// IsRelease reports whether s is a published release version, not a dev or
// snapshot build.
func IsRelease(s string) bool {
	v, ok := ParseVersion(s)
	return ok && v.Pre == ""
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Pre != "" {
		s += "-" + v.Pre
	}
	return s
}

// Less reports whether v has lower precedence than o under semver rules.
func (v Version) Less(o Version) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	if v.Patch != o.Patch {
		return v.Patch < o.Patch
	}
	return lessPre(v.Pre, o.Pre)
}

func lessPre(a, b string) bool {
	if a == b {
		return false
	}
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] == bs[i] {
			continue
		}
		an, aErr := strconv.Atoi(as[i])
		bn, bErr := strconv.Atoi(bs[i])
		switch {
		case aErr == nil && bErr == nil:
			return an < bn
		case aErr == nil:
			return true
		case bErr == nil:
			return false
		default:
			return as[i] < bs[i]
		}
	}
	return len(as) < len(bs)
}
