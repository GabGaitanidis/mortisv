package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"mortis/internal/logging"
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
	logger *slog.Logger,
) (PostExchangeResult, error) {
	log := logging.New(logger, "PostExchange")
	prompt := builder.BuildPostExchangePrompt(profile, recentRapport, userMessage, mortisReply)

	log.Info("checking post-exchange memory update",
		"user_message_len", len(userMessage),
		"morti_reply_len", len(mortisReply),
		"recent_rapport_entries", len(recentRapport),
	)

	raw, err := client.Complete(ctx, prompt, "Return the JSON now.")
	if err != nil {
		log.Error("post-exchange check failed", "error", err)
		return PostExchangeResult{}, fmt.Errorf("post-exchange check failed: %w", err)
	}

	var result PostExchangeResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &result); err != nil {
		log.Error("post-exchange result parse failed", "raw", raw, "error", err)
		return PostExchangeResult{}, fmt.Errorf("post-exchange: unmarshal %q: %w", raw, err)
	}

	log.Info("post-exchange check complete",
		"profile_changed", result.ProfileChanged,
		"noteworthy", result.Noteworthy,
	)
	return result, nil
}


func ApplyPostExchange(
	profileStore *userprofile.ProfileStore,
	rapportStore *rapport.Store,
	profile *userprofile.UserProfile,
	result PostExchangeResult,
	logger *slog.Logger,
) error {
	log := logging.New(logger, "PostExchange")

	if result.ProfileChanged {
		for _, update := range result.ProfileUpdates {
			profile.Apply(update)
		}
		if err := profileStore.Save(profile); err != nil {
			log.Error("failed to save profile update", "error", err)
			return fmt.Errorf("save profile: %w", err)
		}
		log.Info("profile updated",
			"updates", len(result.ProfileUpdates),
		)
	}

	if result.Noteworthy {
		entry := rapport.Entry{
			Note:      result.RapportNote,
			Tag:       result.RapportTag,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		if err := rapportStore.Add(entry); err != nil {
			log.Error("failed to save rapport update", "error", err)
			return fmt.Errorf("save rapport entry: %w", err)
		}
		log.Info("rapport entry saved",
			"tag", result.RapportTag,
		)
	}

	return nil
}
