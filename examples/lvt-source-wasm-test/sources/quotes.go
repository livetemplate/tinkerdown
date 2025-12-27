// Example WASM source module for tinkerdown.
// Build with: tinygo build -o quotes.wasm -target wasi quotes.go
//
//go:build tinygo
package main

import (
	"os"
	"unsafe"
)

// Result buffer for returning data to host
var resultBuf []byte

// Quotes data - in a real module this might fetch from an API
var quotes = `[
	{"id": 1, "text": "The only way to do great work is to love what you do.", "author": "Steve Jobs"},
	{"id": 2, "text": "Innovation distinguishes between a leader and a follower.", "author": "Steve Jobs"},
	{"id": 3, "text": "Stay hungry, stay foolish.", "author": "Steve Jobs"},
	{"id": 4, "text": "The future belongs to those who believe in the beauty of their dreams.", "author": "Eleanor Roosevelt"},
	{"id": 5, "text": "It is during our darkest moments that we must focus to see the light.", "author": "Aristotle"}
]`

//export fetch
func fetch() int32 {
	// In a real implementation, you might:
	// - Read config from environment (os.Getenv)
	// - Make HTTP requests
	// - Process/transform data

	// Check for category filter from config
	category := os.Getenv("category")
	_ = category // Could filter quotes by category

	resultBuf = []byte(quotes)
	return int32(uintptr(unsafe.Pointer(&resultBuf[0])))
}

//export get_result_len
func getResultLen() int32 {
	return int32(len(resultBuf))
}

//export free_result
func freeResult() {
	resultBuf = nil
}

// main is required for WASI but not called
func main() {}
