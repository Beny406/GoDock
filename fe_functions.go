package main

import (
	"os/exec"
	"strings"
	"syscall"

	"github.com/go-vgo/robotgo"
	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetDesktopFilesForFE() []DesktopFileForFE {
	classToInstancesMap := a.getRunningInstances()
	appsForFE := make([]DesktopFileForFE, len(a.desktopFiles))
	for i, app := range a.desktopFiles {
		instances, _ := classToInstancesMap[strings.ToLower(app.Name)]
		appsForFE[i] = DesktopFileForFE{
			Name:      app.Name,
			IconPath:  app.IconPath,
			ExecPath:  app.ExecPath,
			WmClass:   app.WmClass,
			Instances: instances,
		}
	}

	return appsForFE
}

func (a *App) OnIconClick(runningId string, execPath string) error {
	if runningId == "" {
		cmd := exec.Command("sh", "-c", execPath)
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
		// start as a separate process so it doesnt crash when dock crashes
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		err := cmd.Start()
		if err != nil {
			logrus.Error("Failed to execute command:", err)
		}
		return err
	}

	if a.previousId == runningId {
		err := a.x11.MinimizeWindow(runningId)
		if err != nil {
			logrus.Error("Failed to minimize window:", err)
		}
		a.previousId = ""
		return err
	}

	err := a.x11.FocusWindow(runningId)
	if err == nil {
		a.previousId = runningId
	} else {
		logrus.Error("Failed to bring window to front:", err)
	}
	return err
}

func (a *App) WindowHide() {
	x, _ := robotgo.Location()
	if x > 5 {
		a.windowVisible = false
		runtime.WindowHide(a.ctx)
	}
}
