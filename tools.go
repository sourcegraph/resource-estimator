//go:build tools
// +build tools

package main

import (
	// Used to serve the WASM bundle while developing
	_ "github.com/hajimehoshi/wasmserve"
)
