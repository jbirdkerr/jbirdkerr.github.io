package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jbirdkerr/jbirdkerr.github.io/apps/digdog/internal/board"
)

const (
	cellW = 44
	cellH = 44
	cardW = 36
	cardH = 36
)

// boardOrigin returns the top-left pixel of the board grid.
func boardOrigin(b *board.Board) (int, int) {
	// Nudged up a little to leave room for the dog at the bottom.
	return (screenW - b.Cols()*cellW) / 2, (screenH-b.Rows()*cellH)/2 - 8
}

// cellAt returns the card index under the point (x, y) in screen coordinates, or -1.
func cellAt(b *board.Board, x, y int) int {
	ox, oy := boardOrigin(b)
	cx, cy := x-ox, y-oy
	if cx < 0 || cy < 0 {
		return -1
	}
	col, row := cx/cellW, cy/cellH
	if col >= b.Cols() || row >= b.Rows() {
		return -1
	}
	return row*b.Cols() + col
}

// cellCenter returns the pixel centre of card i.
func cellCenter(b *board.Board, i int) (float64, float64) {
	ox, oy := boardOrigin(b)
	return float64(ox + (i%b.Cols())*cellW + cellW/2),
		float64(oy + (i/b.Cols())*cellH + cellH/2)
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

func drawBoard(screen *ebiten.Image, b *board.Board) {
	ox, oy := boardOrigin(b)
	for i := 0; i < b.Len(); i++ {
		c := b.Card(i)
		x := ox + (i%b.Cols())*cellW + (cellW-cardW)/2
		y := oy + (i/b.Cols())*cellH + (cellH-cardH)/2

		switch c.State {
		case board.Buried:
			fillRect(screen, x, y, cardW, cardH, color.RGBA{0x5a, 0x3d, 0x24, 0xff})
			fillRect(screen, x+4, y+4, cardW-8, 8, color.RGBA{0x7a, 0x55, 0x33, 0xff})
			fillRect(screen, x+8, y+20, 4, 4, color.RGBA{0x3f, 0x2a, 0x18, 0xff})
			fillRect(screen, x+22, y+26, 4, 4, color.RGBA{0x3f, 0x2a, 0x18, 0xff})
		case board.Revealed:
			fillRect(screen, x, y, cardW, cardH, color.RGBA{0xf3, 0xf4, 0xf6, 0xff})
			fillRect(screen, x+2, y+2, cardW-4, cardH-4, faceColors[c.Face%len(faceColors)])
			// Pips as well as colour, so pairs are distinguishable without colour vision.
			for k := 0; k <= c.Face; k++ {
				fillRect(screen, x+6+(k%3)*10, y+6+(k/3)*10, 6, 6, color.RGBA{0x0a, 0x0e, 0x27, 0xff})
			}
		case board.Matched:
			fillRect(screen, x+6, y+6, cardW-12, cardH-12, color.RGBA{0x12, 0x0c, 0x08, 0xff})
		}
	}
}
