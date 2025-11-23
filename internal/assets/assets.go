// Package assets embeds the client JavaScript and CSS files
package assets

import (
	"embed"
	"io/fs"
)

//go:embed client/*
var clientFS embed.FS

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
	return clientFS.ReadFile("client/livepage-client.browser.js")
}

// GetClientCSS returns the browser CSS bundle
func GetClientCSS() ([]byte, error) {
	return clientFS.ReadFile("client/livepage-client.browser.css")
}
