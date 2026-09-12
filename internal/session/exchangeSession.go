package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"mortis/internal/logging"
	"mortis/internal/speech"
)

type ExchangeSession struct {
	sttScript   string
	ttsScript   string
	idleTimeout time.Duration
	rawLogger   *slog.Logger
	logger      *logging.Logger

	mu        sync.Mutex
	stt       *speech.PythonSTT
	tts       *speech.PythonTTS
	running   bool
	idleTimer *time.Timer
}

func NewExchangeSession(sttScript, ttsScript string, idleTimeout time.Duration, logger *slog.Logger) *ExchangeSession {
	return &ExchangeSession{
		sttScript:   sttScript,
		ttsScript:   ttsScript,
		idleTimeout: idleTimeout,
		rawLogger:   logger,
		logger:      logging.New(logger, "ExchangeSession"),
	}
}


func (s *ExchangeSession) EnsureStarted(ctx context.Context) (*speech.PythonSTT, *speech.PythonTTS, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.logger.Info("reusing warm session", "state", "warm")
		s.resetIdleTimerLocked()
		return s.stt, s.tts, nil
	}

	s.logger.Info("starting stt/tts — session was cold", "state", "warming")

	stt := speech.NewPythonSTT(s.sttScript, s.rawLogger)
	if err := stt.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("session: start stt: %w", err)
	}

	tts := speech.NewPythonTTS( s.ttsScript, s.rawLogger)
	if err := tts.Start(ctx); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = stt.Shutdown(shutdownCtx)
		return nil, nil, fmt.Errorf("session: start tts: %w", err)
	}

	s.stt = stt
	s.tts = tts
	s.running = true
	s.resetIdleTimerLocked()
	tts.Speak("Hello sir, what can I do for you?")
	return s.stt, s.tts, nil
}

func (s *ExchangeSession) resetIdleTimerLocked() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	s.idleTimer = time.AfterFunc(s.idleTimeout, s.shutdownIdle)
}

func (s *ExchangeSession) shutdownIdle() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.logger.Info("idle timeout reached, shutting down stt/tts", "state", "resting")
	s.stopLocked()
}

func (s *ExchangeSession) stopLocked() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.tts != nil {
		if err := s.tts.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("tts shutdown error", "error", err)
		}
	}
	if s.stt != nil {
		if err := s.stt.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("stt shutdown error", "error", err)
		}
	}

	s.stt = nil
	s.tts = nil
	s.running = false
}

func (s *ExchangeSession) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	if s.running {
		s.stopLocked()
	}
}