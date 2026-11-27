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
	trust(a, true, true)
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

func (g *fakeGate) reopenCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.reopens
}

// trust feeds the permission watch one poll per value.
func trust(a *App, polls ...bool) {
	for _, ok := range polls {
		a.applyTrust(ok)
	}
}

func TestGrantWhileRunningRelaunchesOnce(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	trust(a, false, true, true, true, false, true, true)
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

func TestOneTrustedPollDoesNotRelaunch(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	trust(a, false, true, false, true, false)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
}

func TestUnconfirmedGrantStaysUntrusted(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	trust(a, false, true)
	if a.View().Trusted {
		t.Fatal("view reports trusted before the grant is confirmed")
	}
	if a.trusted.Load() {
		t.Fatal("nudges are on before the grant is confirmed")
	}
}

func TestStartedTrustedNeverRelaunches(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	trust(a, true, true, true)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
	if p, g := gate.counts(); p != 0 || g != 0 {
		t.Fatalf("prompts %d, granted %d", p, g)
	}
}

func TestRevokeNeverRelaunches(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	trust(a, true, false, false)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
	if a.View().Trusted {
		t.Fatal("view reports trusted after the revoke")
	}
}

func TestRegrantAfterRevokeRelaunchesOnce(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true}
	a := newRelaunchApp(gate, r)
	trust(a, true, false, true, true, true)
	if n := r.count(); n != 1 {
		t.Fatalf("relaunches %d", n)
	}
	if !a.View().Quitting {
		t.Fatal("app keeps running after the relaunch was arranged")
	}
}

func TestRelaunchedCopyStartedTrustedThanks(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true, relaunched: true}
	a := newRelaunchApp(gate, r)
	trust(a, true, true)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
	if _, g := gate.counts(); g != 1 {
		t.Fatalf("granted %d", g)
	}
}

func TestRelaunchedCopyStartedTrustedRelaunchesOnRegrant(t *testing.T) {
	gate := &fakeGate{trusted: true}
	r := &fakeRelauncher{works: true, relaunched: true}
	a := newRelaunchApp(gate, r)
	trust(a, true, false, true, true, true)
	if n := r.count(); n != 1 {
		t.Fatalf("relaunches %d", n)
	}
	if gate.reopenCount() != 0 || !a.View().Quitting {
		t.Fatalf("reopens %d, view %+v", gate.reopenCount(), a.View())
	}
}

func TestRelaunchedCopyStartedUntrustedAsksToReopen(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{works: true, relaunched: true}
	a := newRelaunchApp(gate, r)
	trust(a, false, true, true, true)
	if n := r.count(); n != 0 {
		t.Fatalf("relaunches %d", n)
	}
	if _, g := gate.counts(); g != 0 || gate.reopenCount() != 1 {
		t.Fatalf("granted %d, reopens %d", g, gate.reopenCount())
	}
	if v := a.View(); !v.Reopen || v.Quitting {
		t.Fatalf("view %+v", v)
	}
}

func TestReopenWhenRelaunchIsUnavailable(t *testing.T) {
	gate := &fakeGate{}
	r := &fakeRelauncher{}
	a := newRelaunchApp(gate, r)
	trust(a, false, true, true, true)
	if a.View().Quitting {
		t.Fatal("app quit without a relaunch")
	}
	if _, g := gate.counts(); g != 0 || gate.reopenCount() != 1 {
		t.Fatalf("granted %d, reopens %d", g, gate.reopenCount())
	}
	if !a.View().Reopen {
		t.Fatal("view does not ask for a reopen")
	}
	trust(a, false)
	if a.View().Reopen {
		t.Fatal("reopen kept after the revoke")
	}
}
