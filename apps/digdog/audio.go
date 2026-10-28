package main

import (
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	soundSampleRate = 44100
	defaultVolume   = 0.35
)

type AudioSystem struct {
	ctx         *audio.Context
	initialized bool
	available   bool
	muted       bool

	sndDig    []byte
	sndReveal []byte
	sndMatch  []byte
	sndMiss   []byte
	sndWin    []byte
}

func newAudioSystem() *AudioSystem {
	a := &AudioSystem{
		available: isAudioSupported(),
		muted:     false,
	}
	// Pre-synthesize the sound waveforms into PCM bytes
	a.sndDig = a.synthDig()
	a.sndReveal = a.synthReveal()
	a.sndMatch = a.synthMatch()
	a.sndMiss = a.synthMiss()
	a.sndWin = a.synthWin()
	return a
}

// ensureInit starts the audio context on the first user gesture (click, tap, or key).
func (a *AudioSystem) ensureInit() {
	if a.initialized {
		return
	}
	a.initialized = true
	if !a.available {
		a.muted = true
		return
	}
	a.ctx = audio.NewContext(soundSampleRate)
}

func (a *AudioSystem) ToggleMute() {
	if !a.available {
		return
	}
	a.muted = !a.muted
}

func (a *AudioSystem) IsMuted() bool {
	return a.muted
}

func (a *AudioSystem) IsAvailable() bool {
	return a.available
}

func (a *AudioSystem) play(pcm []byte) {
	if a.muted {
		return
	}
	a.ensureInit()
	if a.ctx == nil || len(pcm) == 0 {
		return
	}
	player := a.ctx.NewPlayerFromBytes(pcm)
	player.SetVolume(defaultVolume)
	player.Play()
}

func (a *AudioSystem) PlayDig() {
	a.play(a.sndDig)
}

func (a *AudioSystem) PlayReveal() {
	a.play(a.sndReveal)
}

func (a *AudioSystem) PlayMatch() {
	a.play(a.sndMatch)
}

func (a *AudioSystem) PlayMiss() {
	a.play(a.sndMiss)
}

func (a *AudioSystem) PlayWin() {
	a.play(a.sndWin)
}

// synthPCM produces 16-bit signed stereo little-endian PCM data.
func synthPCM(durationSec float64, fn func(t float64) float64) []byte {
	numSamples := int(durationSec * soundSampleRate)
	buf := make([]byte, numSamples*4)
	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(soundSampleRate)
		val := fn(t)
		if val > 1.0 {
			val = 1.0
		} else if val < -1.0 {
			val = -1.0
		}
		s := int16(val * 32767)
		// Left channel
		buf[i*4+0] = byte(s)
		buf[i*4+1] = byte(s >> 8)
		// Right channel
		buf[i*4+2] = byte(s)
		buf[i*4+3] = byte(s >> 8)
	}
	return buf
}

// synthDig creates a short crisp scratching dirt sound.
func (a *AudioSystem) synthDig() []byte {
	dur := 0.08
	rng := rand.New(rand.NewSource(42))
	return synthPCM(dur, func(t float64) float64 {
		env := math.Pow(1.0-t/dur, 1.5)
		noise := rng.Float64()*2.0 - 1.0
		lowRumble := math.Sin(2 * math.Pi * 85.0 * t)
		return (noise*0.7 + lowRumble*0.3) * env
	})
}

// synthReveal creates a gentle upward frequency sweep (whoosh/flip).
func (a *AudioSystem) synthReveal() []byte {
	dur := 0.09
	phase := 0.0
	return synthPCM(dur, func(t float64) float64 {
		freq := 320.0 + 380.0*(t/dur)
		phase += 2 * math.Pi * freq / soundSampleRate
		env := math.Sin(math.Pi * (t / dur))
		return math.Sin(phase) * env * 0.7
	})
}

// synthMatch creates a warm 3-tone ascending major chord chime.
func (a *AudioSystem) synthMatch() []byte {
	dur := 0.35
	phase1, phase2, phase3 := 0.0, 0.0, 0.0
	f1, f2, f3 := 523.25, 659.25, 783.99 // C5, E5, G5
	return synthPCM(dur, func(t float64) float64 {
		phase1 += 2 * math.Pi * f1 / soundSampleRate
		phase2 += 2 * math.Pi * f2 / soundSampleRate
		phase3 += 2 * math.Pi * f3 / soundSampleRate

		v1 := math.Sin(phase1) * math.Exp(-12.0*math.Max(0, t))
		v2 := 0.0
		if t >= 0.08 {
			v2 = math.Sin(phase2) * math.Exp(-10.0*(t-0.08))
		}
		v3 := 0.0
		if t >= 0.16 {
			v3 = math.Sin(phase3) * math.Exp(-8.0*(t-0.16))
		}
		return (v1*0.45 + v2*0.45 + v3*0.45) * 0.8
	})
}

// synthMiss creates a soft low descending two-tone bonk.
func (a *AudioSystem) synthMiss() []byte {
	dur := 0.22
	phase1, phase2 := 0.0, 0.0
	f1, f2 := 220.0, 164.81 // A3, E3
	return synthPCM(dur, func(t float64) float64 {
		phase1 += 2 * math.Pi * f1 / soundSampleRate
		phase2 += 2 * math.Pi * f2 / soundSampleRate

		v1 := math.Sin(phase1) * math.Exp(-14.0*math.Max(0, t))
		v2 := 0.0
		if t >= 0.09 {
			v2 = math.Sin(phase2) * math.Exp(-12.0*(t-0.09))
		}
		return (v1*0.5 + v2*0.5) * 0.7
	})
}

// synthWin creates a cheerful victory arpeggio (C5 -> E5 -> G5 -> C6).
func (a *AudioSystem) synthWin() []byte {
	dur := 0.65
	phases := make([]float64, 4)
	freqs := []float64{523.25, 659.25, 783.99, 1046.50} // C5, E5, G5, C6
	starts := []float64{0.0, 0.10, 0.20, 0.30}

	return synthPCM(dur, func(t float64) float64 {
		total := 0.0
		for k, f := range freqs {
			phases[k] += 2 * math.Pi * f / soundSampleRate
			if t >= starts[k] {
				lt := t - starts[k]
				total += math.Sin(phases[k]) * math.Exp(-5.5*lt)
			}
		}
		return total * 0.35
	})
}
