//go:build !(js && wasm)

package main

func isReducedMotionPreferred() bool {
	return false
}
