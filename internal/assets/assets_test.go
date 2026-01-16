package assets

import (
	"testing"
)

func TestGetClientJS(t *testing.T) {
	data, err := GetClientJS()
	if err != nil {
		t.Fatalf("GetClientJS failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetClientJS returned empty data")
	}
}

func TestGetClientCSS(t *testing.T) {
	data, err := GetClientCSS()
	if err != nil {
		t.Fatalf("GetClientCSS failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetClientCSS returned empty data")
	}
}

func TestGetPrismJS(t *testing.T) {
	data, err := GetPrismJS()
	if err != nil {
		t.Fatalf("GetPrismJS failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetPrismJS returned empty data")
	}
}

func TestGetPrismCSS(t *testing.T) {
	data, err := GetPrismCSS()
	if err != nil {
		t.Fatalf("GetPrismCSS failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetPrismCSS returned empty data")
	}
}

func TestGetPrismLanguage_ValidLanguages(t *testing.T) {
	for lang := range SupportedPrismLanguages {
		t.Run(lang, func(t *testing.T) {
			data, err := GetPrismLanguage(lang)
			if err != nil {
				t.Fatalf("GetPrismLanguage(%q) failed: %v", lang, err)
			}
			if len(data) == 0 {
				t.Errorf("GetPrismLanguage(%q) returned empty data", lang)
			}
		})
	}
}

func TestGetPrismLanguage_InvalidLanguage(t *testing.T) {
	invalidLanguages := []string{
		"invalid",
		"python",
		"../../../etc/passwd",
		"go/../../secret",
		"",
	}

	for _, lang := range invalidLanguages {
		t.Run(lang, func(t *testing.T) {
			_, err := GetPrismLanguage(lang)
			if err == nil {
				t.Errorf("GetPrismLanguage(%q) should have returned an error", lang)
			}
		})
	}
}

func TestGetMermaidJS(t *testing.T) {
	data, err := GetMermaidJS()
	if err != nil {
		t.Fatalf("GetMermaidJS failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetMermaidJS returned empty data")
	}
}

func TestGetPicoCSS(t *testing.T) {
	data, err := GetPicoCSS()
	if err != nil {
		t.Fatalf("GetPicoCSS failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("GetPicoCSS returned empty data")
	}
}

func TestClientFS(t *testing.T) {
	fsys := ClientFS()
	if fsys == nil {
		t.Fatal("ClientFS returned nil")
	}

	// Verify we can read a file from the fs
	file, err := fsys.Open("tinkerdown-client.browser.js")
	if err != nil {
		t.Fatalf("Failed to open file from ClientFS: %v", err)
	}
	file.Close()
}
