package win

import (
	"errors"
	"fmt"
)

// PowerKind is a POWER_REQUEST_TYPE value.
type PowerKind uint32

const (
	PowerDisplayRequired PowerKind = 0
	PowerSystemRequired  PowerKind = 1
)

func (k PowerKind) String() string {
	switch k {
	case PowerDisplayRequired:
		return "display"
	case PowerSystemRequired:
		return "system"
	}
	return fmt.Sprintf("kind %d", uint32(k))
}

// PowerRequests is the part of the Windows power request API the inhibitor uses.
// A request handle belongs to the process, so any thread may set or clear it.
type PowerRequests interface {
	Create() (uintptr, error)
	Set(h uintptr, kind PowerKind) error
	Clear(h uintptr, kind PowerKind) error
}

var powerKinds = [...]PowerKind{PowerDisplayRequired, PowerSystemRequired}

// PowerInhibitor keeps the system and display awake with one power request
// handle, created on first use. Each kind is set at most once, so repeated
// Acquire and Release calls never stack requests.
type PowerInhibitor struct {
	api    PowerRequests
	handle uintptr
	set    map[PowerKind]bool
}

func NewPowerInhibitor(api PowerRequests) *PowerInhibitor {
	return &PowerInhibitor{api: api, set: map[PowerKind]bool{}}
}

// Acquire sets every kind. If one fails, the kinds it set are cleared again.
func (p *PowerInhibitor) Acquire() error {
	if p.handle == 0 {
		h, err := p.api.Create()
		if err != nil {
			return fmt.Errorf("PowerCreateRequest: %w", err)
		}
		p.handle = h
	}
	var added []PowerKind
	for _, k := range powerKinds {
		if p.set[k] {
			continue
		}
		if err := p.api.Set(p.handle, k); err != nil {
			errs := []error{fmt.Errorf("PowerSetRequest %s: %w", k, err)}
			for _, a := range added {
				errs = append(errs, p.clear(a))
			}
			return errors.Join(errs...)
		}
		p.set[k] = true
		added = append(added, k)
	}
	return nil
}

// Release clears every kind that is set. A kind that fails to clear stays
// recorded, so the next Release tries it again.
func (p *PowerInhibitor) Release() error {
	var errs []error
	for _, k := range powerKinds {
		if p.set[k] {
			errs = append(errs, p.clear(k))
		}
	}
	return errors.Join(errs...)
}

func (p *PowerInhibitor) clear(k PowerKind) error {
	if err := p.api.Clear(p.handle, k); err != nil {
		return fmt.Errorf("PowerClearRequest %s: %w", k, err)
	}
	delete(p.set, k)
	return nil
}
