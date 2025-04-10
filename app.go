package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/go-vgo/robotgo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	ticker *time.Ticker
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

func MapToDesktopFileForFE(df DesktopFile, instances []WmCtrlInstance) DesktopFileForFE {
	return DesktopFileForFE{
		Name:      df.Name,
		IconPath:  df.IconPath,
		ExecPath:  df.ExecPath,
		WmClass:   df.WmClass,
		Instances: instances,
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) GetDesktopFiles() []DesktopFileForFE {
	classToInstancesMap := a.getRunningInstances()

	file, err := os.ReadFile("/home/petr/GoDock/apps.json")
	if err != nil {
		logrus.Error("Error reading apps.json: %v", err)
		return nil
	}

	var apps []DesktopFile
	if err := json.Unmarshal(file, &apps); err != nil {
		logrus.Error("Error parsing apps.json: %v", err)
		return nil
	}

	appsForFE := make([]DesktopFileForFE, len(apps))
	for i, app := range apps {
		instances, _ := classToInstancesMap[strings.ToLower(app.Name)]
		appsForFE[i] = MapToDesktopFileForFE(app, instances)
		appsForFE[i].Instances = instances
	}

	return appsForFE
}

func (a *App) TrackMouse() {
	go func() {
		for {
			_, height := robotgo.GetScreenSize()
			x, y := robotgo.Location()
			// Define the middle of the screen (e.g., between 1/3 and 2/3 of the screen height)
			middleMin := height / 4
			middleMax := 3 * height / 4

			// Show the dock when the mouse is within the middle of the Y-axis
			if x <= 5 && y >= middleMin && y <= middleMax {
				runtime.WindowShow(a.ctx)
				a.setSizeAndPosition()
			}
			time.Sleep(100 * time.Millisecond) // Polling rate
		}
	}()
}

func (a *App) WindowHide() {
	x, _ := robotgo.Location()
	if x > 5 {
		runtime.WindowHide(a.ctx)
	}
}

var previousId string

func (a *App) BringToFrontOrLaunch(runningId string, execPath string) error {
	if runningId == "" {
		err := exec.Command("sh", "-c", execPath).Start()
		if err != nil {
			logrus.Error("Failed to execute command:", err)
		}
		return err
	}

	if previousId == runningId {
		err := exec.Command("xdotool", "windowminimize", runningId).Run()
		if err != nil {
			logrus.Error("Failed to minimize window:", err)
		}
		previousId = ""
		return err
	}

	err := exec.Command("wmctrl", "-ia", runningId).Run()
	if err == nil {
		previousId = runningId
	} else {
		logrus.Error("Failed to bring window to front:", err)
	}
	return err
}

func (a *App) getRunningInstances() map[string][]WmCtrlInstance {
	// Run the wmctrl command to list windows
	output, err := exec.Command("wmctrl", "-lx").Output()
	if err != nil {
		logrus.Error("Failed to execute wmctrl:", err)
	}

	// Parse the output into application names
	lines := strings.Split(string(output), "\n")
	apps := make(map[string][]WmCtrlInstance)

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue // Skip malformed lines
		}

		// Check if the second column is "-1" (invisible desktops)
		if parts[1] == "-1" {
			continue
		}

		// Extract the application class from the third column
		appClass := strings.Split(parts[2], ".")[0] // Get only the class name, ignoring the instance
		windowID := parts[0]

		// Append the window ID to the list of window IDs for this app class
		instance := WmCtrlInstance{
			WindowId: windowID,
			Name:     strings.Join(parts[4:], " "),
		}
		apps[appClass] = append(apps[appClass], instance)
	}

	return apps
}

func (a *App) StartTicker() {
	// Create a new ticker that ticks every 1 second
	ticker := time.NewTicker(1 * time.Second)
	a.ticker = ticker

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

func (a *App) setSizeAndPosition() {
	apps := a.GetDesktopFiles()
	_, height := robotgo.GetScreenSize()
	runtime.WindowSetPosition(a.ctx, 0, height/2-(len(apps)*40))
	runtime.WindowSetSize(a.ctx, 85, len(apps)*74)
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logrus.Info("Starting Wails app")
	runtime.WindowSetAlwaysOnTop(ctx, true) // Keep the window on top
	a.TrackMouse()                          // Start tracking mouse globally

}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	a.setSizeAndPosition()
	logrus.Info("DOM is ready")
	a.StartTicker()
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
	// Perform your teardown here
}
