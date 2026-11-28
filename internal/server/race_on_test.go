//go:build race

package server

// raceDetector is true when the tests run with -race, which makes code
// several times slower, so timing checks allow for it.
const raceDetector = true
