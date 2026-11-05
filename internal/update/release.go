package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// DefaultRepo is the GitHub repository Catsitter ships from.
const DefaultRepo = "xanderwasserman/catsitter"

var repoName = regexp.MustCompile(`^[A-Za-z0-9-]+/[A-Za-z0-9._-]+$`)

// ErrNotModified means the latest release has not changed since the given ETag.
var ErrNotModified = errors.New("release not modified")

// Release is one published Catsitter release.
type Release struct {
	Version Version
	Tag     string
	Notes   string
	URL     string
}

// Feed finds the latest release and where its files are.
type Feed struct {
	// Repo is "owner/name" on GitHub.
	Repo string
	// Base replaces GitHub with a mirror laid out like the installers expect:
	// Base/latest and Base/download/<tag>/<file>.
	Base      string
	Client    *http.Client
	UserAgent string
}

// FeedFromEnv honours the same CATSITTER_REPO and CATSITTER_RELEASE_BASE
// overrides as the install scripts. A repo that is not "owner/name" is ignored.
func FeedFromEnv(userAgent string) Feed {
	repo := os.Getenv("CATSITTER_REPO")
	if !repoName.MatchString(repo) {
		repo = DefaultRepo
	}
	return Feed{
		Repo:      repo,
		Base:      strings.TrimRight(os.Getenv("CATSITTER_RELEASE_BASE"), "/"),
		Client:    &http.Client{},
		UserAgent: userAgent,
	}
}

// PageURL is raw when it is an https://github.com page inside repo, and
// otherwise the release page for tag, so a feed cannot hand the browser
// another site, a file or a program.
func PageURL(repo, raw, tag string) string {
	fallback := "https://github.com/" + repo + "/releases/tag/" + url.PathEscape(tag)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil || u.Opaque != "" {
		return fallback
	}
	if !strings.HasPrefix(u.Path, "/"+repo+"/") || strings.Contains(u.Path, "..") || strings.ContainsAny(u.Path, "\\") {
		return fallback
	}
	return u.String()
}

// DownloadURL is where a release file lives.
func (f Feed) DownloadURL(tag, name string) string {
	if f.Base != "" {
		return f.Base + "/download/" + tag + "/" + name
	}
	return "https://github.com/" + f.Repo + "/releases/download/" + tag + "/" + name
}

func (f Feed) latestURL() string {
	if f.Base != "" {
		return f.Base + "/latest"
	}
	return "https://api.github.com/repos/" + f.Repo + "/releases/latest"
}

// Latest fetches the newest published release. With a non-empty etag the
// request is conditional, which GitHub does not count against the rate limit;
// an unchanged release returns ErrNotModified. The returned string is the new ETag.
func (f Feed) Latest(ctx context.Context, etag string) (Release, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.latestURL(), nil)
	if err != nil {
		return Release{}, "", err
	}
	if err := httpsOnly(req.URL); err != nil {
		return Release{}, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if f.UserAgent != "" {
		req.Header.Set("User-Agent", f.UserAgent)
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := f.client().Do(req)
	if err != nil {
		return Release{}, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return Release{}, etag, ErrNotModified
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, "", fmt.Errorf("latest release: %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Release{}, "", err
	}
	rel, err := parseLatest(body)
	if err != nil {
		return Release{}, "", err
	}
	rel.URL = PageURL(f.Repo, rel.URL, rel.Tag)
	return rel, resp.Header.Get("ETag"), nil
}

// parseLatest reads a GitHub release object, or a bare tag as a mirror serves it.
func parseLatest(body []byte) (Release, error) {
	var gh struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	tag := strings.TrimSpace(string(body))
	if strings.HasPrefix(tag, "{") {
		if err := json.Unmarshal(body, &gh); err != nil {
			return Release{}, fmt.Errorf("latest release: %w", err)
		}
		tag = gh.TagName
	}
	v, ok := ParseVersion(tag)
	if !ok {
		return Release{}, fmt.Errorf("latest release has an unexpected tag %q", tag)
	}
	return Release{Version: v, Tag: "v" + v.String(), Notes: gh.Body, URL: gh.HTMLURL}, nil
}

// client is the configured client, made to refuse redirects off HTTPS.
func (f Feed) client() *http.Client {
	c := http.Client{}
	if f.Client != nil {
		c = *f.Client
	}
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return httpsOnly(req.URL)
	}
	return &c
}

// httpsOnly refuses any URL that is not HTTPS, including a mirror base.
func httpsOnly(u *url.URL) error {
	if u.Scheme != "https" {
		return fmt.Errorf("refusing %s: updates need HTTPS", u.Redacted())
	}
	return nil
}
