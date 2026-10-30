package cat

import (
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/xanderwasserman/catsitter/internal/pet"
)

// Cat is the only pet that ships with Catsitter.
type Cat struct {
	once   sync.Once
	frames map[pet.Activity][]*ebiten.Image
}

// New returns the cat.
func New() *Cat { return &Cat{} }

func (c *Cat) Name() string { return "cat" }

func (c *Cat) Size() (int, int) { return frameW * Scale, frameH * Scale }

func (c *Cat) Frame(activity pet.Activity, tick int64) *ebiten.Image {
	c.once.Do(c.load)
	frames := c.frames[activity]
	if len(frames) == 0 {
		frames = c.frames[pet.ActivityIdle]
	}
	return frames[(tick/2)%int64(len(frames))]
}

func (c *Cat) load() {
	src := Frames()
	c.frames = map[pet.Activity][]*ebiten.Image{
		pet.ActivityIdle:  images(src["idle"]),
		pet.ActivityWalk:  images(src["walk"]),
		pet.ActivitySit:   images(src["sit"]),
		pet.ActivitySleep: images(src["sleep"]),
		pet.ActivityPaws:  images(src["paws"]),
	}
}

func images(frames []*image.RGBA) []*ebiten.Image {
	out := make([]*ebiten.Image, len(frames))
	for i, frame := range frames {
		out[i] = ebiten.NewImageFromImage(frame)
	}
	return out
}
