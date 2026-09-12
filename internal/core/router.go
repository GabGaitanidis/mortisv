package core

import (
	"fmt"
	"log/slog"
	"regexp"

	"mortis/internal/logging"
	"mortis/internal/speech"
)

var resultRefPattern = regexp.MustCompile(`\$result\[([^\]]+)\]`)

type CommandRouter struct {
	Modules map[string]Module
	Tts     speech.TtsBridge
	logger  *logging.Logger
}

func NewCommandRouter(tts speech.TtsBridge, modules map[string]Module, logger *slog.Logger) *CommandRouter {
	return &CommandRouter{
		Modules: modules,
		Tts:     tts,
		logger:  logging.New(logger, "CommandRouter"),
	}
}

func (r *CommandRouter) RouteAll(commands []Command) error {
	resultsByActivity := map[string]string{}
	resultsByKey := map[string]string{}

	for i, cmd := range commands {
		r.resolveReferences(&cmd, resultsByActivity, resultsByKey)

		module, ok := r.Modules[cmd.Module]
		if !ok {
			r.logger.Warn("unknown module", "module", cmd.Module)
			continue
		}

		result, err := module.Execute(cmd, r.Tts)
		if err != nil {
			r.logger.Error("command failed", "activity", cmd.ActivityName, "error", err)
			continue
		}

		resultsByActivity[cmd.ActivityName] = result
		if key, ok := cmd.Params["key"]; ok {
			resultsByKey[fmt.Sprint(key)] = result
		}

		hasMore := i < len(commands)-1
		speakText := result
		if hasMore {
			speakText = result + " and"
		}
		r.logger.Info("speaking", "text", speakText)
		if err := r.Tts.Speak(speakText); err != nil {
			r.logger.Error("tts error", "error", err)
		}
	}
	return nil
}

func (r *CommandRouter) resolveReferences(cmd *Command, byActivity, byKey map[string]string) {
	for k, v := range cmd.Params {
		strVal, ok := v.(string)
		if !ok {
			continue
		}
		resolved := resultRefPattern.ReplaceAllStringFunc(strVal, func(match string) string {
			ref := resultRefPattern.FindStringSubmatch(match)[1]
			if replacement, ok := byActivity[ref]; ok {
				return replacement
			}
			if replacement, ok := byKey[ref]; ok {
				return replacement
			}
			r.logger.Warn("could not resolve reference", "reference", ref)
			return match
		})
		cmd.Params[k] = resolved
	}
}