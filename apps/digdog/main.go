package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenW   = 320
	screenH   = 180
	frameSize = 32
	dogSpeed  = 2.0 // pixels per tick
)

//go:embed assets/dog.png
var dogPNG []byte

// Animation steps through a slice of frames, advancing once every ticksPerFrame updates.
type Animation struct {
	frames        []*ebiten.Image
	ticksPerFrame int
	loop          bool
	tick          int
}

func (a *Animation) Update() { a.tick++ }
func (a *Animation) Reset()  { a.tick = 0 }

func (a *Animation) index() int {
	i := a.tick / a.ticksPerFrame
	if a.loop {
		return i % len(a.frames)
	}
	if i >= len(a.frames) {
		return len(a.frames) - 1
	}
	return i
}

func (a *Animation) Frame() *ebiten.Image { return a.frames[a.index()] }

func (a *Animation) Done() bool {
	return !a.loop && a.tick/a.ticksPerFrame >= len(a.frames)
}

func sliceRow(sheet *ebiten.Image, row, count int) []*ebiten.Image {
	frames := make([]*ebiten.Image, count)
	for i := 0; i < count; i++ {
		r := image.Rect(i*frameSize, row*frameSize, (i+1)*frameSize, (row+1)*frameSize)
		frames[i] = sheet.SubImage(r).(*ebiten.Image)
	}
	return frames
}

type Game struct {
	anims   map[string]*Animation
	current string

	board      *Board
	dogX, dogY float64 // centre of the dog sprite
	facing     float64 // 1 = right, -1 = left
	target     int     // card index the dog is heading for or digging at
	walking    bool
	digging    bool
}

func newGame() (*Game, error) {
	img, _, err := image.Decode(bytes.NewReader(dogPNG))
	if err != nil {
		return nil, err
	}
	sheet := ebiten.NewImageFromImage(img)

	g := &Game{
		anims: map[string]*Animation{
			"idle":  {frames: sliceRow(sheet, 0, 4), ticksPerFrame: 15, loop: true},
			"dig":   {frames: sliceRow(sheet, 1, 4), ticksPerFrame: 6},
			"sniff": {frames: sliceRow(sheet, 2, 3), ticksPerFrame: 10},
			"happy": {frames: sliceRow(sheet, 3, 3), ticksPerFrame: 8},
			"sad":   {frames: sliceRow(sheet, 4, 2), ticksPerFrame: 12},
		},
	}
	g.reset()
	return g, nil
}

func (g *Game) reset() {
	g.board = NewBoard(4, 3)
	g.dogX, g.dogY = screenW/2, screenH-18
	g.facing = 1
	g.walking, g.digging = false, false
	g.play("idle")
}

func (g *Game) play(name string) {
	g.current = name
	g.anims[name].Reset()
}

// pointerPos returns a fresh click or tap in logical screen coordinates.
func pointerPos() (int, int, bool) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		return x, y, true
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		return x, y, true
	}
	return 0, 0, false
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.reset()
	}
	g.board.Update()

	// Only accept a new pick while the dog is free.
	if !g.walking && !g.digging {
		if x, y, ok := pointerPos(); ok {
			if i := g.board.CellAt(x, y); g.board.Pick(i) {
				g.target = i
				g.walking = true
			}
		}
	}

	if g.walking {
		g.walk()
	}

	anim := g.anims[g.current]
	anim.Update()
	if anim.Done() {
		if g.digging {
			g.digging = false
			g.afterReveal(g.board.Reveal(g.target))
		} else {
			g.play("idle")
		}
	}
	return nil
}

// walk moves the dog toward the target mound and starts digging on arrival.
func (g *Game) walk() {
	cx, cy := g.board.CellCenter(g.target)
	tx, ty := cx, cy+16 // stand just below the mound so the card stays visible
	dx, dy := tx-g.dogX, ty-g.dogY
	dist := math.Hypot(dx, dy)

	if dist <= dogSpeed {
		g.dogX, g.dogY = tx, ty
		g.walking = false
		g.digging = true
		g.play("dig")
		return
	}
	g.dogX += dx / dist * dogSpeed
	g.dogY += dy / dist * dogSpeed
	if math.Abs(dx) > 0.5 {
		g.facing = math.Copysign(1, dx)
	}
}

func (g *Game) afterReveal(r Result) {
	switch r {
	case ResultMatch, ResultWin:
		g.play("happy")
	case ResultMiss:
		g.play("sad")
	default:
		g.play("idle")
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 0x1a, G: 0x12, B: 0x0c, A: 0xff})
	g.board.Draw(screen)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	if g.facing < 0 {
		op.GeoM.Scale(-1, 1)
		op.GeoM.Translate(frameSize, 0)
	}
	op.GeoM.Translate(math.Round(g.dogX)-frameSize/2, math.Round(g.dogY)-frameSize/2)
	screen.DrawImage(g.anims[g.current].Frame(), op)

	msg := fmt.Sprintf("moves: %d", g.board.Moves)
	if g.board.Won {
		msg = fmt.Sprintf("all matched in %d moves! press R", g.board.Moves)
	}
	ebitenutil.DebugPrintAt(screen, msg, 4, 2)
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return screenW, screenH
}

func main() {
	ebiten.SetWindowSize(screenW*3, screenH*3)
	ebiten.SetWindowTitle("DigDog")

	g, err := newGame()
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
