package app

import (
	"sync"
	"testing"

	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/host"
)

type fakeGate struct {
	mu      sync.Mutex
	trusted bool
	prompts int
	granted int
	reopens int
}

func (g *fakeGate) Trusted() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.trusted
}

func (g *fakeGate) Prompt() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.prompts++
}

func (g *fakeGate) Granted() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.granted++
}

func (g *fakeGate) Reopen() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reopens++
}

func (g *fakeGate) counts() (prompts, granted int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.prompts, g.granted
}

func newTrustApp(gate *fakeGate) *App {
	return New(config.Defaults(), config.Defaults(), "", host.Backends{Gate: gate}, nil)
}

func TestMissingPermissionPromptsOnce(t *testing.T) {
	gate := &fakeGate{}
	a := newTrustApp(gate)
	a.applyTrust(false)
	a.applyTrust(false)
	if p, g := gate.counts(); p != 1 || g != 0 {
		t.Fatalf("prompts %d, granted %d", p, g)
	}
	if a.View().Trusted {
		t.Fatal("view reports trusted")
	}
}

func TestGrantWhileRunningConfirmsOnceAndNeverPromptsAgain(t *testing.T) {
	gate := &fakeGate{}
	a := newTrustApp(gate)
	a.applyTrust(false)
	a.applyTrust(true)
	a.applyTrust(true)
	a.applyTrust(false)
	a.applyTrust(true)
	if p, g := gate.counts(); p != 1 || g != 1 {
		t.Fatalf("prompts %d, granted %d", p, g)
	}
}

func TestTrustedAtStartNeitherPromptsNorConfirms(t *testing.T) {
	gate := &fakeGate{trusted: true}
	a := newTrustApp(gate)
	a.applyTrust(true)
	a.applyTrust(false)
	if p, g := gate.counts(); p != 0 || g != 0 {
		t.Fatalf("prompts %d, granted %d", p, g)
	}
}

func TestNoPromptWhenNudgesAreOff(t *testing.T) {
	gate := &fakeGate{}
	cfg := config.Defaults()
	cfg.Mouse, cfg.Shift, cfg.Switch = false, false, false
	a := New(cfg, cfg, "", host.Backends{Gate: gate}, nil)
	a.applyTrust(false)
	if p, _ := gate.counts(); p != 0 {
		t.Fatalf("prompts %d", p)
	}
}

func TestRequestAccessPromptsOnlyWhileMissing(t *testing.T) {
	gate := &fakeGate{}
	a := newTrustApp(gate)
	a.applyTrust(false)
	a.RequestAccess()
	if p, _ := gate.counts(); p != 2 {
		t.Fatalf("prompts %d", p)
	}
	a.applyTrust(true)
	a.RequestAccess()
	if p, _ := gate.counts(); p != 2 {
		t.Fatalf("prompts after grant %d", p)
	}
}

type fakeRelauncher struct {
	mu         sync.Mutex
	works      bool
	relaunched bool
	calls      int
}

func (r *fakeRelauncher) Relaunch() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.works
}

func (r *fakeRelauncher) Relaunched() bool { return r.relaunched }

func (r *fakeRelauncher) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func newRelaunchApp(gate *fakeGate, r *fakeRelauncher) *App {
	return New(config.Defaults(), config.Defaults(), "", host.Backends{Gate: gate, Relaunch: r}, nil)
}

func TestGrantWhileRunningRelaunchesOnce(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	a.applyTrust(false)
	a.applyTrust(true)
	a.applyTrust(true)
	a.applyTrust(false)
	a.applyTrust(true)
	if n := r.count(); n != 1 {
		t.Fatalf("relaunches %d", n)
	}
	if !a.View().Quitting {
		t.Fatal("app keeps running after the relaunch was arranged")
	}
	if _, g := gate.counts(); g != 0 {
		t.Fatalf("granted %d before the relaunch", g)
	}
}

func TestStartedTrustedNeverRelaunches(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	a.applyTrust(true)
	a.applyTrust(false)
	a.applyTrust(true)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
	if a.View().Quitting {
		t.Fatal("app quit")
	}
}

func TestRevokeNeverRelaunches(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	a.applyTrust(true)
	a.applyTrust(false)
	a.applyTrust(false)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
	if a.View().Trusted {
		t.Fatal("view reports trusted after the revoke")
	}
}

func TestRelaunchedCopyThanksAndNeverRelaunches(t *testing.T) {
	for _, startTrusted := range []bool{true, false} {
		gate := &fakeGate{trusted: startTrusted}
		r := &fakeRelauncher{works: true, relaunched: true}
		a := newRelaunchApp(gate, r)
		a.applyTrust(startTrusted)
		a.applyTrust(true)
		if n := r.count(); n != 0 {
			t.Fatalf("started trusted %v: relaunches %d", startTrusted, n)
		}
		if _, g := gate.counts(); g != 1 {
			t.Fatalf("started trusted %v: granted %d", startTrusted, g)
		}
	}
}

func TestGrantAsksToReopenWhenRelaunchIsUnavailable(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{}
	a := newRelaunchApp(gate, r)
	a.applyTrust(false)
	a.applyTrust(true)
	a.applyTrust(true)
	if a.View().Quitting {
		t.Fatal("app quit without a relaunch")
	}
	gate.mu.Lock()
	reopens, granted := gate.reopens, gate.granted
	gate.mu.Unlock()
	if reopens != 1 || granted != 0 {
		t.Fatalf("reopens %d, granted %d", reopens, granted)
	}
}
