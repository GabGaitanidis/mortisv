package question

import (
	"context"
	"fmt"
	"strings"
	"time"

	"mortis/internal/ai"
	"mortis/internal/core"
	"mortis/internal/speech"
)

type Module struct {
	client *ai.GroqClient
}

func NewModule(client *ai.GroqClient) *Module {
	return &Module{client: client}
}

var _ core.Module = (*Module)(nil)

func (m *Module) Name() string { return "question" }

func (m *Module) Execute(cmd core.Command, _ speech.TtsBridge) (string, error) {
	if cmd.Action != "answer" {
		return "Unknown question action", fmt.Errorf("question: unknown action %q", cmd.Action)
	}
	question := strings.TrimSpace(cmd.GetString("question"))
	if question == "" {
		return "I need a question to answer", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	answer, err := m.client.Complete(ctx, "Answer the user's question accurately and concisely. Return plain text only.", question)
	if err != nil {
		return "I could not answer that question", err
	}
	return answer, nil
}

func (m *Module) Description() string {
	return `question
	- answer: params MUST include "question" (string). Answers general knowledge questions using the AI client.
	Use module "question" for general questions; use user.get_fact for personal facts.`
}
