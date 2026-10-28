// Package main generates an animated pixel-art dog sprite sheet for DigDog.
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
		frameW = 32
		frameH = 32
		cols   = 4
		rows   = 5
		imgW   = frameW * cols
		imgH   = frameH * rows
	)

	img := image.NewNRGBA(image.Rect(0, 0, imgW, imgH))

	// Colors for Henry the dog
	fur := color.NRGBA{0xc8, 0x82, 0x3a, 0xff}       // warm golden tan
	furDark := color.NRGBA{0x96, 0x56, 0x20, 0xff}   // dark brown ear & saddle
	furLight := color.NRGBA{0xfa, 0xeb, 0xd7, 0xff}  // cream chest & paws
	noseCol := color.NRGBA{0x2b, 0x1d, 0x14, 0xff}   // dark nose & eyes
	tongueCol := color.NRGBA{0xf4, 0x72, 0xb6, 0xff} // pink tongue
	white := color.NRGBA{0xff, 0xff, 0xff, 0xff}

	fill := func(x, y, w, h int, c color.NRGBA) {
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				img.SetNRGBA(x+dx, y+dy, c)
			}
		}
	}

	// Frame counts per row
	frameCounts := []int{4, 4, 3, 3, 2}

	for row := 0; row < rows; row++ {
		numFrames := frameCounts[row]
		for col := 0; col < numFrames; col++ {
			fx := col * frameW
			fy := row * frameH

			// Vertical bounce or offset for animations
			yOff := 0
			headYOff := 0
			tailPos := 0 // 0=up, 1=mid, 2=down
			pawsExtended := false

			switch row {
			case 0: // idle (tail wag)
				tailPos = col % 3
				if col == 1 || col == 3 {
					yOff = 0
				}
			case 1: // dig (paws moving down, head low)
				headYOff = 4
				pawsExtended = (col%2 == 1)
				tailPos = 1
			case 2: // sniff (head down to ground)
				headYOff = 5 + (col % 2)
				tailPos = 1
			case 3: // happy (bounce up)
				if col == 1 {
					yOff = -3
				} else if col == 2 {
					yOff = -1
				}
				tailPos = 0
			case 4: // sad (head drooping, tail low)
				headYOff = 3
				tailPos = 2
			}

			// --- Body ---
			fill(fx+8, fy+13+yOff, 12, 9, fur)
			fill(fx+10, fy+13+yOff, 7, 5, furDark)  // saddle patch
			fill(fx+16, fy+16+yOff, 4, 6, furLight) // belly/chest highlight

			// --- Legs & Paws ---
			// Back leg
			fill(fx+8, fy+22+yOff, 3, 5, fur)
			fill(fx+7, fy+26+yOff, 4, 2, furLight)
			// Front leg
			frontLegY := fy + 22 + yOff
			if pawsExtended {
				frontLegY += 2
			}
			fill(fx+17, frontLegY, 3, 5, fur)
			fill(fx+16, frontLegY+4, 4, 2, furLight)

			// --- Head & Muzzle ---
			hy := fy + 7 + yOff + headYOff
			fill(fx+17, hy, 8, 8, fur)
			fill(fx+23, hy+3, 5, 4, furLight) // snout
			fill(fx+26, hy+3, 2, 2, noseCol)  // nose tip

			// Eye
			if row == 3 {
				// Happy squint eye ^^
				fill(fx+21, hy+2, 2, 1, noseCol)
				fill(fx+23, hy+5, 2, 2, tongueCol) // tongue out!
			} else if row == 4 {
				// Sad droopy eye
				fill(fx+21, hy+3, 2, 1, noseCol)
			} else {
				// Normal eye
				fill(fx+21, hy+2, 2, 2, noseCol)
				fill(fx+22, hy+2, 1, 1, white) // sparkle
			}

			// Ear (floppy)
			earY := hy - 1
			if row == 4 {
				earY += 3 // drooping
			}
			fill(fx+15, earY, 3, 6, furDark)

			// --- Tail ---
			switch tailPos {
			case 0: // up
				fill(fx+5, fy+8+yOff, 4, 2, fur)
				fill(fx+7, fy+10+yOff, 2, 4, fur)
			case 1: // mid
				fill(fx+4, fy+12+yOff, 5, 2, fur)
			case 2: // down
				fill(fx+5, fy+15+yOff, 2, 4, fur)
				fill(fx+4, fy+18+yOff, 2, 3, fur)
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
