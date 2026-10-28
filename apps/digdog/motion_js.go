//go:build js && wasm

package main

import "syscall/js"

func isReducedMotionPreferred() bool {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return false
	}
	matchMedia := window.Get("matchMedia")
	if !matchMedia.Truthy() {
		return false
	}
	res := matchMedia.Invoke("(prefers-reduced-motion: reduce)")
	if !res.Truthy() {
		return false
	}
	return res.Get("matches").Bool()
}
