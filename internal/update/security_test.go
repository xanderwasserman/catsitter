package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

const evilTag = "v9.9.9-../../../../home/x/.config"

func TestUnsafeTagsDoNotParse(t *testing.T) {
	for _, tag := range []string{
		evilTag,
		"v1.2.3-a/b",
		`v1.2.3-a\b`,
		"v1.2.3-a..b",
		"v1.2.3-.a",
		"v1.2.3-a.",
		"v1.2.3-a b",
		"v1.2.3-%2e%2e",
		"v1.2.3+../x",
		"v1.2.3+",
		"v+1.2.3",
		"v1.+2.3",
		"v1.2.3-rc.1;rm",
	} {
		if v, ok := ParseVersion(tag); ok {
			t.Errorf("%q parsed as %+v", tag, v)
		}
	}
	for _, tag := range []string{"v1.2.3-rc.1", "v1.2.3-x-y.0a", "v1.2.3+build.5-a"} {
		if _, ok := ParseVersion(tag); !ok {
			t.Errorf("%q should parse", tag)
		}
	}
}

func TestFeedRejectsAnUnsafeTag(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"` + evilTag + `"}`))
	}))
	defer srv.Close()
	if rel, _, err := (Feed{Repo: "o/r", Base: srv.URL, Client: srv.Client()}).Latest(context.Background(), ""); err == nil {
		t.Fatalf("unsafe tag accepted: %+v", rel)
	}
}

// isolateTemp points the temp directory at a fresh folder and returns it.
func isolateTemp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	tmp := filepath.Join(root, "tmp")
	if err := os.Mkdir(tmp, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	return tmp
}

func TestStageRefusesAnUnsafeTag(t *testing.T) {
	tmp := isolateTemp(t)
	var hits atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer srv.Close()
	// Built by hand, as a buggy or hostile Releases could, since ParseVersion refuses it.
	rel := Release{Version: Version{9, 9, 9, "../../../../home/x/.config"}, Tag: evilTag}
	if dir, err := (Feed{Base: srv.URL, Client: srv.Client()}).Stage(context.Background(), rel, "linux", "amd64", nil); err == nil {
		t.Fatalf("staged into %s", dir)
	}
	if hits.Load() != 0 {
		t.Error("an unsafe tag reached the network")
	}
	entries, _ := os.ReadDir(tmp)
	if len(entries) != 0 {
		t.Errorf("temp folder not empty: %v", entries)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(tmp), "home")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a folder was created outside the temp folder")
	}

	mismatched := Release{Version: Version{9, 9, 9, ""}, Tag: "v9.9.9/../../x"}
	if _, err := (Feed{Base: srv.URL, Client: srv.Client()}).Stage(context.Background(), mismatched, "linux", "amd64", nil); err == nil {
		t.Error("a tag that does not match its version was staged")
	}
}

func TestOfferNeverOffersAPreReleaseToARelease(t *testing.T) {
	if Offer("0.1.3", mustVersion(t, "0.1.4-rc.1")) {
		t.Error("a release build was offered a pre-release")
	}
	if !Offer("0.1.3", mustVersion(t, "0.1.4")) {
		t.Error("a release build should be offered the next release")
	}
}

func TestPageURL(t *testing.T) {
	const repo = "o/r"
	const fallback = "https://github.com/o/r/releases/tag/v0.1.4"
	kept := []string{
		"https://github.com/o/r/releases/tag/v0.1.4",
		"https://github.com/o/r/releases",
	}
	for _, raw := range kept {
		if got := PageURL(repo, raw, "v0.1.4"); got != raw {
			t.Errorf("%q became %q", raw, got)
		}
	}
	rebuilt := []string{
		"",
		"file:///C:/Windows/System32/calc.exe",
		`C:\Windows\System32\calc.exe`,
		`\\server\share\run.exe`,
		"/usr/bin/xcalc",
		"javascript:alert(1)",
		"http://github.com/o/r/releases/tag/v0.1.4",
		"https://evil.example/o/r/releases/tag/v0.1.4",
		"https://github.com.evil.example/o/r/releases",
		"https://github.com/other/repo/releases/tag/v0.1.4",
		"https://github.com/o/rx/releases",
		"https://github.com/o/r/../../evil/x",
		"https://github.com/o/r/%2e%2e/%2e%2e/evil",
		"https://user@github.com/o/r/releases",
		"https://github.com:8443/o/r/releases",
		"https://github.com/o/r",
	}
	for _, raw := range rebuilt {
		if got := PageURL(repo, raw, "v0.1.4"); got != fallback {
			t.Errorf("%q kept as %q", raw, got)
		}
	}
}

func TestFeedRebuildsAForeignReleaseURL(t *testing.T) {
	for _, html := range []string{"file:///C:/Windows/System32/calc.exe", "https://evil.example/o/r/releases/tag/v0.1.4", "https://github.com/other/repo/releases/tag/v0.1.4"} {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"tag_name":"v0.1.4","html_url":` + strconv.Quote(html) + `}`))
		}))
		rel, _, err := (Feed{Repo: "o/r", Base: srv.URL, Client: srv.Client()}).Latest(context.Background(), "")
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if rel.URL != "https://github.com/o/r/releases/tag/v0.1.4" {
			t.Errorf("%q kept as %q", html, rel.URL)
		}
	}
}

func TestServiceRebuildsForeignReleaseURLs(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "update.json")}
	store.Save(Record{Latest: &SavedRelease{Tag: "v0.1.4", URL: "file:///C:/Windows/System32/calc.exe"}})
	s := NewService(Options{Version: "0.1.3", Repo: "o/r", Store: store, Releases: &fakeReleases{}})
	if got := s.Status().Latest.URL; got != "https://github.com/o/r/releases/tag/v0.1.4" {
		t.Errorf("saved URL kept as %q", got)
	}

	evil := release(t, "0.1.5")
	evil.URL = `C:\Windows\System32\calc.exe`
	s = NewService(Options{Version: "0.1.3", Repo: "o/r", Store: Store{Path: filepath.Join(t.TempDir(), "u.json")}, Releases: &fakeReleases{rel: evil}})
	s.check(context.Background(), true)
	if got := s.Status().Latest.URL; got != "https://github.com/o/r/releases/tag/v0.1.5" {
		t.Errorf("fresh URL kept as %q", got)
	}
}

func TestPlainHTTPIsRefused(t *testing.T) {
	var hits atomic.Int32
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write([]byte("v0.1.4"))
	}))
	defer plain.Close()

	if _, _, err := (Feed{Repo: "o/r", Base: plain.URL}).Latest(context.Background(), ""); err == nil {
		t.Error("an http:// base was used")
	}
	rel := release(t, "0.1.4")
	if _, err := (Feed{Base: plain.URL}).Stage(context.Background(), rel, "linux", "amd64", nil); err == nil {
		t.Error("an http:// download was used")
	}

	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+r.URL.Path, http.StatusFound)
	}))
	defer redirect.Close()
	f := Feed{Repo: "o/r", Base: redirect.URL, Client: redirect.Client()}
	if _, _, err := f.Latest(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Errorf("a redirect to http:// was followed: %v", err)
	}
	if _, err := f.Stage(context.Background(), rel, "linux", "amd64", nil); err == nil {
		t.Error("a download redirected to http:// was used")
	}
	if hits.Load() != 0 {
		t.Errorf("the plain server was reached %d times", hits.Load())
	}
}

func TestDownloadsAreCapped(t *testing.T) {
	body := strings.Repeat("x", 1000)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sized" {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		} else {
			w.(http.Flusher).Flush()
		}
		w.Write([]byte(body))
	}))
	defer srv.Close()
	f := Feed{Client: srv.Client()}
	dir := t.TempDir()
	for _, path := range []string{"/sized", "/streamed"} {
		if err := f.download(context.Background(), srv.URL+path, filepath.Join(dir, "f"), 100, nil, nil); err == nil {
			t.Errorf("%s: 1000 bytes passed a 100-byte cap", path)
		}
	}
	if err := f.download(context.Background(), srv.URL+"/sized", filepath.Join(dir, "f"), 1000, nil, nil); err != nil {
		t.Errorf("a file at the cap was refused: %v", err)
	}
}
