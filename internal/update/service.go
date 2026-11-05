package update

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// State is where the updater is in its cycle.
type State int

const (
	// Idle means no check has finished yet.
	Idle State = iota
	// Current means the running version is the newest.
	Current
	// Available means a newer release can be installed.
	Available
	// Downloading means the new release is being fetched and verified.
	Downloading
	// Installing means the installer is replacing this copy.
	Installing
	// Failed means the last install attempt did not work.
	Failed
)

// Status is a snapshot for the menus.
type Status struct {
	State   State
	Version string
	// Enabled is false for dev and snapshot builds, which never check.
	Enabled  bool
	Latest   Release
	Checking bool
	// Manual is true when the last finished check was asked for by the user.
	Manual bool
	// Unreachable is true when that check could not reach GitHub.
	Unreachable bool
	// Checks counts finished checks, so a view can tell when a new result lands.
	Checks   int
	Progress float64
	Problem  string
	// CanInstall is false when this copy was not put there by the installer.
	CanInstall bool
	// Updated is the previous version when this launch follows an update.
	Updated string
	Auto    bool
}

// Releases is where updates come from.
type Releases interface {
	Latest(ctx context.Context, etag string) (Release, string, error)
	Stage(ctx context.Context, rel Release, goos, goarch string, progress Progress) (string, error)
}

// Launcher starts the installer against a staged release.
type Launcher interface {
	Start(dir string, v Version) (wait func() error, err error)
}

// Options configures a Service.
type Options struct {
	Version string
	// Repo is "owner/name"; release pages outside it are never shown.
	Repo       string
	GOOS       string
	GOARCH     string
	Releases   Releases
	Launcher   Launcher
	Store      Store
	CanInstall bool
	Auto       bool
	Now        func() time.Time
}

const (
	// firstCheckDelay keeps the network quiet while the app starts.
	firstCheckDelay = 15 * time.Second
	// startupGap skips the launch check when one ran this recently.
	startupGap = time.Hour
	// checkEvery is how often a long-running Catsitter looks again.
	checkEvery     = 12 * time.Hour
	checkTimeout   = 20 * time.Second
	installTimeout = 10 * time.Minute
)

// Service checks for updates in the background and installs one on request.
type Service struct {
	opt  Options
	mu   sync.Mutex
	st   Status
	rec  Record
	busy bool
	// askedDuringCheck makes the running check report like a manual one.
	askedDuringCheck bool
}

// NewService reads the saved record. A launch that follows an update is
// noted in Status.Updated.
func NewService(opt Options) *Service {
	if opt.Now == nil {
		opt.Now = time.Now
	}
	s := &Service{opt: opt, rec: opt.Store.Load()}
	s.st = Status{Version: opt.Version, Enabled: IsRelease(opt.Version), CanInstall: opt.CanInstall, Auto: opt.Auto}
	if p := s.rec.Pending; p != nil {
		if p.To == opt.Version {
			s.st.Updated = p.From
		}
		s.rec.Pending = nil
		s.save()
		if ownDownload(p.Dir) {
			go func() {
				time.Sleep(time.Minute)
				os.RemoveAll(p.Dir)
			}()
		}
	}
	if rel, ok := s.rec.Latest.Release(); ok && Offer(opt.Version, rel.Version) {
		s.st.State = Available
		s.st.Latest = s.page(rel)
	}
	return s
}

// Status returns the current snapshot.
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st
}

// SetAuto turns the automatic checks on or off.
func (s *Service) SetAuto(on bool) {
	s.mu.Lock()
	s.st.Auto = on
	s.mu.Unlock()
}

// Run checks on launch and then every 12 hours while automatic checks are on.
func (s *Service) Run(ctx context.Context) {
	if !s.st.Enabled {
		return
	}
	timer := time.NewTimer(firstCheckDelay)
	defer timer.Stop()
	gap := startupGap
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		s.mu.Lock()
		due := s.st.Auto && Due(s.rec.CheckedAt, s.opt.Now(), gap)
		s.mu.Unlock()
		if due {
			s.check(ctx, false)
		}
		gap = checkEvery
		timer.Reset(time.Hour)
	}
}

// CheckNow looks for an update because the user asked. It returns at once.
func (s *Service) CheckNow() {
	if !s.st.Enabled {
		return
	}
	go s.check(context.Background(), true)
}

func (s *Service) check(ctx context.Context, manual bool) {
	s.mu.Lock()
	if s.installing() {
		s.mu.Unlock()
		return
	}
	if s.busy {
		s.askedDuringCheck = s.askedDuringCheck || manual
		s.mu.Unlock()
		return
	}
	s.busy = true
	s.st.Checking = true
	etag := s.rec.ETag
	if _, ok := s.rec.Latest.Release(); !ok {
		etag = ""
	}
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	rel, newTag, err := s.opt.Releases.Latest(ctx, etag)
	if errors.Is(err, ErrNotModified) {
		rel, _ = s.rec.Latest.Release()
		err = nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
	s.st.Checking = false
	manual = manual || s.askedDuringCheck
	s.askedDuringCheck = false
	if err != nil {
		log.Printf("update check: %v", err)
		if manual {
			s.st.Unreachable = true
			s.st.Manual = true
			s.st.Checks++
		}
		return
	}
	s.rec.CheckedAt = s.opt.Now()
	s.rec.ETag = newTag
	s.rec.Latest = saveRelease(rel)
	s.save()
	s.st.Unreachable = false
	if !s.installing() {
		s.st.Latest = s.page(rel)
		s.st.State = Current
		if Offer(s.opt.Version, rel.Version) {
			s.st.State = Available
		}
	}
	s.st.Manual = manual
	s.st.Checks++
}

// page keeps the release URL only when it points into this repository.
func (s *Service) page(rel Release) Release {
	repo := s.opt.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	rel.URL = PageURL(repo, rel.URL, rel.Tag)
	return rel
}

func (s *Service) installing() bool {
	return s.st.State == Downloading || s.st.State == Installing
}

// Install downloads, verifies and installs the available update. The
// installer stops this process and starts the new version.
func (s *Service) Install() {
	s.mu.Lock()
	if !s.st.CanInstall || (s.st.State != Available && s.st.State != Failed) || s.st.Latest.Tag == "" {
		s.mu.Unlock()
		return
	}
	rel := s.st.Latest
	s.st.State = Downloading
	s.st.Progress = 0
	s.st.Problem = ""
	s.mu.Unlock()
	go s.install(rel)
}

func (s *Service) install(rel Release) {
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	dir, err := s.opt.Releases.Stage(ctx, rel, s.opt.GOOS, s.opt.GOARCH, func(done, total int64) {
		if total <= 0 {
			return
		}
		s.mu.Lock()
		s.st.Progress = float64(done) / float64(total)
		s.mu.Unlock()
	})
	if err != nil {
		s.fail(err, "")
		return
	}
	s.mu.Lock()
	s.st.State = Installing
	s.st.Progress = 1
	s.rec.Pending = &Pending{From: s.opt.Version, To: rel.Version.String(), Dir: dir}
	s.save()
	s.mu.Unlock()
	log.Printf("installing Catsitter %s", rel.Version)
	wait, err := s.opt.Launcher.Start(dir, rel.Version)
	if err != nil {
		s.fail(err, dir)
		return
	}
	if err := wait(); err != nil {
		s.fail(err, dir)
		return
	}
	s.fail(errors.New("the installer finished but this Catsitter is still running"), dir)
}

func (s *Service) fail(err error, dir string) {
	log.Printf("update: %v", err)
	if dir != "" {
		os.RemoveAll(dir)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rec.Pending = nil
	s.save()
	s.st.State = Failed
	s.st.Problem = err.Error()
}

func (s *Service) save() {
	if s.opt.Store.Path == "" {
		return
	}
	if err := s.opt.Store.Save(s.rec); err != nil {
		log.Printf("save update state: %v", err)
	}
}

// ownDownload reports whether dir is a staging folder Stage created.
func ownDownload(dir string) bool {
	if dir == "" || !strings.HasPrefix(filepath.Base(dir), stagePrefix) {
		return false
	}
	rel, err := filepath.Rel(os.TempDir(), dir)
	return err == nil && rel == filepath.Base(dir)
}
