package core

import "mortis/internal/speech"

type Module interface {
	Name() string
	Description() string
	Execute(cmd Command, tts speech.TtsBridge) (string, error)
}
