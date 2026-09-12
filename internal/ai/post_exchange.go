package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"mortis/internal/rapport"
	"mortis/internal/userprofile"
)


type PostExchangeResult struct {
	ProfileChanged bool                      `json:"profileChanged"`
	ProfileUpdates []userprofile.FieldUpdate `json:"profileUpdates"`
	Noteworthy     bool                      `json:"noteworthy"`
	RapportNote    string                    `json:"rapportNote"`
	RapportTag     string                    `json:"rapportTag"`
}


func CheckPostExchange(
	ctx context.Context,
	client *GroqClient,
	builder *PromptBuilder,
	profile *userprofile.UserProfile,
	recentRapport []rapport.Entry,
	userMessage, mortisReply string,
) (PostExchangeResult, error) {
	prompt := builder.BuildPostExchangePrompt(profile, recentRapport, userMessage, mortisReply)

	
	raw, err := client.Complete(ctx, prompt, "Return the JSON now.")
	if err != nil {
		return PostExchangeResult{}, fmt.Errorf("post-exchange check failed: %w", err)
	}

	var result PostExchangeResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &result); err != nil {
		return PostExchangeResult{}, fmt.Errorf("post-exchange: unmarshal %q: %w", raw, err)
	}

	
	return result, nil
}


func ApplyPostExchange(
	profileStore *userprofile.ProfileStore,
	rapportStore *rapport.Store,
	profile *userprofile.UserProfile,
	result PostExchangeResult,
) error {

	if result.ProfileChanged {
		for _, update := range result.ProfileUpdates {
			profile.Apply(update)
		}
		if err := profileStore.Save(profile); err != nil {
			return fmt.Errorf("save profile: %w", err)
		}
	}

	if result.Noteworthy {
		entry := rapport.Entry{
			Note:      result.RapportNote,
			Tag:       result.RapportTag,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if err := rapportStore.Add(entry); err != nil {
			return fmt.Errorf("save rapport entry: %w", err)
		}
	}

	return nil
}
