// Package main generates an animated pixel-art Red Poodle sprite sheet for DigDog.
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

	// Rich Red Poodle Palette (Cinnamon / Copper / Auburn / Apricot)
	redCoat := color.NRGBA{0xc4, 0x5a, 0x22, 0xff}   // Primary copper-red coat
	redLight := color.NRGBA{0xe6, 0x82, 0x48, 0xff}  // Sunlit curls / highlights (apricot)
	redDark := color.NRGBA{0x8f, 0x36, 0x12, 0xff}   // Shadow curls / creases (deep auburn)
	redDeep := color.NRGBA{0x66, 0x22, 0x0a, 0xff}   // Darkest outline / ear shadow
	noseCol := color.NRGBA{0x1c, 0x12, 0x0c, 0xff}   // Black truffle nose & dark eye
	eyeShine := color.NRGBA{0xff, 0xff, 0xff, 0xff}  // White sparkle
	tongueCol := color.NRGBA{0xf4, 0x72, 0xb6, 0xff} // Pink tongue

	px := func(x, y int, c color.NRGBA) {
		if x >= 0 && x < imgW && y >= 0 && y < imgH {
			img.SetNRGBA(x, y, c)
		}
	}

	fill := func(x, y, w, h int, c color.NRGBA) {
		for dy := 0; dy < h; dy++ {
			for dx := 0; dx < w; dx++ {
				px(x+dx, y+dy, c)
			}
		}
	}

	frameCounts := []int{4, 4, 3, 3, 2}

	for row := 0; row < rows; row++ {
		numFrames := frameCounts[row]
		for col := 0; col < numFrames; col++ {
			fx := col * frameW
			fy := row * frameH

			yOff := 0
			headYOff := 0
			headXOff := 0
			tailPhase := col % 4
			pawPhase := col % 2
			isHappy := (row == 3)
			isSad := (row == 4)
			isDig := (row == 1)

			switch row {
			case 0: // idle: breathing & tail wag
				if col == 1 || col == 3 {
					yOff = -1
				}
				tailPhase = col
			case 1: // dig: crouch forward, paws alternate digging
				headYOff = 3
				headXOff = 2
				tailPhase = 0
			case 2: // sniff: head right down to ground
				headYOff = 6
				headXOff = 1
				tailPhase = 1
			case 3: // happy: playful bounce
				if col == 1 {
					yOff = -3
				} else if col == 2 {
					yOff = -1
				}
				tailPhase = (col * 2) % 4
			case 4: // sad: drooping posture
				headYOff = 2
				tailPhase = 3
			}

			// --- Tail with Poodle Pom-Pom ---
			// Tail stem curves upward from rump (x: 6, y: 15)
			switch tailPhase {
			case 0: // high & arched forward
				fill(fx+6, fy+13+yOff, 2, 3, redDark)
				fill(fx+5, fy+10+yOff, 2, 3, redCoat)
				// Fluffy pom-pom on top
				fill(fx+4, fy+7+yOff, 4, 4, redCoat)
				fill(fx+5, fy+6+yOff, 3, 2, redLight)
				fill(fx+4, fy+9+yOff, 4, 2, redDark)
			case 1: // upright
				fill(fx+5, fy+13+yOff, 2, 4, redDark)
				fill(fx+5, fy+10+yOff, 2, 3, redCoat)
				// Pom-pom
				fill(fx+3, fy+8+yOff, 5, 4, redCoat)
				fill(fx+4, fy+7+yOff, 3, 2, redLight)
				fill(fx+4, fy+11+yOff, 4, 1, redDark)
			case 2: // arched back
				fill(fx+5, fy+14+yOff, 3, 3, redDark)
				fill(fx+3, fy+12+yOff, 3, 2, redCoat)
				// Pom-pom
				fill(fx+2, fy+10+yOff, 4, 4, redCoat)
				fill(fx+3, fy+9+yOff, 3, 2, redLight)
				fill(fx+2, fy+13+yOff, 4, 1, redDark)
			case 3: // lowered (sad / resting)
				fill(fx+5, fy+15+yOff, 2, 3, redDark)
				fill(fx+4, fy+17+yOff, 3, 3, redCoat)
				fill(fx+3, fy+19+yOff, 3, 3, redDark)
			}

			// --- Curly Poodle Body ---
			by := fy + 13 + yOff
			// Main body core
			fill(fx+8, by+1, 11, 8, redCoat)
			fill(fx+7, by+2, 1, 6, redDark) // rump curve
			// Curl clusters (texture bumps on coat)
			px(fx+8, by, redLight)
			px(fx+10, by, redLight)
			px(fx+13, by, redLight)
			px(fx+15, by, redLight)
			fill(fx+9, by+1, 2, 1, redLight)
			fill(fx+12, by+1, 2, 1, redLight)
			fill(fx+10, by+4, 3, 2, redDark) // deep curl shadow
			fill(fx+14, by+5, 2, 2, redDark)

			// Poodle Chest / Front Ruff (voluminous curly puff)
			fill(fx+15, by+1, 5, 8, redCoat)
			fill(fx+17, by, 3, 2, redLight)
			fill(fx+18, by+2, 3, 4, redLight)
			fill(fx+15, by+7, 4, 2, redDark)

			// --- Back Leg & Fluffy Boot ---
			// Hip
			fill(fx+8, fy+18+yOff, 4, 4, redCoat)
			fill(fx+7, fy+19+yOff, 2, 3, redDark)
			// Slender hock
			fill(fx+9, fy+22+yOff, 2, 4, redCoat)
			// Poodle curly foot puff (pom-pom boot)
			fill(fx+7, fy+25+yOff, 5, 4, redCoat)
			fill(fx+8, fy+25+yOff, 3, 1, redLight)
			fill(fx+7, fy+28+yOff, 5, 1, redDark)

			// --- Front Leg & Fluffy Boot ---
			fLegY := fy + 21 + yOff
			if isDig {
				if pawPhase == 0 {
					fLegY += 2
				} else {
					fLegY -= 1
				}
			}
			// Slender leg
			fill(fx+17, fLegY, 2, 4, redCoat)
			// Poodle curly front boot
			fill(fx+16, fLegY+3, 5, 4, redCoat)
			fill(fx+17, fLegY+3, 3, 1, redLight)
			fill(fx+16, fLegY+6, 5, 1, redDeep)

			// Digging dirt specks flying from paws
			if isDig && pawPhase == 0 {
				px(fx+22, fy+27, color.NRGBA{0x7a, 0x48, 0x22, 0xff})
				px(fx+24, fy+25, color.NRGBA{0x5a, 0x33, 0x18, 0xff})
			}

			// --- Head, Topknot, Ears & Muzzle ---
			hx := fx + 17 + headXOff
			hy := fy + 6 + yOff + headYOff

			// Poodle Topknot (the signature curly puff on top of the head)
			fill(hx+2, hy-2, 6, 4, redCoat)
			fill(hx+3, hy-3, 4, 2, redLight) // topknot crest highlight
			px(hx+1, hy-1, redCoat)
			px(hx+8, hy-1, redCoat)
			fill(hx+3, hy+1, 4, 1, redDark) // crease under topknot

			// Head core
			fill(hx+1, hy+1, 8, 7, redCoat)

			// Long Fluffy Poodle Ear (hangs down beside cheek, wide curly bell at bottom)
			earY := hy + 1
			if isSad {
				earY += 2 // droop
			}
			fill(hx-1, earY, 3, 4, redDark)
			fill(hx-2, earY+3, 4, 5, redDark) // fluffy lower ear bell
			fill(hx-1, earY+4, 2, 3, redDeep) // inner ear shadow
			fill(hx-2, earY+7, 4, 1, redDeep)

			// Slender Poodle Muzzle (fine tapered snout)
			fill(hx+7, hy+3, 5, 4, redCoat)
			fill(hx+8, hy+3, 4, 2, redLight) // bridge of snout
			// Dark nose leather at tip
			fill(hx+11, hy+3, 2, 2, noseCol)

			// Expressive Eye
			if isHappy {
				// Happy squint eye ^^
				fill(hx+5, hy+2, 2, 1, noseCol)
				px(hx+4, hy+3, noseCol)
				px(hx+7, hy+3, noseCol)
				// Cute little tongue
				fill(hx+9, hy+6, 2, 2, tongueCol)
			} else if isSad {
				// Drooping eye
				fill(hx+5, hy+3, 2, 1, noseCol)
				px(hx+4, hy+2, noseCol)
			} else {
				// Dark round eye with white shine
				fill(hx+5, hy+2, 2, 2, noseCol)
				px(hx+6, hy+2, eyeShine)
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
