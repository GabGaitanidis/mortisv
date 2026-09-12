package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mortis/internal/ai"
	"mortis/internal/core"
	"mortis/internal/modules"
	"mortis/internal/rapport"
	"mortis/internal/session"
	"mortis/internal/speech"
	"mortis/internal/userprofile"

	"github.com/joho/godotenv"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

func loadEnv() {
	if err := godotenv.Load(); err != nil {
		logger.Warn("no .env file found, relying on system environment variables")
	}
}

func main() {
	loadEnv()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutdown signal received")
		cancel()
	}()

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		logger.Warn("no API key loaded; Groq requests may fail")
	}

	waitWord := speech.NewPythonWakeWord("scripts/wake.py", logger)
	if err := waitWord.Start(ctx); err != nil {
		logger.Error("wake word bridge failed to start", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := waitWord.Shutdown(shutdownCtx); err != nil {
			logger.Error("wake word shutdown error", "error", err)
		}
	}()

	client := ai.NewGroqClient(apiKey, logger)
	moduleList := modules.ModulesList()

	moduleMap := make(map[string]core.Module, len(moduleList))
	for _, m := range moduleList {
		moduleMap[m.Name()] = m
	}

	profileStore, err := userprofile.NewProfileStore(os.Getenv("PROFILE_FILE"))
	if err != nil {
		logger.Error("failed to open profile store", "error", err)
		os.Exit(1)
	}

	rapportStore, err := rapport.NewStore(os.Getenv("RAPPORT_FILE"))
	if err != nil {
		logger.Error("failed to open rapport store", "error", err)
		os.Exit(1)
	}

	manager := ai.NewAIManager(client, moduleList, logger, profileStore, rapportStore)

	exchangeSession := session.NewExchangeSession("scripts/stt.py", "scripts/tts.py", 5*time.Minute, logger)
	defer exchangeSession.Shutdown()

	logger.Info("mortis ready, listening for wake word")
	runAssistantLoop(ctx, waitWord, manager, moduleMap, exchangeSession)
}

func runAssistantLoop(ctx context.Context, wake speech.WakeWordBridge, manager *ai.AIManager, moduleMap map[string]core.Module, exchangeSession *session.ExchangeSession) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := wake.Listen(); err != nil {
			logger.Error("wake word error", "error", err)
			continue
		}

		logger.Info("wake word detected")

		stt, tts, err := exchangeSession.EnsureStarted(ctx)
		if err != nil {
			logger.Error("failed to start exchange session", "error", err)
			continue
		}

		command, err := stt.Listen()
		if err != nil {
			logger.Error("stt error while listening for command", "error", err)
			continue
		}

		if command == "" || command == "no input" {
			logger.Info("no command heard, returning to wake word mode")
			continue
		}

		router := core.NewCommandRouter(tts, moduleMap, logger)
		handleCommand(ctx, manager, router, command)
	}
}

func handleCommand(ctx context.Context, manager *ai.AIManager, router *core.CommandRouter, transcript string) {
	askCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	raw := manager.Ask(askCtx, transcript)

	var commands []core.Command
	if err := json.Unmarshal([]byte(raw), &commands); err != nil {
		logger.Error("failed to unmarshal commands", "error", err, "raw", raw)
		return
	}

	if err := router.RouteAll(commands); err != nil {
		logger.Error("failed to route commands", "error", err)
	}
}