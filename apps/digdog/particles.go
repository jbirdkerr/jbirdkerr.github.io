package main

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

// Particle is a single dirt speck flying through the air.
type Particle struct {
	x, y   float64
	vx, vy float64
	life   int
	col    color.RGBA
}

// Particles is a slice of active Particle values.
type Particles []Particle

var dirtColors = [2]color.RGBA{
	{0x6b, 0x46, 0x26, 0xff},
	{0x8a, 0x5e, 0x38, 0xff},
}

// Spawn adds one new dirt particle near (x, y). The total is capped at 200.
func (ps *Particles) Spawn(x, y float64) {
	if len(*ps) >= 200 {
		return
	}
	p := Particle{
		x:    x,
		y:    y,
		vx:   (rand.Float64()*2 - 1),    // -1 to 1
		vy:   -(rand.Float64()*1.5 + 1), // -2.5 to -1
		life: 20 + rand.Intn(11),        // 20..30
		col:  dirtColors[rand.Intn(2)],
	}
	*ps = append(*ps, p)
}

// Update advances every particle by one tick and removes dead ones.
func (ps *Particles) Update() {
	alive := (*ps)[:0]
	for _, p := range *ps {
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.15 // gravity
		p.life--
		if p.life > 0 {
			alive = append(alive, p)
		}
	}
	*ps = alive
}

// Draw renders every particle as a 2×2 square.
func (ps Particles) Draw(screen *ebiten.Image) {
	for _, p := range ps {
		fillRect(screen, int(math.Round(p.x)), int(math.Round(p.y)), 2, 2, p.col)
	}
}
