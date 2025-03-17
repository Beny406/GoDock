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

type AppInfo struct {
	Name      string `json:"name"`
	IconPath  string `json:"iconPath"`
	RunningId string `json:"runningId"`
	ExecPath  string `json:"execPath"`
	WmClass   string `json:"wmClass"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) GetDesktopFiles() []AppInfo {
	runningApps := a.getRunningApps()
	apps := []AppInfo{}

	// Get .desktop files from /usr/share/applications
	desktopFiles, _ := filepath.Glob("/home/petr/GoDock/*.desktop")

	for _, file := range desktopFiles {
		name, icon, exec, wmClass := parseDesktopFile(file)
		runningId, _ := runningApps[strings.ToLower(name)]
		apps = append(apps, AppInfo{Name: name, IconPath: icon, RunningId: runningId, ExecPath: exec, WmClass: wmClass})
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
			middleMin := height / 3
			middleMax := 2 * height / 3

			// Show the dock when the mouse is within the middle of the Y-axis
			if x <= 5 && y >= middleMin && y <= middleMax {
				runtime.WindowShow(a.ctx)
				runtime.WindowShow(a.ctx)
			} else if x >= 100 {
				runtime.WindowHide(a.ctx)
			}
			time.Sleep(100 * time.Millisecond) // Polling rate
		}
	}()
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

func (a *App) getRunningApps() map[string]string {
	// Run the wmctrl command to list windows
	cmd := exec.Command("wmctrl", "-lx")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Failed to execute wmctrl:", err)
	}

	// Parse the output into application names
	lines := strings.Split(string(output), "\n")
	apps := make(map[string]string)

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
		apps[appClass] = parts[0]

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
			// Do some processing (e.g., checking system stats, data, etc.)

			// Send result to the frontend (use runtime.SendToFrontend)
			var runningApps [][]string
			for key, value := range a.getRunningApps() {
				runningApps = append(runningApps, []string{key, value})
			}

			runtime.EventsEmit(a.ctx, "update", runningApps)

		}
	}()

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

	apps := a.GetDesktopFiles()
	_, height := robotgo.GetScreenSize()
	runtime.WindowSetPosition(ctx, -5, height/2-(len(apps)*40))
	runtime.WindowSetSize(ctx, 85, len(apps)*74)
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
