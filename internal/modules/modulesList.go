package modules

import (
	"net/http"

	"mortis/internal/ai"
	"mortis/internal/core"
	browserMod "mortis/internal/modules/browser"
	calendarMod "mortis/internal/modules/calendar"
	filemod "mortis/internal/modules/file"
	questionMod "mortis/internal/modules/question"
	systemMod "mortis/internal/modules/system"
	unknownsafeoption "mortis/internal/modules/unknownSafeOption"
)

func ModulesList(client *ai.GroqClient) []core.Module {
	return []core.Module{
		&filemod.Module{},
		browserMod.NewModule(client),
		questionMod.NewModule(client),
		&systemMod.Module{},
		calendarMod.NewModule(http.DefaultClient),
		&unknownsafeoption.Module{},
	}
}
