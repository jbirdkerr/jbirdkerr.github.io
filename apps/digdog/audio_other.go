//go:build !linux && !(js && wasm)

package main

func isAudioSupported() bool {
	return true
}
