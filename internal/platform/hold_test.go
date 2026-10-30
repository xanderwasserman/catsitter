package platform

import (
	"errors"
	"testing"
)

func TestHoldAndTapReleasesModifierWhenTapFails(t *testing.T) {
	var got []string
	err := HoldAndTap(
		func() error { got = append(got, "mod-down"); return nil },
		func() error { got = append(got, "mod-up"); return nil },
		func() error { got = append(got, "key-down"); return errors.New("tab failed") },
		func() error { got = append(got, "key-up"); return nil },
		3,
		nil, nil, nil,
	)
	if err == nil || err.Error() != "tab failed" {
		t.Fatalf("err = %v", err)
	}
	want := []string{"mod-down", "key-down", "mod-up"}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v", got)
		}
	}
}

func TestHoldAndTapReleasesModifierWhenKeyUpFails(t *testing.T) {
	var got []string
	err := HoldAndTap(
		func() error { got = append(got, "mod-down"); return nil },
		func() error { got = append(got, "mod-up"); return nil },
		func() error { got = append(got, "key-down"); return nil },
		func() error { got = append(got, "key-up"); return errors.New("up failed") },
		2,
		nil, nil, nil,
	)
	if err == nil || err.Error() != "up failed" {
		t.Fatalf("err = %v", err)
	}
	want := []string{"mod-down", "key-down", "key-up", "mod-up"}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v", got)
		}
	}
}

func TestHoldAndTapCompletesAndReleasesOnce(t *testing.T) {
	var got []string
	err := HoldAndTap(
		func() error { got = append(got, "mod-down"); return nil },
		func() error { got = append(got, "mod-up"); return nil },
		func() error { got = append(got, "key-down"); return nil },
		func() error { got = append(got, "key-up"); return nil },
		2,
		func() { got = append(got, "pause") },
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"mod-down", "pause", "key-down", "key-up", "key-down", "key-up", "mod-up"}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v", got)
		}
	}
}

func TestHoldAndTapSkipsReleaseWhenModifierNeverWentDown(t *testing.T) {
	ups := 0
	err := HoldAndTap(
		func() error { return errors.New("no modifier") },
		func() error { ups++; return nil },
		func() error { return nil },
		func() error { return nil },
		1,
		nil, nil, nil,
	)
	if err == nil || ups != 0 {
		t.Fatalf("err=%v ups=%d", err, ups)
	}
}
