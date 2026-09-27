package speech

import (
	"os"
	"os/exec"
)

func pythonBin() string {
	if configured := os.Getenv("PYTHON_BIN"); configured != "" && executable(configured) {
		return configured
	}
	if executable("/app/venv/bin/python") {
		return "/app/venv/bin/python"
	}
	if configured := os.Getenv("PYTHON_BIN"); configured != "" {
		return configured
	}
	return "python3"
}

func executable(path string) bool {
	if _, err := exec.LookPath(path); err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
