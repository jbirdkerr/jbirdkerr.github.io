package main

import (
	"image"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	cellW     = 44
	cellH     = 44
	cardW     = 36
	cardH     = 36
	missTicks = 45 // how long a mismatched pair stays face up (60 ticks = 1s)
)

type CardState int

const (
	Buried CardState = iota
	Revealed
	Matched
)

// Result tells the game what just happened after a card was dug up.
type Result int

const (
	ResultFirst Result = iota // first card of a pair
	ResultMatch
	ResultMiss
	ResultWin
)

type Card struct {
	Face  int
	State CardState
}

type Board struct {
	cols, rows int
	cards      []Card

	first        int // index of the first card dug up this turn, or -1
	missA, missB int // the pair waiting to be buried again after a miss
	missTimer    int
	locked       bool // true while the dog is busy or a miss is showing

	Moves int
	Won   bool
}

func NewBoard(cols, rows int) *Board {
	n := cols * rows
	if n%2 != 0 {
		panic("board needs an even number of cells")
	}
	faces := make([]int, 0, n)
	for i := 0; i < n/2; i++ {
		faces = append(faces, i, i)
	}
	rand.Shuffle(len(faces), func(i, j int) { faces[i], faces[j] = faces[j], faces[i] })

	cards := make([]Card, n)
	for i, f := range faces {
		cards[i] = Card{Face: f}
	}
	return &Board{cols: cols, rows: rows, cards: cards, first: -1}
}

func (b *Board) origin() (int, int) {
	// Nudged up a little to leave room for the dog at the bottom.
	return (screenW - b.cols*cellW) / 2, (screenH-b.rows*cellH)/2 - 8
}

// CellAt returns the card index under a point in screen coordinates, or -1.
func (b *Board) CellAt(x, y int) int {
	ox, oy := b.origin()
	cx, cy := x-ox, y-oy
	if cx < 0 || cy < 0 {
		return -1
	}
	col, row := cx/cellW, cy/cellH
	if col >= b.cols || row >= b.rows {
		return -1
	}
	return row*b.cols + col
}

func (b *Board) CellCenter(i int) (float64, float64) {
	ox, oy := b.origin()
	return float64(ox + (i%b.cols)*cellW + cellW/2), float64(oy + (i/b.cols)*cellH + cellH/2)
}

// Pick reserves a buried card for digging. It returns false if the pick isn't allowed.
func (b *Board) Pick(i int) bool {
	if b.locked || i < 0 || b.cards[i].State != Buried {
		return false
	}
	b.locked = true
	return true
}

// Reveal flips a picked card face up and works out what that means.
func (b *Board) Reveal(i int) Result {
	b.cards[i].State = Revealed

	if b.first < 0 {
		b.first = i
		b.locked = false
		return ResultFirst
	}

	a := b.first
	b.first = -1
	b.Moves++

	if b.cards[a].Face == b.cards[i].Face {
		b.cards[a].State, b.cards[i].State = Matched, Matched
		if b.allMatched() {
			b.Won = true // stays locked
			return ResultWin
		}
		b.locked = false
		return ResultMatch
	}

	b.missA, b.missB, b.missTimer = a, i, missTicks // stays locked until Update re-buries them
	return ResultMiss
}

// Update counts down the miss timer and buries a mismatched pair again.
func (b *Board) Update() {
	if b.missTimer == 0 {
		return
	}
	b.missTimer--
	if b.missTimer == 0 {
		b.cards[b.missA].State = Buried
		b.cards[b.missB].State = Buried
		b.locked = false
	}
}

func (b *Board) allMatched() bool {
	for _, c := range b.cards {
		if c.State != Matched {
			return false
		}
	}
	return true
}

// --- Placeholder drawing (swap for real sprites from cards.png later) ---

var faceColors = []color.RGBA{
	{0xe5, 0x48, 0x4d, 0xff},
	{0x3b, 0x82, 0xf6, 0xff},
	{0x34, 0xd3, 0x99, 0xff},
	{0xfb, 0xbf, 0x24, 0xff},
	{0xa8, 0x55, 0xf7, 0xff},
	{0x0e, 0xa5, 0xe9, 0xff},
}

func fillRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	dst.SubImage(image.Rect(x, y, x+w, y+h)).(*ebiten.Image).Fill(c)
}

func (b *Board) Draw(screen *ebiten.Image) {
	ox, oy := b.origin()
	for i, c := range b.cards {
		x := ox + (i%b.cols)*cellW + (cellW-cardW)/2
		y := oy + (i/b.cols)*cellH + (cellH-cardH)/2

		switch c.State {
		case Buried:
			fillRect(screen, x, y, cardW, cardH, color.RGBA{0x5a, 0x3d, 0x24, 0xff})
			fillRect(screen, x+4, y+4, cardW-8, 8, color.RGBA{0x7a, 0x55, 0x33, 0xff})
			fillRect(screen, x+8, y+20, 4, 4, color.RGBA{0x3f, 0x2a, 0x18, 0xff})
			fillRect(screen, x+22, y+26, 4, 4, color.RGBA{0x3f, 0x2a, 0x18, 0xff})
		case Revealed:
			fillRect(screen, x, y, cardW, cardH, color.RGBA{0xf3, 0xf4, 0xf6, 0xff})
			fillRect(screen, x+2, y+2, cardW-4, cardH-4, faceColors[c.Face%len(faceColors)])
			// Pips as well as colour, so pairs are distinguishable without colour vision.
			for k := 0; k <= c.Face; k++ {
				fillRect(screen, x+6+(k%3)*10, y+6+(k/3)*10, 6, 6, color.RGBA{0x0a, 0x0e, 0x27, 0xff})
			}
		case Matched:
			fillRect(screen, x+6, y+6, cardW-12, cardH-12, color.RGBA{0x12, 0x0c, 0x08, 0xff})
		}
	}
}
