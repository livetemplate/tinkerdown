//go:build !race

package server

// raceDetector is true when the tests run with -race; see race_on_test.go.
const raceDetector = false
