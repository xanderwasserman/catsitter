// Package pet is the floating desktop pet. Another animal can implement Pet later.
package pet

import "github.com/hajimehoshi/ebiten/v2"

// Activity is what the pet is doing right now.
type Activity int

const (
	ActivityIdle Activity = iota
	ActivityWalk
	ActivitySit
	ActivitySleep
	ActivityPaws
)

// Pet draws one animation frame.
type Pet interface {
	Name() string
	// Size is the on-screen size of the character in pixels.
	Size() (w, h int)
	Frame(activity Activity, tick int64) *ebiten.Image
}
