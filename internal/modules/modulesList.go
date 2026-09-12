package modules

import (
	"mortis/internal/core"
	filemod "mortis/internal/modules/file"
	unknownsafeoption "mortis/internal/modules/unknownSafeOption"
)

func ModulesList() []core.Module {
	return []core.Module{
		&filemod.Module{},
		&unknownsafeoption.Module{},
	}
}
