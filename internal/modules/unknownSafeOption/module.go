package unknownsafeoption

import (
	"mortis/internal/core"
	"mortis/internal/speech"
)


type Module struct{}
 
var _ core.Module = (*Module)(nil)
 
func (m *Module) Name() string { return "unknown" }
 
func (m *Module) Execute(cmd core.Command, tts speech.TtsBridge) (string, error) {
	return "I'm not sure how to help with that yet.", nil
}
 
func (m *Module) Description() string {
	return `unknown
- none: fallback for anything that doesn't match another module. params MUST include "value" (string) — the original unclassified input.
Use module "unknown" when nothing else fits.`
}