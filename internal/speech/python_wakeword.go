package speech

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"mortis/internal/logging"
)

const (
	StateWakeStarting = "wake_starting"
	StateWakeReady    = "wake_ready"
	StateWakeListening = "wake_listening"
)

type PythonWakeWord struct {
	pythonBin  string
	scriptPath string
	logger     *logging.Logger

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

type WakeWordBridge interface {
	Start(context.Context) error
	Listen() error
	Shutdown(context.Context) error
}

var _ WakeWordBridge = (*PythonWakeWord)(nil)

func NewPythonWakeWord(scriptPath string, logger *slog.Logger) *PythonWakeWord {
	var pythonBin = "/home/gabz/Desktop/projects/mortisgo/mortis-go/venv/bin/python3"
	return &PythonWakeWord{
		pythonBin:  pythonBin,
		scriptPath: scriptPath,
		logger:     logging.New(logger, "PythonWakeWord"),
	}
}

func (t *PythonWakeWord) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Info("starting subprocess", "state", StateWakeStarting, "script", t.scriptPath)

	cmd := exec.CommandContext(context.Background(), t.pythonBin, t.scriptPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("wakeword: stdin pipe: ", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("wakeword: stdout pipe: ", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("wakeword: start process: ", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.scanner = bufio.NewScanner(stdout)

	readyErr := make(chan error, 1)
	go func() {
		if !t.scanner.Scan() {
			readyErr <- fmt.Errorf("wakeword: process exited before READY: ", t.scanner.Err())
			return
		}
		if line := t.scanner.Text(); line != "READY" {
			readyErr <- fmt.Errorf("wakeword: unexpected startup line: ", line)
			return
		}
		readyErr <- nil
	}()

	select {
	case err := <-readyErr:
		if err != nil {
			_ = cmd.Process.Kill()
			return err
		}
		t.logger.Info("ready", "state", StateWakeReady)
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return fmt.Errorf("wakeword: startup cancelled: ", ctx.Err())
	}
}


func (t *PythonWakeWord) Listen() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return fmt.Errorf("wakeword: Listen called before Start")
	}

	t.logger.Info("listening for wake word", "state", StateWakeListening)

	if _, err := fmt.Fprintln(t.stdin, "listen"); err != nil {
		return fmt.Errorf("wakeword: write to subprocess: ", err)
	}

	if !t.scanner.Scan() {
		return fmt.Errorf("wakeword: subprocess closed unexpectedly: ", t.scanner.Err())
	}

	resp := t.scanner.Text()
	if resp != "detected" {
		return fmt.Errorf("wakeword: unexpected response: ", resp)
	}

	t.logger.Info("wake word detected", "state", StateWakeReady)
	return nil
}

func (t *PythonWakeWord) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return nil
	}

	t.logger.Info("shutting down")

	_, _ = fmt.Fprintln(t.stdin, "__EXIT__")
	_ = t.stdin.Close()

	done := make(chan error, 1)
	go func() { done <- t.cmd.Wait() }()

	select {
	case err := <-done:
		t.cmd = nil
		return err
	case <-ctx.Done():
		_ = t.cmd.Process.Kill()
		t.cmd = nil
		return fmt.Errorf("wakeword: shutdown timed out, process killed: ", ctx.Err())
	}
}