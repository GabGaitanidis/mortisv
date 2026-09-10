package file

import (
	"os"
	"path/filepath"
)

type Handler struct {
	path string
}

func NewHandler(path string) *Handler {
	return &Handler{path: path}
}

func (h *Handler) Create() (bool, error) {
	if dir := filepath.Dir(h.path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return false, err
		}
	}
	f, err := os.OpenFile(h.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	return true, nil
}

func (h *Handler) Read() (string, error) {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (h *Handler) Write(content string) error {
	return os.WriteFile(h.path, []byte(content), 0o644)
}

func (h *Handler) Delete() error {
	return os.Remove(h.path) // Note: Requires confirmation from user
}
