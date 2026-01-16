// Package assets embeds the client JavaScript, CSS, and vendor libraries
package assets

import (
	"embed"
	"io/fs"
)

//go:embed client/*
var clientFS embed.FS

//go:embed vendor/prism/*
var prismFS embed.FS

//go:embed vendor/mermaid/*
var mermaidFS embed.FS

//go:embed vendor/pico/*
var picoFS embed.FS

// ClientFS returns the embedded client files
func ClientFS() fs.FS {
	sub, err := fs.Sub(clientFS, "client")
	if err != nil {
		panic(err)
	}
	return sub
}

// GetClientJS returns the browser JavaScript bundle
func GetClientJS() ([]byte, error) {
	return clientFS.ReadFile("client/tinkerdown-client.browser.js")
}

// GetClientCSS returns the browser CSS bundle
func GetClientCSS() ([]byte, error) {
	return clientFS.ReadFile("client/tinkerdown-client.browser.css")
}

// GetPrismJS returns the Prism.js core library
func GetPrismJS() ([]byte, error) {
	return prismFS.ReadFile("vendor/prism/prism.min.js")
}

// GetPrismCSS returns the Prism.js theme CSS
func GetPrismCSS() ([]byte, error) {
	return prismFS.ReadFile("vendor/prism/prism-tomorrow.min.css")
}

// GetPrismLanguage returns a Prism language component
func GetPrismLanguage(lang string) ([]byte, error) {
	return prismFS.ReadFile("vendor/prism/prism-" + lang + ".min.js")
}

// GetMermaidJS returns the Mermaid.js library
func GetMermaidJS() ([]byte, error) {
	return mermaidFS.ReadFile("vendor/mermaid/mermaid.min.js")
}

// GetPicoCSS returns the Pico CSS framework
func GetPicoCSS() ([]byte, error) {
	return picoFS.ReadFile("vendor/pico/pico.min.css")
}
