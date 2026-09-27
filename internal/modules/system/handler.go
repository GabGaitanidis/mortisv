package system

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Handler struct{}

func newHandler() *Handler {
	return &Handler{}
}

func (h *Handler) executeCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command %s: %v, output: %s", command, err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func (h *Handler) info() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("OS: %s; architecture: %s; hostname: %s; working directory: %s", runtime.GOOS, runtime.GOARCH, hostname, workingDir), nil
}

func (h *Handler) openApp(app string) error {
	if app == "" {
		return fmt.Errorf("application name is required")
	}
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command, args = "open", []string{"-a", app}
	case "linux":
		command, args = app, nil
	case "windows":
		command, args = "cmd", []string{"/c", "start", "", app}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return exec.Command(command, args...).Start()
}
