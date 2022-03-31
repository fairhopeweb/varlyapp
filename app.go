package main

import (
	"context"
	"encoding/base64"
	"fmt"
	f "io/fs"
	"os"

	"github.com/varlyapp/varlyapp/backend/fs"
	"github.com/varlyapp/varlyapp/backend/nft"
	"github.com/varlyapp/varlyapp/backend/settings"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	Docs     map[string]interface{}
	Settings *settings.Settings
}

// NewApp creates a new App application struct
func NewApp() *App {
	dir, _ := fs.GetApplicationDocumentsDirectory()
	settings, _ := settings.LoadSettings(fmt.Sprint(dir, "settings.json"))

	return &App{
		Settings: settings,
	}
}

// startup is called at application startup
func (app *App) startup(ctx context.Context) {
	// Perform your setup here
	app.ctx = ctx
	// menu := menu.NewMenuFromItems(
	// 	menu.SubMenu("File", menu.NewMenuFromItems(
	// 		menu.Text("Open", keys.CmdOrCtrl("o"), func(cd *menu.CallbackData) {
	// 			fmt.Println(cd)
	// 		}),
	// 		menu.Text("Save", keys.CmdOrCtrl("s"), func(cd *menu.CallbackData) {
	// 			runtime.EventsEmit(ctx, "shortcut.save")
	// 		}),
	// 		menu.Separator(),
	// 		menu.Text("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
	// 			runtime.Quit(ctx)
	// 		}),
	// 	)),
	// )

	// runtime.MenuUpdateApplicationMenu(ctx)
}

// domReady is called after the front-end dom has been loaded
func (app *App) domReady(ctx context.Context) {
	// Add your action here
}

// shutdown is called at application termination
func (app *App) shutdown(ctx context.Context) {
	// Perform your teardown here
}

func (app *App) GetSettings() *settings.Settings {
	return app.Settings
}

func (app *App) SaveSettings(settings *settings.Settings) {
	app.Settings.Documents = append(app.Settings.Documents, settings.Documents...)
	fmt.Println(app.Settings.Documents)

	app.Settings.Save()
}

func (app *App) SaveDocuments(documents []settings.Document) {
	app.Settings.SaveDocuments(documents)
}

func (app *App) OpenDirectoryDialog() string {
	path, _ := runtime.OpenDirectoryDialog(app.ctx, runtime.OpenDialogOptions{})

	return path
}

func (app *App) OpenFileDialog() string {
	path, _ := runtime.OpenFileDialog(app.ctx, runtime.OpenDialogOptions{})

	return path
}

func (app *App) SaveFileDialog() string {
	path, _ := runtime.SaveFileDialog(app.ctx, runtime.SaveDialogOptions{})

	return path
}

func (app *App) GenerateCollectionFromConfig(config nft.NewCollectionConfig) {
	nft.GenerateCollectionFromConfig(app.ctx, config)
}

func (app *App) ReadLayers(dir string) fs.CollectionConfig {
	return nft.ReadLayers(app.ctx, dir)
}

func (app *App) GetApplicationDocumentsDirectory() string {
	path, _ := fs.GetApplicationDocumentsDirectory()

	return path
}

func (app *App) EncodeImage(path string) string {
	image, err := os.ReadFile(path)

	if err != nil {
		return fmt.Sprintf("Error: %s", err)
	}

	encoded := base64.StdEncoding.EncodeToString(image)

	encoded = fmt.Sprintf("data:image/png;base64,%s", encoded)

	return encoded
}

func (app *App) SaveFile(file string, data string) bool {
	err := os.WriteFile(file, []byte(data), os.ModePerm)

	if err != nil {
		return false
	}

	return true
}

func (app *App) GetImageStats(path string) f.FileInfo {
	info, err := os.Stat(path)

	if err != nil {
		fmt.Println("unable to get stat: ", err)
	}
	return info
}

func (app *App) MessageDialog(options runtime.MessageDialogOptions) string {
	res, _ := runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{
		Type: runtime.QuestionDialog,
		Title: options.Title,
		Message: options.Message,
		Buttons: options.Buttons,
		DefaultButton: options.DefaultButton,
	})

	return res
}
