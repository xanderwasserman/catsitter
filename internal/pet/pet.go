// Package pet is the floating desktop pet. Another animal can implement Pet later.
package pet

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

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
	// Frame is the image for activity at tick, an AnimTick count.
	Frame(activity Activity, tick int64) *ebiten.Image
}

// AnimRate is how many animation ticks pass in a second.
const AnimRate = 12

// AnimTick counts animation ticks over elapsed wall time. It does not follow
// the window's tick rate, which rises while a panel is open, so animations
// keep their pace whatever is on screen.
func AnimTick(elapsed time.Duration) int64 {
	return int64(elapsed / (time.Second / AnimRate))
}
