package update

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fakeReleases struct {
	mu       sync.Mutex
	rel      Release
	etag     string
	err      error
	stageErr error
	calls    int
	lastTag  string
	staged   chan struct{}
}

func (f *fakeReleases) Latest(_ context.Context, etag string) (Release, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.lastTag = etag
	if f.err != nil {
		return Release{}, "", f.err
	}
	if etag != "" && etag == f.etag {
		return Release{}, etag, ErrNotModified
	}
	return f.rel, f.etag, nil
}

func (f *fakeReleases) Stage(_ context.Context, rel Release, goos, goarch string, progress Progress) (string, error) {
	progress(50, 100)
	if f.staged != nil {
		<-f.staged
	}
	if f.stageErr != nil {
		return "", f.stageErr
	}
	return "", nil
}

type fakeLauncher struct {
	started chan Version
	err     error
}

func (l *fakeLauncher) Start(dir string, v Version) (func() error, error) {
	l.started <- v
	return func() error { return l.err }, nil
}

func release(t *testing.T, v string) Release {
	return Release{Version: mustVersion(t, v), Tag: "v" + v, Notes: "- new", URL: "https://example/" + v}
}

func newTestService(t *testing.T, version string, rel *fakeReleases) (*Service, Store) {
	t.Helper()
	store := Store{Path: filepath.Join(t.TempDir(), "update.json")}
	return NewService(Options{
		Version: version, GOOS: "linux", GOARCH: "amd64",
		Releases: rel, Launcher: &fakeLauncher{started: make(chan Version, 1)},
		Store: store, CanInstall: true, Auto: true,
	}), store
}

func waitFor(t *testing.T, s *Service, ok func(Status) bool) Status {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st := s.Status(); ok(st) {
			return st
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out; status %+v", s.Status())
	return Status{}
}

func TestCheckOffersANewerRelease(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3"), etag: `"a"`}
	s, store := newTestService(t, "0.1.2", rel)
	s.check(context.Background(), false)
	st := s.Status()
	if st.State != Available || st.Latest.Version.String() != "0.1.3" || st.Manual || st.Checks != 1 {
		t.Fatalf("%+v", st)
	}
	if rec := store.Load(); rec.ETag != `"a"` || rec.Latest.Tag != "v0.1.3" || rec.CheckedAt.IsZero() {
		t.Errorf("record %+v", rec)
	}

	// The next check is conditional and reuses the saved release.
	s.check(context.Background(), true)
	if rel.lastTag != `"a"` {
		t.Errorf("sent ETag %q", rel.lastTag)
	}
	if st := s.Status(); st.State != Available || !st.Manual || st.Checks != 2 {
		t.Errorf("after 304: %+v", st)
	}
}

func TestCheckOnTheLatestIsCurrent(t *testing.T) {
	s, _ := newTestService(t, "0.1.3", &fakeReleases{rel: release(t, "0.1.3")})
	s.check(context.Background(), true)
	if st := s.Status(); st.State != Current || st.Unreachable {
		t.Errorf("%+v", st)
	}
}

func TestOfflineIsQuietUnlessAsked(t *testing.T) {
	rel := &fakeReleases{err: errors.New("no network")}
	s, _ := newTestService(t, "0.1.2", rel)
	s.check(context.Background(), false)
	if st := s.Status(); st.Unreachable || st.Checks != 0 || st.State != Idle {
		t.Errorf("automatic check spoke up: %+v", st)
	}
	s.check(context.Background(), true)
	if st := s.Status(); !st.Unreachable || !st.Manual || st.Checks != 1 {
		t.Errorf("manual check: %+v", st)
	}
}

func TestOfflineKeepsAKnownUpdate(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3")}
	s, _ := newTestService(t, "0.1.2", rel)
	s.check(context.Background(), false)
	rel.err = errors.New("offline")
	s.check(context.Background(), true)
	if st := s.Status(); st.State != Available || !st.Unreachable {
		t.Errorf("%+v", st)
	}
}

func TestDevBuildsNeverCheck(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3")}
	for _, v := range []string{"dev", "0.1.3-next"} {
		s, _ := newTestService(t, v, rel)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s.Run(ctx)
		s.CheckNow()
		time.Sleep(20 * time.Millisecond)
		if st := s.Status(); st.Enabled || st.State != Idle {
			t.Errorf("%s: %+v", v, st)
		}
	}
	if rel.calls != 0 {
		t.Errorf("dev builds made %d requests", rel.calls)
	}
}

func TestSavedReleaseOffersAtLaunch(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "update.json")}
	store.Save(Record{Latest: &SavedRelease{Tag: "v0.1.3"}})
	s := NewService(Options{Version: "0.1.2", Store: store, Releases: &fakeReleases{}})
	if st := s.Status(); st.State != Available {
		t.Errorf("%+v", st)
	}
	s = NewService(Options{Version: "0.1.3", Store: store, Releases: &fakeReleases{}})
	if st := s.Status(); st.State != Idle {
		t.Errorf("already on it: %+v", st)
	}
}

func TestLaunchAfterUpdateGreets(t *testing.T) {
	store := Store{Path: filepath.Join(t.TempDir(), "update.json")}
	store.Save(Record{Pending: &Pending{From: "0.1.2", To: "0.1.3"}})
	s := NewService(Options{Version: "0.1.3", Store: store})
	if st := s.Status(); st.Updated != "0.1.2" {
		t.Errorf("%+v", st)
	}
	if store.Load().Pending != nil {
		t.Error("pending update was not cleared")
	}

	store.Save(Record{Pending: &Pending{From: "0.1.2", To: "0.1.3"}})
	s = NewService(Options{Version: "0.1.2", Store: store})
	if st := s.Status(); st.Updated != "" {
		t.Errorf("an update that did not land greeted: %+v", st)
	}
}

func TestInstallNeedsAClick(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3")}
	s, _ := newTestService(t, "0.1.2", rel)
	launcher := s.opt.Launcher.(*fakeLauncher)
	s.Install()
	if st := s.Status(); st.State != Idle {
		t.Fatalf("installed with nothing offered: %+v", st)
	}
	s.check(context.Background(), false)
	select {
	case v := <-launcher.started:
		t.Fatalf("a check started the installer for %s", v)
	default:
	}
}

func TestInstallRunsTheInstaller(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3"), staged: make(chan struct{})}
	s, store := newTestService(t, "0.1.2", rel)
	launcher := s.opt.Launcher.(*fakeLauncher)
	launcher.err = errors.New("checksum mismatch for x")
	s.check(context.Background(), false)

	s.Install()
	st := waitFor(t, s, func(st Status) bool { return st.Progress == 0.5 })
	if st.State != Downloading {
		t.Fatalf("%+v", st)
	}
	close(rel.staged)
	if v := <-launcher.started; v.String() != "0.1.3" {
		t.Errorf("installer got %s", v)
	}
	st = waitFor(t, s, func(st Status) bool { return st.State == Failed })
	if st.Problem != "checksum mismatch for x" {
		t.Errorf("problem %q", st.Problem)
	}
	if store.Load().Pending != nil {
		t.Error("a failed install left a pending update")
	}
}

func TestInstallWithoutTheInstalledCopy(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3")}
	s, _ := newTestService(t, "0.1.2", rel)
	s.opt.CanInstall = false
	s.st.CanInstall = false
	s.check(context.Background(), false)
	s.Install()
	if st := s.Status(); st.State != Available {
		t.Errorf("%+v", st)
	}
}

func TestDownloadFailureIsReported(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3"), stageErr: errors.New("download: 404 Not Found")}
	s, _ := newTestService(t, "0.1.2", rel)
	s.check(context.Background(), false)
	s.Install()
	st := waitFor(t, s, func(st Status) bool { return st.State == Failed })
	if st.Problem != "download: 404 Not Found" {
		t.Errorf("%+v", st)
	}
}

func TestCheckDuringInstallKeepsInstalling(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3"), staged: make(chan struct{})}
	s, _ := newTestService(t, "0.1.2", rel)
	s.check(context.Background(), false)
	s.Install()
	waitFor(t, s, func(st Status) bool { return st.State == Downloading })
	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
	s.check(context.Background(), true)
	if st := s.Status(); st.State != Downloading {
		t.Errorf("a check during the download reset the state: %+v", st)
	}
	s.Install()
	close(rel.staged)
	<-s.opt.Launcher.(*fakeLauncher).started
	select {
	case <-s.opt.Launcher.(*fakeLauncher).started:
		t.Error("the installer started twice")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestManualClickDuringAutomaticCheckIsReported(t *testing.T) {
	rel := &fakeReleases{rel: release(t, "0.1.3")}
	s, _ := newTestService(t, "0.1.2", rel)
	s.mu.Lock()
	s.busy = true
	s.mu.Unlock()
	s.check(context.Background(), true)
	s.mu.Lock()
	s.busy = false
	s.mu.Unlock()
	s.check(context.Background(), false)
	if st := s.Status(); !st.Manual || st.State != Available {
		t.Errorf("%+v", st)
	}
}
