package main

import (
	"os"
	"path/filepath"
	goruntime "runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application menu
	appMenu := createMenu(app)

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Tinkerdown Desktop",
		Width:            1280,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		DisableResize:    false,
		Fullscreen:       false,
		Frameless:        false,
		StartHidden:      false,
		HideWindowOnClose: false,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Menu:             appMenu,
		AssetServer: &assetserver.Options{
			Handler: app.GetHandler(),
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []any{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			About: &mac.AboutInfo{
				Title:   "Tinkerdown Desktop",
				Message: "Interactive documentation and markdown app viewer.\n\nBuilt with Wails and Go.",
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
		os.Exit(1)
	}
}

func createMenu(app *App) *menu.Menu {
	appMenu := menu.NewMenu()

	// File menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("Open File...", keys.CmdOrCtrl("o"), func(cd *menu.CallbackData) {
		app.OpenFile()
	})
	fileMenu.AddText("Open Directory...", keys.CmdOrCtrl("shift+o"), func(cd *menu.CallbackData) {
		app.OpenDirectory()
	})
	fileMenu.AddSeparator()

	// Add recent files placeholder (could be enhanced later)
	recentMenu := fileMenu.AddSubmenu("Open Recent")
	recentMenu.AddText("Clear Recent", nil, func(cd *menu.CallbackData) {})

	fileMenu.AddSeparator()

	if goruntime.GOOS != "darwin" {
		fileMenu.AddText("Exit", keys.OptionOrAlt("F4"), func(cd *menu.CallbackData) {
			os.Exit(0)
		})
	}

	// Edit menu (standard on macOS)
	if goruntime.GOOS == "darwin" {
		editMenu := appMenu.AddSubmenu("Edit")
		editMenu.AddText("Undo", keys.CmdOrCtrl("z"), nil)
		editMenu.AddText("Redo", keys.CmdOrCtrl("shift+z"), nil)
		editMenu.AddSeparator()
		editMenu.AddText("Cut", keys.CmdOrCtrl("x"), nil)
		editMenu.AddText("Copy", keys.CmdOrCtrl("c"), nil)
		editMenu.AddText("Paste", keys.CmdOrCtrl("v"), nil)
		editMenu.AddText("Select All", keys.CmdOrCtrl("a"), nil)
	}

	// View menu
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Reload", keys.CmdOrCtrl("r"), func(cd *menu.CallbackData) {
		// Wails handles reload
	})
	viewMenu.AddText("Toggle Full Screen", keys.Key("F11"), func(cd *menu.CallbackData) {
		// Wails handles fullscreen
	})
	viewMenu.AddSeparator()
	viewMenu.AddText("Developer Tools", keys.CmdOrCtrl("shift+i"), func(cd *menu.CallbackData) {
		// Open dev tools
	})

	// Help menu
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("Documentation", nil, func(cd *menu.CallbackData) {
		// Open docs
	})
	helpMenu.AddText("Report Issue", nil, func(cd *menu.CallbackData) {
		// Open issue tracker
	})
	if goruntime.GOOS != "darwin" {
		helpMenu.AddSeparator()
		helpMenu.AddText("About Tinkerdown", nil, func(cd *menu.CallbackData) {
			// Show about dialog
		})
	}

	return appMenu
}

// GetHomeDirectory returns the user's home directory.
func GetHomeDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// GetDefaultDirectory returns a sensible default directory.
func GetDefaultDirectory() string {
	home := GetHomeDirectory()
	// Check for common locations
	docsDir := filepath.Join(home, "Documents")
	if _, err := os.Stat(docsDir); err == nil {
		return docsDir
	}
	return home
}
