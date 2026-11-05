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

func (g *fakeGate) counts() (prompts, granted int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.prompts, g.granted
}

func newTrustApp(gate *fakeGate) *App {
	return New(config.Defaults(), config.Defaults(), "", host.Backends{Gate: gate})
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
	a := New(cfg, cfg, "", host.Backends{Gate: gate})
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
