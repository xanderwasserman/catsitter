package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func mustVersion(t *testing.T, s string) Version {
	t.Helper()
	v, ok := ParseVersion(s)
	if !ok {
		t.Fatalf("parse %q", s)
	}
	return v
}

func TestParseVersion(t *testing.T) {
	good := map[string]Version{
		"1.2.3":           {1, 2, 3, ""},
		"v0.1.2":          {0, 1, 2, ""},
		" 0.1.3-rc.1 ":    {0, 1, 3, "rc.1"},
		"0.1.3-next":      {0, 1, 3, "next"},
		"1.0.0+build.5":   {1, 0, 0, ""},
		"10.20.30-beta.2": {10, 20, 30, "beta.2"},
	}
	for in, want := range good {
		if got, ok := ParseVersion(in); !ok || got != want {
			t.Errorf("%q: %+v %v, want %+v", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "dev", "1.2", "1.2.3.4", "1.x.3", "01.2.3", "1.2.3-", "-1.2.3"} {
		if v, ok := ParseVersion(in); ok {
			t.Errorf("%q parsed as %+v", in, v)
		}
	}
}

func TestVersionOrder(t *testing.T) {
	// Each version has lower precedence than the next.
	order := []string{"0.1.2", "0.1.3-alpha", "0.1.3-alpha.1", "0.1.3-alpha.beta", "0.1.3-beta.2", "0.1.3-beta.11", "0.1.3-rc.1", "0.1.3", "0.1.10", "0.2.0", "1.0.0"}
	for i := 0; i+1 < len(order); i++ {
		a, b := mustVersion(t, order[i]), mustVersion(t, order[i+1])
		if !a.Less(b) || b.Less(a) {
			t.Errorf("%s should come before %s", a, b)
		}
	}
	if v := mustVersion(t, "0.1.2"); v.Less(v) {
		t.Error("a version is not less than itself")
	}
}

func TestIsRelease(t *testing.T) {
	for in, want := range map[string]bool{"0.1.2": true, "v1.0.0": true, "dev": false, "0.1.3-next": false, "": false} {
		if got := IsRelease(in); got != want {
			t.Errorf("IsRelease(%q) = %v", in, got)
		}
	}
}

func TestOffer(t *testing.T) {
	latest := mustVersion(t, "0.1.3")
	cases := map[string]bool{
		"0.1.2":      true,
		"0.1.3":      false,
		"0.2.0":      false,
		"dev":        false,
		"0.1.2-next": false,
		"":           false,
	}
	for cur, want := range cases {
		if got := Offer(cur, latest); got != want {
			t.Errorf("Offer(%q, 0.1.3) = %v", cur, got)
		}
	}
}

func TestDue(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	if !Due(time.Time{}, now, time.Hour) {
		t.Error("never checked should be due")
	}
	if Due(now.Add(-30*time.Minute), now, time.Hour) {
		t.Error("checked recently should not be due")
	}
	if !Due(now.Add(-time.Hour), now, time.Hour) {
		t.Error("an hour later should be due")
	}
	if !Due(now.Add(time.Hour), now, time.Hour) {
		t.Error("a check in the future means the clock moved, so check")
	}
}

func TestArchiveName(t *testing.T) {
	v := mustVersion(t, "0.1.3")
	cases := map[[2]string]string{
		{"linux", "amd64"}:   "catsitter_0.1.3_linux_amd64.tar.gz",
		{"linux", "arm64"}:   "catsitter_0.1.3_linux_arm64.tar.gz",
		{"darwin", "arm64"}:  "catsitter_0.1.3_darwin_arm64.zip",
		{"darwin", "amd64"}:  "catsitter_0.1.3_darwin_amd64.zip",
		{"windows", "amd64"}: "catsitter_0.1.3_windows_amd64.zip",
		{"windows", "arm64"}: "catsitter_0.1.3_windows_arm64.zip",
	}
	for in, want := range cases {
		if got, err := ArchiveName(v, in[0], in[1]); err != nil || got != want {
			t.Errorf("%v: %q %v, want %q", in, got, err, want)
		}
	}
	for _, in := range [][2]string{{"linux", "386"}, {"freebsd", "amd64"}, {"windows", "arm"}} {
		if got, err := ArchiveName(v, in[0], in[1]); err == nil {
			t.Errorf("%v: got %q, want an error", in, got)
		}
	}
}

func TestInstallPath(t *testing.T) {
	h := Home{Dir: "/home/cat", LocalAppData: `C:\Users\cat\AppData\Local`}
	if got := InstallPath("darwin", h); got != filepath.Join("/home/cat", "Applications", "Catsitter.app", "Contents", "MacOS", "catsitter") {
		t.Errorf("darwin: %s", got)
	}
	if got := InstallPath("linux", h); got != filepath.Join("/home/cat", ".local", "bin", "catsitter") {
		t.Errorf("linux: %s", got)
	}
	if got := InstallPath("windows", h); got != filepath.Join(h.LocalAppData, "Programs", "Catsitter", "catsitter.exe") {
		t.Errorf("windows: %s", got)
	}
	if got := InstallPath("windows", Home{Dir: "x"}); got != "" {
		t.Errorf("windows without LOCALAPPDATA: %q", got)
	}
}

func TestSamePath(t *testing.T) {
	if !samePath(`C:\Users\Cat\catsitter.exe`, `c:\users\cat\CATSITTER.EXE`, "windows") {
		t.Error("windows paths ignore case")
	}
	if samePath("/home/Cat/x", "/home/cat/x", "linux") {
		t.Error("linux paths keep case")
	}
}

func TestInstalledFollowsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	home := t.TempDir()
	h := Home{Dir: home}
	want := InstallPath(runtime.GOOS, h)
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, "link")
	if err := os.Symlink(want, link); err != nil {
		t.Fatal(err)
	}
	if !Installed(link, h) {
		t.Error("a symlink to the installed copy counts as installed")
	}
	if Installed(filepath.Join(home, "elsewhere"), h) {
		t.Error("another path is not the installed copy")
	}
}

func sum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestChecksums(t *testing.T) {
	archive := []byte("pretend archive")
	text := "# not a checksum line\n" +
		strings.ToUpper(sum(archive)) + "  catsitter_0.1.3_linux_amd64.tar.gz\n" +
		sum([]byte("other")) + " *catsitter_0.1.3_windows_amd64.zip\n" +
		"deadbeef  short.zip\n"
	sums, err := ParseChecksums(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	if len(sums) != 2 {
		t.Fatalf("parsed %v", sums)
	}
	if _, ok := sums["catsitter_0.1.3_windows_amd64.zip"]; !ok {
		t.Error("binary-mode star is stripped from the name")
	}

	h, err := sums.Expect("catsitter_0.1.3_linux_amd64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	h.Write(archive)
	if err := h.Verify(); err != nil {
		t.Errorf("matching bytes: %v", err)
	}

	h, _ = sums.Expect("catsitter_0.1.3_linux_amd64.tar.gz")
	h.Write([]byte("tampered"))
	if err := h.Verify(); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("tampered bytes: %v", err)
	}

	if _, err := sums.Expect("catsitter_0.1.3_darwin_arm64.zip"); err == nil {
		t.Error("a file missing from checksums.txt is an error")
	}
}

func TestHighlights(t *testing.T) {
	body := "<!-- hidden -->\n## What's new\n\n- **Update** from the [menu](https://x)\n* An `About` panel\n1. Fewer naps\n\n---\n**Full Changelog**: https://github.com/a/b/compare/v1...v2\n"
	got := Highlights(body, 5)
	want := []string{"Update from the menu", "An About panel", "Fewer naps"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q", got)
	}
	if got := Highlights(body, 2); len(got) != 2 {
		t.Errorf("max 2: %q", got)
	}
	if got := Highlights("\n", 5); len(got) != 0 {
		t.Errorf("empty notes: %q", got)
	}
}

func TestParseLatest(t *testing.T) {
	rel, err := parseLatest([]byte(`{"tag_name":"v0.1.3","body":"notes","html_url":"https://example/r"}`))
	if err != nil || rel.Tag != "v0.1.3" || rel.Notes != "notes" || rel.URL != "https://example/r" {
		t.Errorf("json: %+v %v", rel, err)
	}
	rel, err = parseLatest([]byte("0.1.3\n"))
	if err != nil || rel.Tag != "v0.1.3" {
		t.Errorf("bare tag: %+v %v", rel, err)
	}
	if _, err := parseLatest([]byte(`{"tag_name":"nightly"}`)); err == nil {
		t.Error("a tag that is not a version is an error")
	}
}

func TestFeedLatestIsConditional(t *testing.T) {
	var agent, match string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent, match = r.UserAgent(), r.Header.Get("If-None-Match")
		if match == `"v3"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v3"`)
		w.Write([]byte(`{"tag_name":"v0.1.3","body":"- hi"}`))
	}))
	defer srv.Close()
	f := Feed{Repo: "o/r", Base: srv.URL, UserAgent: "catsitter/test"}

	rel, etag, err := f.Latest(context.Background(), "")
	if err != nil || rel.Version.String() != "0.1.3" || etag != `"v3"` {
		t.Fatalf("%+v %q %v", rel, etag, err)
	}
	if agent != "catsitter/test" {
		t.Errorf("user agent %q", agent)
	}
	if rel.URL != "https://github.com/o/r/releases/tag/v0.1.3" {
		t.Errorf("default release URL %q", rel.URL)
	}
	if _, _, err := f.Latest(context.Background(), `"v3"`); !errors.Is(err, ErrNotModified) {
		t.Errorf("same ETag: %v", err)
	}
	if match != `"v3"` {
		t.Errorf("If-None-Match %q", match)
	}
}

func TestFeedURLs(t *testing.T) {
	gh := Feed{Repo: "o/r"}
	if got := gh.latestURL(); got != "https://api.github.com/repos/o/r/releases/latest" {
		t.Error(got)
	}
	if got := gh.DownloadURL("v1.0.0", "checksums.txt"); got != "https://github.com/o/r/releases/download/v1.0.0/checksums.txt" {
		t.Error(got)
	}
	mirror := Feed{Repo: "o/r", Base: "http://mirror"}
	if got := mirror.DownloadURL("v1.0.0", "a.zip"); got != "http://mirror/download/v1.0.0/a.zip" {
		t.Error(got)
	}
}

// releaseServer serves one release in the mirror layout.
func releaseServer(t *testing.T, archive []byte, sums string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/download/v0.1.3/checksums.txt", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(sums)) })
	mux.HandleFunc("/download/v0.1.3/catsitter_0.1.3_linux_amd64.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(archive)))
		w.Write(archive)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestStageVerifiesTheArchive(t *testing.T) {
	archive := bytes.Repeat([]byte("cat"), 10000)
	rel := Release{Version: mustVersion(t, "0.1.3"), Tag: "v0.1.3"}
	name := "catsitter_0.1.3_linux_amd64.tar.gz"

	srv := releaseServer(t, archive, sum(archive)+"  "+name+"\n")
	var last, total int64
	dir, err := Feed{Base: srv.URL}.Stage(context.Background(), rel, "linux", "amd64", func(d, t int64) { last, total = d, t })
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	got, err := os.ReadFile(filepath.Join(dir, "download", "v0.1.3", name))
	if err != nil || !bytes.Equal(got, archive) {
		t.Fatalf("staged archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "download", "v0.1.3", "checksums.txt")); err != nil {
		t.Error("checksums.txt is staged for the installer")
	}
	if last != int64(len(archive)) || total != int64(len(archive)) {
		t.Errorf("progress %d/%d", last, total)
	}

	bad := releaseServer(t, archive, sum([]byte("other"))+"  "+name+"\n")
	if dir, err := (Feed{Base: bad.URL}).Stage(context.Background(), rel, "linux", "amd64", nil); err == nil {
		os.RemoveAll(dir)
		t.Fatal("a mismatched archive was staged")
	} else if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("mismatch error: %v", err)
	}

	missing := releaseServer(t, archive, sum(archive)+"  something_else.zip\n")
	if _, err := (Feed{Base: missing.URL}).Stage(context.Background(), rel, "linux", "amd64", nil); err == nil {
		t.Error("an archive without a checksum was staged")
	}
}

func TestLastLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	os.WriteFile(path, []byte("Downloading Catsitter 0.1.3\r\ncatsitter install: checksum mismatch for x.zip\r\n\r\n"), 0o600)
	if got := lastLine(path); got != "checksum mismatch for x.zip" {
		t.Errorf("%q", got)
	}
}

func TestStoreRoundTrip(t *testing.T) {
	s := Store{Path: filepath.Join(t.TempDir(), "sub", "update.json")}
	if r := s.Load(); r.Latest != nil || r.Pending != nil {
		t.Fatalf("missing file: %+v", r)
	}
	want := Record{ETag: `"x"`, Latest: &SavedRelease{Tag: "v0.1.3", Notes: "n"}, Pending: &Pending{From: "0.1.2", To: "0.1.3"}}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	got := s.Load()
	if got.ETag != want.ETag || got.Latest.Tag != "v0.1.3" || got.Pending.To != "0.1.3" {
		t.Errorf("%+v", got)
	}
	os.WriteFile(s.Path, []byte("{broken"), 0o600)
	if r := s.Load(); r.ETag != "" {
		t.Error("a damaged file reads as empty")
	}
}

func TestInstallerCommandHandsOverTheDownload(t *testing.T) {
	dir := t.TempDir()
	cmd, err := Installer{Script: []byte("echo hi\n")}.command(dir, mustVersion(t, "0.1.3"))
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Dir != filepath.Dir(dir) {
		t.Errorf("dir %q", cmd.Dir)
	}
	if cmd.Env[len(cmd.Env)-1] != "CATSITTER_INSTALL_FROM="+dir {
		t.Errorf("env ends with %q", cmd.Env[len(cmd.Env)-1])
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, filepath.Join(dir, scriptName)) || !strings.Contains(args, "0.1.3") {
		t.Errorf("args %q", args)
	}
	if b, err := os.ReadFile(filepath.Join(dir, scriptName)); err != nil || string(b) != "echo hi\n" {
		t.Errorf("script %q %v", b, err)
	}
	if _, err := (Installer{}).command(dir, mustVersion(t, "0.1.3")); err == nil {
		t.Error("an empty script is an error")
	}
}

func TestLastLineSkipsPowerShellNoise(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	os.WriteFile(path, []byte("catsitter install: checksum mismatch for x.zip\r\nAt C:\\t\\install.ps1:70 char:5\r\n+ FullyQualifiedErrorId : catsitter install: checksum mismatch for x.zip\r\n"), 0o600)
	if got := lastLine(path); got != "checksum mismatch for x.zip" {
		t.Errorf("%q", got)
	}
	os.WriteFile(path, []byte("sh: 1: oops\n"), 0o600)
	if got := lastLine(path); got != "sh: 1: oops" {
		t.Errorf("untagged: %q", got)
	}
}

func TestOwnDownload(t *testing.T) {
	if !ownDownload(filepath.Join(os.TempDir(), "catsitter-update-123")) {
		t.Error("a staging folder is ours")
	}
	for _, dir := range []string{"", "/", filepath.Join(os.TempDir(), "other"), filepath.Join(os.TempDir(), "a", "catsitter-update-1"), "/home/cat/catsitter-update-1"} {
		if ownDownload(dir) {
			t.Errorf("%q is not ours", dir)
		}
	}
}
