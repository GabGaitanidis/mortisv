package ai

import (
	"context"
	"log/slog"

	"mortis/internal/core"
	"mortis/internal/logging"
	"mortis/internal/rapport"
	"mortis/internal/userprofile"
)

type AIManager struct {
	client       *GroqClient
	modules      []core.Module
	profileStore *userprofile.ProfileStore
	rapportStore *rapport.Store
	logger       *logging.Logger
}

func NewAIManager(client *GroqClient, modules []core.Module, logger *slog.Logger, profileStore *userprofile.ProfileStore, rapportStore *rapport.Store) *AIManager {
	return &AIManager{
		client:       client,
		modules:      modules,
		profileStore: profileStore,
		rapportStore: rapportStore,
		logger:       logging.New(logger, "AIManager"),
	}
}

func (m *AIManager) Ask(ctx context.Context, userInput string) string {
	m.logger.Info("input delivered", "input_len", len(userInput))

	profile, err := m.currentProfile()
	if err != nil {
		m.logger.Error("error loading profile", "error", err)
		profile = nil
	}

	recentRapport, err := m.recentRapport()
	if err != nil {
		m.logger.Error("error loading rapport", "error", err)
		recentRapport = nil
	}

	builder := NewPromptBuilder(m.modules)
	systemPrompt := builder.Build(userInput, profile, recentRapport)

	response, err := m.client.Complete(ctx, systemPrompt, userInput)
	if err != nil {
		m.logger.Error("error sending request", "error", err)
		return "There is something wrong with the server, please check logs"
	}

	return response
}

func (m *AIManager) currentProfile() (Profile, error) {
	if m.profileStore == nil {
		return nil, nil
	}
	return m.profileStore.Load()
}

func (m *AIManager) recentRapport() ([]rapport.Entry, error) {
	if m.rapportStore == nil {
		return nil, nil
	}
	return m.rapportStore.Recent(10)
}