//go:build linux && !js

package main

import (
	"os"
)

func isAudioSupported() bool {
	if os.Getenv("DIGDOG_NO_AUDIO") == "1" {
		return false
	}
	// Check if ALSA has sound cards registered
	if data, err := os.ReadFile("/proc/asound/cards"); err == nil && len(data) > 0 {
		return true
	}
	// Check for active PulseAudio / PipeWire socket
	if runtimeDir := os.Getenv("XDG_RUNTIME_DIR"); runtimeDir != "" {
		if _, err := os.Stat(runtimeDir + "/pulse/native"); err == nil {
			return true
		}
		if _, err := os.Stat(runtimeDir + "/pipewire-0"); err == nil {
			return true
		}
	}
	return false
}
