package system

import (
	"fmt"

	"mortis/internal/core"
	"mortis/internal/speech"
)

type Module struct{}

var _ core.Module = (*Module)(nil)

func (m *Module) Name() string { return "system" }

func (m *Module) Execute(cmd core.Command, _ speech.TtsBridge) (string, error) {
	handler := newHandler()
	if cmd.Action == "info" {
		return handler.info()
	}
	if cmd.Action == "open_app" {
		if err := handler.openApp(cmd.GetString("app")); err != nil {
			return "I could not open that application", err
		}
		return "Application opened", nil
	}
	if cmd.Action != "execute" {
		return "There is an unknown action in system module", fmt.Errorf("system: unknown action %q", cmd.Action)
	}
	command := cmd.GetString("command")
	if command == "" {
		return "I need a system command", nil
	}
	output, err := handler.executeCommand(command, cmd.GetStringSlice("args")...)
	if err != nil {
		return "There is an error in system module", err
	}
	return output, nil
}

func (m *Module) Description() string {
	return `system
	- execute: params MUST include "command" (string) and MAY include "args" (array of strings). Executes an OS command available on this machine.
	- info: returns the current operating system, architecture, hostname, and working directory.
	- open_app: params MUST include "app" (string). Opens an installed application using the operating system.
	Use module "system" only for explicit system commands.`
}
