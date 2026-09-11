package modules

import (
	"mortis/internal/core"
	filemod "mortis/internal/modules/file"
)

func ModulesList() []core.Module {
	return []core.Module{
		&filemod.Module{},
	}
}
