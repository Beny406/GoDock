package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/go-vgo/robotgo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// #cgo LDFLAGS: -lX11
// #include <X11/Xlib.h>
//
// void initXThreads() {
//     XInitThreads();
// }
import "C"

func init() {
	C.initXThreads()
}

// App struct
type App struct {
	ctx           context.Context
	ticker        *time.Ticker
	desktopFiles  []DesktopFile
	screenWidth   int
	screenHeight  int
	windowVisible bool
	x11           *X11Client
	previousId    string
}

type DesktopFileForFE struct {
	Name      string           `json:"name"`
	IconPath  string           `json:"iconPath"`
	Instances []WmCtrlInstance `json:"instances"`
	ExecPath  string           `json:"execPath"`
	WmClass   string           `json:"wmClass"`
}

type DesktopFile struct {
	Name     string `json:"name"`
	IconPath string `json:"iconPath"`
	ExecPath string `json:"execPath"`
	WmClass  string `json:"wmClass"`
}

type WmCtrlApp struct {
	WmClass   string           `json:"wmClass"`
	Instances []WmCtrlInstance `json:"instances"`
}

type WmCtrlInstance struct {
	WindowId string `json:"windowId"`
	Name     string `json:"name"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	logrus.Info("Starting Wails app")
	a.ctx = ctx

	var err error
	a.x11, err = NewX11Client()
	if err != nil {
		logrus.Fatalf("Failed to connect to X11: %v", err)
	}

	a.screenWidth, a.screenHeight = robotgo.GetScreenSize()
	logrus.Infof("Screen size: %dx%d", a.screenWidth, a.screenHeight)
	a.desktopFiles = a.getDesktopFiles()
	logrus.Infof("Loaded %d desktop files", len(a.desktopFiles))
	runtime.WindowSetAlwaysOnTop(ctx, true)
	a.trackMouse()
	a.startTicker()
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	logrus.Info("DOM is ready")
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	logrus.Info("Before close")
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	logrus.Info("Shutting down Wails app")
	a.ticker.Stop()
	if a.x11 != nil {
		a.x11.Close()
	}
}

func (a *App) getDesktopFiles() []DesktopFile {
	home, err := os.UserHomeDir()
	if err != nil {
		logrus.Errorf("Failed to get home directory: %v", err)
		return nil
	}
	path := filepath.Join(home, "GoDock", "apps.json")

	file, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("Error reading apps.json: %v", err)
		return nil
	}

	var desktopFiles []DesktopFile
	if err := json.Unmarshal(file, &desktopFiles); err != nil {
		logrus.Errorf("Error parsing apps.json: %v", err)
		return nil
	}
	return desktopFiles
}

func (a *App) trackMouse() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logrus.Error("trackMouse panic:", r)
			}
		}()

		for {
			x, y := robotgo.Location()
			// Define the middle of the screen (e.g., between 1/3 and 2/3 of the screen height)
			middleMin := a.screenHeight / 4
			middleMax := 3 * a.screenHeight / 4

			// Show the dock when the mouse is within the middle of the Y-axis
			if x <= 5 && y >= middleMin && y <= middleMax {
				if !a.windowVisible {
					a.windowVisible = true
					runtime.WindowShow(a.ctx)
					a.setSizeAndPosition()
				}
			}
			time.Sleep(100 * time.Millisecond) // Polling rate
		}
	}()
}

func (a *App) startTicker() {
	// Create a new ticker that ticks every 1 second
	ticker := time.NewTicker(1 * time.Second)
	a.ticker = ticker

	defer func() {
		if r := recover(); r != nil {
			logrus.Error("startTicker panic:", r)
		}
	}()

	go func() {
		// Periodically send updates to the frontend
		for range ticker.C {
			var apps []WmCtrlApp
			for wmClass, instances := range a.getRunningInstances() {
				apps = append(apps,
					WmCtrlApp{
						WmClass:   wmClass,
						Instances: instances,
					})

			}

			runtime.EventsEmit(a.ctx, "update", apps)
		}
	}()
}

func (a *App) getRunningInstances() map[string][]WmCtrlInstance {
	return a.x11.ListWindows()
}

func (a *App) setSizeAndPosition() {
	yPosition := a.screenHeight/2 - (len(a.desktopFiles) * 40)
	windowsHeight := len(a.desktopFiles) * 74
	logrus.Infof("Setting dock position to y: %d, height: %d", yPosition, windowsHeight)

	runtime.WindowSetPosition(a.ctx, 0, yPosition)
	runtime.WindowSetSize(a.ctx, 85, windowsHeight)
}
