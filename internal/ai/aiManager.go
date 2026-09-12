package ai

import (
	"context"
	"log/slog"
	"time"

	"mortis/internal/core"
	"mortis/internal/logging"
	"mortis/internal/rapport"
	"mortis/internal/userprofile"
)

const recentRapportLimit = 10

type AIManager struct {
	client       *GroqClient
	modules      []core.Module
	logger       *logging.Logger
	profileStore *userprofile.ProfileStore
	rapportStore *rapport.Store
}

func NewAIManager(
	client *GroqClient,
	modules []core.Module,
	logger *slog.Logger,
	profileStore *userprofile.ProfileStore,
	rapportStore *rapport.Store,
) *AIManager {
	return &AIManager{
		client:       client,
		modules:      modules,
		logger:       logging.New(logger, "AIManager"),
		profileStore: profileStore,
		rapportStore: rapportStore,
	}
}

func (m *AIManager) Ask(ctx context.Context, userInput string) string {
	m.logger.Info("input delivered", "input_len", len(userInput))

	profile, err := m.profileStore.Load()
	if err != nil {
		m.logger.Error("failed to load profile, continuing with empty profile", "error", err)
		profile = userprofile.NewUserProfile()
	}

	recentRapport, err := m.rapportStore.Recent(recentRapportLimit)
	if err != nil {
		m.logger.Error("failed to load rapport, continuing with none", "error", err)
	}

	builder := NewPromptBuilder(m.modules)
	systemPrompt := builder.Build(userInput, profile, recentRapport)

	response, err := m.client.Complete(ctx, systemPrompt, userInput)
	if err != nil {
		m.logger.Error("error sending request", "error", err)
		return "There is something wrong with the server, please check logs"
	}

	go m.checkPostExchange(profile, recentRapport, userInput, response)

	return response
}

func (m *AIManager) checkPostExchange(profile *userprofile.UserProfile, recentRapport []rapport.Entry, userInput, response string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	builder := NewPromptBuilder(m.modules)
	result, err := CheckPostExchange(ctx, m.client, builder, profile, recentRapport, userInput, response)
	if err != nil {
		m.logger.Error("post-exchange check failed", "error", err)
		return
	}

	if err := ApplyPostExchange(m.profileStore, m.rapportStore, profile, result); err != nil {
		m.logger.Error("failed to apply post-exchange result", "error", err)
	}
}