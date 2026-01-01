//go:build tinygo.wasm

// Package main implements a tinkerdown WASM data source.
// Build with: tinygo build -o [[.ProjectName]].wasm -target wasi source.go
package main

import (
	"encoding/json"
	"os"
	"unsafe"
)

// Global state for result passing
var lastResult []byte
var lastError string

// main is required but not used
func main() {}

// fetch is called by tinkerdown to get data.
// It must return a pointer to a JSON array.
//
// Safety: The unsafe.Pointer usage here is required for WASM host interop.
// The pointer remains valid because lastResult is a package-level variable
// that persists until free_result() is called by the host.
//
//export fetch
func fetch() int32 {
	// Get configuration from environment (set via source options)
	category := os.Getenv("category")
	if category == "" {
		category = "default"
	}

	// Your data fetching logic here
	// This example returns static data - replace with your API calls
	data := []map[string]interface{}{
		{
			"id":       1,
			"title":    "Item One",
			"category": category,
		},
		{
			"id":       2,
			"title":    "Item Two",
			"category": category,
		},
		{
			"id":       3,
			"title":    "Item Three",
			"category": category,
		},
	}

	// Serialize to JSON
	result, err := json.Marshal(data)
	if err != nil {
		lastError = err.Error()
		lastResult = nil
		return 0
	}

	lastResult = result
	lastError = ""
	if len(lastResult) == 0 {
		return 0
	}
	return int32(uintptr(unsafe.Pointer(&lastResult[0])))
}

// get_result_len returns the length of the last fetch result.
//
//export get_result_len
func get_result_len() int32 {
	return int32(len(lastResult))
}

// free_result frees memory from the last result.
//
//export free_result
func free_result() {
	lastResult = nil
	lastError = ""
}

// get_error returns a pointer to the error string if fetch failed.
//
// Safety: Uses unsafe.StringData which returns a pointer to the string's
// underlying bytes. Valid while lastError is unchanged.
//
//export get_error
func get_error() int32 {
	if lastError == "" {
		return 0
	}
	return int32(uintptr(unsafe.Pointer(unsafe.StringData(lastError))))
}

// get_error_len returns the length of the error string.
//
//export get_error_len
func get_error_len() int32 {
	return int32(len(lastError))
}

// write handles write operations (optional).
// action: the action name (e.g., "Add", "Delete", "Update")
// data: JSON object with the data to write
// Returns 0 on success, non-zero on error.
//
//export write
func write(actionPtr, actionLen, dataPtr, dataLen int32) int32 {
	// Uncomment to enable write operations:
	//
	// action := ptrToString(actionPtr, actionLen)
	// data := ptrToString(dataPtr, dataLen)
	//
	// switch action {
	// case "Add":
	//     // Handle add
	// case "Delete":
	//     // Handle delete
	// case "Update":
	//     // Handle update
	// default:
	//     lastError = "unknown action: " + action
	//     return 1
	// }

	lastError = "write not implemented"
	return 1
}

// Helper to convert pointer to string (for write operations)
func ptrToString(ptr, length int32) string {
	if ptr == 0 || length == 0 {
		return ""
	}
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(ptr))), length)
	return string(bytes)
}
