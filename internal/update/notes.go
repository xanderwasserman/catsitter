package update

import (
	"regexp"
	"strings"
)

var (
	mdLink     = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)
	mdEmphasis = regexp.MustCompile("(\\*\\*|__|`)")
	mdBullet   = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+`)
	mdComment  = regexp.MustCompile(`(?s)<!--.*?-->`)
	changelog  = regexp.MustCompile(`(?i)^full changelog\b`)
)

// Highlights turns Markdown release notes into at most max plain lines.
// Headings, blank lines, horizontal rules and GitHub's "Full Changelog"
// link are dropped.
func Highlights(body string, max int) []string {
	body = mdComment.ReplaceAllString(body, "")
	var out []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.Trim(line, "-*_= ") == "" {
			continue
		}
		line = mdBullet.ReplaceAllString(line, "")
		line = mdLink.ReplaceAllString(line, "$1")
		line = mdEmphasis.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)
		if line == "" || changelog.MatchString(line) {
			continue
		}
		if len(out) == max {
			break
		}
		out = append(out, line)
	}
	return out
}
