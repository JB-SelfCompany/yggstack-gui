//go:build windows

package platform

import (
	"fmt"
	"os/user"
	"path/filepath"
	"time"

	"github.com/capnspacehook/taskmaster"
	"github.com/rickb777/date/period"
)

const (
	// Task Scheduler folder and task name
	taskFolder = "\\Yggstack"
	taskName   = "Yggstack-GUI Autostart"
	taskPath   = taskFolder + "\\" + taskName

	// Delay after user logon before starting the application (in seconds)
	// This gives Windows time to fully initialize the graphics subsystem
	logonDelaySeconds = 30
)

// enableAutoStartWindows enables autostart on Windows using Task Scheduler
// Creates a scheduled task that runs on user logon with a delay
// This is more reliable than Registry Run key for CEF applications
func enableAutoStartWindows() error {
	// Get executable path
	exePath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Validate executable path for security
	if err := validateExecutablePath(exePath); err != nil {
		return fmt.Errorf("invalid executable path: %w", err)
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Connect to Task Scheduler
	ts, err := taskmaster.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Task Scheduler: %w", err)
	}
	defer ts.Disconnect()

	// Delete existing task if it exists
	_ = ts.DeleteTask(taskPath)

	// Create a new task definition
	def := ts.NewTaskDefinition()

	// Configure task settings
	def.Settings.AllowDemandStart = true
	def.Settings.AllowHardTerminate = true
	def.Settings.DontStartOnBatteries = false
	def.Settings.Enabled = true
	def.Settings.Hidden = false
	def.Settings.Priority = 7 // Normal priority
	def.Settings.RunOnlyIfIdle = false
	def.Settings.RunOnlyIfNetworkAvailable = false
	def.Settings.StartWhenAvailable = true
	def.Settings.StopIfGoingOnBatteries = false
	def.Settings.WakeToRun = false
	def.Settings.MultipleInstances = taskmaster.TASK_INSTANCES_IGNORE_NEW

	// Create a LogonTrigger with delay
	// The delay gives Windows time to fully initialize graphics
	// Empty UserID means trigger for any user - Principal restricts to current user
	trigger := taskmaster.LogonTrigger{
		TaskTrigger: taskmaster.TaskTrigger{
			Enabled:       true,
			StartBoundary: time.Now(),
		},
		Delay:  period.NewHMS(0, 0, logonDelaySeconds), // 30 second delay
		UserID: "",                                     // Any user (Principal handles the restriction)
	}
	def.AddTrigger(trigger)

	// Get working directory (same as executable directory)
	// CEF applications need this to find DLLs and resources
	workingDir := filepath.Dir(exePath)

	// Create an execution action with --minimized flag
	action := taskmaster.ExecAction{
		Path:       exePath,
		Args:       "--minimized",
		WorkingDir: workingDir,
	}
	def.AddAction(action)

	// Set principal to run as current user with interactive logon
	def.Principal.UserID = currentUser.Username
	def.Principal.LogonType = taskmaster.TASK_LOGON_INTERACTIVE_TOKEN
	def.Principal.RunLevel = taskmaster.TASK_RUNLEVEL_LUA // Limited User Access (no admin)

	// Create the task folder if it doesn't exist
	_, _, _ = ts.CreateTask(taskFolder+"\\dummy", ts.NewTaskDefinition(), false)
	_ = ts.DeleteTask(taskFolder + "\\dummy")

	// Register the task
	_, _, err = ts.CreateTask(taskPath, def, true)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	return nil
}

// disableAutoStartWindows disables autostart on Windows by removing the scheduled task
func disableAutoStartWindows() error {
	// Connect to Task Scheduler
	ts, err := taskmaster.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Task Scheduler: %w", err)
	}
	defer ts.Disconnect()

	// Delete the task
	err = ts.DeleteTask(taskPath)
	if err != nil {
		// Task might not exist, which is fine
		return nil
	}

	return nil
}

// isAutoStartEnabledWindows checks if autostart is enabled on Windows
// Returns true if the scheduled task exists and is enabled
func isAutoStartEnabledWindows() (bool, error) {
	// Connect to Task Scheduler
	ts, err := taskmaster.Connect()
	if err != nil {
		return false, fmt.Errorf("failed to connect to Task Scheduler: %w", err)
	}
	defer ts.Disconnect()

	// Try to get the task
	task, err := ts.GetRegisteredTask(taskPath)
	if err != nil {
		// Task doesn't exist
		return false, nil
	}
	defer task.Release()

	// Check if task is enabled
	return task.Definition.Settings.Enabled, nil
}

// syncAutoStartPath updates the scheduled task with the current executable path
// This should be called at application startup to handle cases where the app was moved
func syncAutoStartPath() error {
	enabled, err := isAutoStartEnabledWindows()
	if err != nil || !enabled {
		return err // Not enabled or error - nothing to sync
	}

	// Get current executable path
	currentPath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Connect to Task Scheduler
	ts, err := taskmaster.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to Task Scheduler: %w", err)
	}
	defer ts.Disconnect()

	// Get the existing task
	task, err := ts.GetRegisteredTask(taskPath)
	if err != nil {
		return nil // Task doesn't exist, nothing to sync
	}
	defer task.Release()

	// Check if the path matches
	for _, action := range task.Definition.Actions {
		if execAction, ok := action.(taskmaster.ExecAction); ok {
			if execAction.Path == currentPath {
				return nil // Path already matches
			}
		}
	}

	// Path doesn't match, recreate the task
	return enableAutoStartWindows()
}

// Stub implementations for Linux functions (not used on Windows)
// These are needed for compilation when building on Windows

func enableAutoStartLinux() error {
	return fmt.Errorf("Linux autostart not available on Windows")
}

func disableAutoStartLinux() error {
	return fmt.Errorf("Linux autostart not available on Windows")
}

func isAutoStartEnabledLinux() (bool, error) {
	return false, fmt.Errorf("Linux autostart not available on Windows")
}
