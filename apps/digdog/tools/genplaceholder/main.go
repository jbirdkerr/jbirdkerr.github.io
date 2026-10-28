// Package main generates a placeholder dog.png sprite sheet for DigDog.
// 128x160 pixels, 32x32 frames, 4 columns x 5 rows.
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	if err := os.MkdirAll("assets", 0o755); err != nil {
		panic(err)
	}

	const (
		frameW  = 32
		frameH  = 32
		cols    = 4
		rows    = 5
		imgW    = frameW * cols // 128
		imgH    = frameH * rows // 160
		dotSize = 4
	)

	type rowDef struct {
		color  color.NRGBA
		frames int
	}

	rowDefs := []rowDef{
		{color.NRGBA{139, 90, 43, 255}, 4},  // row 0: idle, brown
		{color.NRGBA{255, 140, 0, 255}, 4},  // row 1: dig, orange
		{color.NRGBA{34, 139, 34, 255}, 3},  // row 2: sniff, green
		{color.NRGBA{255, 215, 0, 255}, 3},  // row 3: happy, yellow
		{color.NRGBA{70, 130, 180, 255}, 2}, // row 4: sad, blue
	}

	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))

	for row, rd := range rowDefs {
		// Darken the fill color for the border.
		border := color.NRGBA{
			R: saturatingSub(rd.color.R, 60),
			G: saturatingSub(rd.color.G, 60),
			B: saturatingSub(rd.color.B, 60),
			A: 255,
		}

		for col := 0; col < cols; col++ {
			x0 := col * frameW
			y0 := row * frameH
			x1 := x0 + frameW
			y1 := y0 + frameH

			if col >= rd.frames {
				// Unused cell: leave fully transparent (zero value).
				continue
			}

			// Fill frame with row color.
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					img.SetNRGBA(x, y, rd.color)
				}
			}

			// Draw 1-pixel border.
			for x := x0; x < x1; x++ {
				img.SetNRGBA(x, y0, border)
				img.SetNRGBA(x, y1-1, border)
			}
			for y := y0; y < y1; y++ {
				img.SetNRGBA(x0, y, border)
				img.SetNRGBA(x1-1, y, border)
			}

			// Draw (col+1) white 4x4 dots near the top-left.
			white := color.NRGBA{255, 255, 255, 255}
			numDots := col + 1
			for d := 0; d < numDots; d++ {
				dotX := x0 + 2 + d*(dotSize+1)
				dotY := y0 + 2
				for dy := 0; dy < dotSize; dy++ {
					for dx := 0; dx < dotSize; dx++ {
						img.SetNRGBA(dotX+dx, dotY+dy, white)
					}
				}
			}
		}
	}

	f, err := os.Create("assets/dog.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// saturatingSub subtracts b from a, clamping to 0.
func saturatingSub(a, b uint8) uint8 {
	if b > a {
		return 0
	}
	return a - b
}
