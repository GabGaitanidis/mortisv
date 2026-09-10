package core

import (
	"fmt"
	"log"
	"regexp"

	"mortis/internal/speech"
)

var resultRefPattern = regexp.MustCompile(`\$result\[([^\]]+)\]`)

type CommandRouter struct {
	Modules map[string]Module
	Tts     speech.TtsBridge
}

func NewCommandRouter(tts speech.TtsBridge, modules map[string]Module) *CommandRouter {
	return &CommandRouter{Modules: modules, Tts: tts}
}

func (r *CommandRouter) RouteAll(commands []Command) error {
	resultsByActivity := map[string]string{}
	resultsByKey := map[string]string{}

	for i, cmd := range commands {
		resolveReferences(&cmd, resultsByActivity, resultsByKey)

		module, ok := r.Modules[cmd.Module]
		if !ok {
			log.Printf("unknown module: %s", cmd.Module)
			continue
		}

		result, err := module.Execute(cmd, r.Tts)
		if err != nil {
			log.Printf("command failed: %s - %v", cmd.ActivityName, err)
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
		log.Printf("Speaking... %s", speakText)
		if err := r.Tts.Speak(speakText); err != nil {
			log.Printf("tts error: %v", err)
		}
	}
	return nil
}

func resolveReferences(cmd *Command, byActivity, byKey map[string]string) {
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
			log.Printf("[CHAIN] could not resolve reference: %s", ref)
			return match
		})
		cmd.Params[k] = resolved
	}
}
