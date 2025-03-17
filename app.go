package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx    context.Context
	ticker *time.Ticker
}

type DesktopFile struct {
	Name      string           `json:"name"`
	IconPath  string           `json:"iconPath"`
	Instances []WmCtrlInstance `json:"instances"`
	ExecPath  string           `json:"execPath"`
	WmClass   string           `json:"wmClass"`
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

func (a *App) GetDesktopFiles() []DesktopFile {
	classToInstancesMap := a.getRunningInstances()
	apps := []DesktopFile{}

	// Get .desktop files from /usr/share/applications
	desktopFiles, _ := filepath.Glob("/home/petr/GoDock/*.desktop")

	for _, file := range desktopFiles {
		name, icon, exec, wmClass := parseDesktopFile(file)
		instances, _ := classToInstancesMap[strings.ToLower(name)]
		apps = append(apps, DesktopFile{Name: name, IconPath: icon, Instances: instances, ExecPath: exec, WmClass: wmClass})
	}

	return apps
}

func parseDesktopFile(filePath string) (string, string, string, string) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", "", ""
	}

	var name, icon, exec, wmClass string
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Name=") {
			name = strings.TrimPrefix(line, "Name=")
		} else if strings.HasPrefix(line, "Icon=") {
			icon = strings.TrimPrefix(line, "Icon=")
		} else if strings.HasPrefix(line, "Exec=") {
			exec = strings.TrimPrefix(line, "Exec=")
		} else if strings.HasPrefix(line, "StartupWMClass=") {
			wmClass = strings.TrimPrefix(line, "StartupWMClass=")
		}

	}

	return name, icon, exec, wmClass
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
		return exec.Command("sh", "-c", execPath).Start()
	}

	if previousId == runningId {
		err := exec.Command("xdotool", "windowminimize", runningId).Run()
		previousId = ""
		return err
	}

	err := exec.Command("wmctrl", "-ia", runningId).Run()
	if err == nil {
		previousId = runningId
	}
	return err
}

func (a *App) getRunningInstances() map[string][]WmCtrlInstance {
	// Run the wmctrl command to list windows
	cmd := exec.Command("wmctrl", "-lx")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Failed to execute wmctrl:", err)
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
	runtime.WindowSetAlwaysOnTop(ctx, true) // Keep the window on top
	a.TrackMouse()                          // Start tracking mouse globally

}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
	a.setSizeAndPosition()
	a.StartTicker()
}

// beforeClose is called when the application is about to quit,
// either by clicking the window close button or calling runtime.Quit.
// Returning true will cause the application to continue, false will continue shutdown as normal.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	a.ticker.Stop()
	// Perform your teardown here
}
