package file

import (
	"os"
	"path/filepath"

	"mortis/internal/core"
	"mortis/internal/speech"
)


type Module struct{}

var _ core.Module = (*Module)(nil)

func (m *Module) Name() string { return "file" }

func (m *Module) Execute(cmd core.Command, tts speech.TtsBridge) (string, error) {

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	fileRoot := filepath.Join(home, "Desktop")

	path := cmd.GetString("path")
	if path == "" {
		path = "Desktop"
	}
	handler := NewHandler(filepath.Join(fileRoot, path))

	switch cmd.Action {
	case "create":
		if _, err := handler.Create(); err != nil {
			return "I cannot create the file", nil
		}
		return "File created", nil

	case "read":
		content, err := handler.Read()
		if err != nil {
			return "I cannot read the file", nil
		}
		return "File contents: " + content, nil

	case "write":
		if err := handler.Write(cmd.GetString("content")); err != nil {
			return "I cannot write to the file", nil
		}
		return "File written successfully", nil

	case "delete":
		if err := handler.Delete(); err != nil {
			return "I cannot delete the file", nil
		}
		return "File deleted successfully", nil

	default:
		return "Something went wrong with actions", nil
	}
}

func (m *Module) Description() string {
	return `file
			- create_folder: params MUST include "path" (string)
			- create: params MUST include "path" (string)
			- write: params MUST include "path" (string) and "content" (string; use "" if no content was specified — if content is "", use action "create" instead)
			- read: params MUST include "path" (string)
			- delete: params MUST include "path" (string)
			Use module "file" (NOT the action name) for all of the above.`
}
