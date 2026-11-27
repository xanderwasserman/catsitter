package win

import (
	"errors"
	"fmt"
)

// powerKind is a POWER_REQUEST_TYPE value.
type powerKind uint32

const (
	powerDisplayRequired powerKind = 0
	powerSystemRequired  powerKind = 1
)

func (k powerKind) String() string {
	switch k {
	case powerDisplayRequired:
		return "display"
	case powerSystemRequired:
		return "system"
	}
	return fmt.Sprintf("kind %d", uint32(k))
}

// powerRequests is the part of the Windows power request API the inhibitor
// uses. A request handle belongs to the process, so any thread may use it.
// Closing the handle drops every request still set on it.
type powerRequests interface {
	create() (uintptr, error)
	set(h uintptr, kind powerKind) error
	clear(h uintptr, kind powerKind) error
	close(h uintptr) error
}

var powerKinds = [...]powerKind{powerDisplayRequired, powerSystemRequired}

// powerInhibitor keeps the system and display awake with one power request
// handle per hold. Each kind is set at most once, so repeated Acquire and
// Release calls never stack requests.
type powerInhibitor struct {
	api    powerRequests
	handle uintptr
	set    map[powerKind]bool
}

func newPowerInhibitor(api powerRequests) *powerInhibitor {
	return &powerInhibitor{api: api, set: map[powerKind]bool{}}
}

// Acquire sets every kind. If one fails, everything is released again.
func (p *powerInhibitor) Acquire() error {
	if p.handle == 0 {
		h, err := p.api.create()
		if err != nil {
			return fmt.Errorf("PowerCreateRequest: %w", err)
		}
		p.handle = h
	}
	for _, k := range powerKinds {
		if p.set[k] {
			continue
		}
		if err := p.api.set(p.handle, k); err != nil {
			return errors.Join(fmt.Errorf("PowerSetRequest %s: %w", k, err), p.Release())
		}
		p.set[k] = true
	}
	return nil
}

// Release clears every kind that is set and closes the handle. Clear
// failures are reported even when closing the handle dropped the request.
// Whatever is still held after a failure is tried again on the next Release.
func (p *powerInhibitor) Release() error {
	if p.handle == 0 {
		return nil
	}
	var errs []error
	for _, k := range powerKinds {
		if !p.set[k] {
			continue
		}
		if err := p.api.clear(p.handle, k); err != nil {
			errs = append(errs, fmt.Errorf("PowerClearRequest %s: %w", k, err))
			continue
		}
		delete(p.set, k)
	}
	if err := p.api.close(p.handle); err != nil {
		errs = append(errs, fmt.Errorf("CloseHandle: %w", err))
	} else {
		p.handle = 0
		clear(p.set)
	}
	return errors.Join(errs...)
}
