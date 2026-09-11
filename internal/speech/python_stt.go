package speech

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)


const (
	StateSttStarting = "stt_starting"
	StateSttReady    = "stt_ready"
	StateListening   = "stt_listening"
)


type PythonSTT struct {
	pythonBin  string
	scriptPath string
	logger     *slog.Logger

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

var _ SttBridge = (*PythonSTT)(nil)

func NewPythonSTT(pythonBin, scriptPath string, logger *slog.Logger) *PythonSTT {
	if pythonBin == "" {
		pythonBin = "python3"
	}
	return &PythonSTT{
		pythonBin:  pythonBin,
		scriptPath: scriptPath,
		logger:     logger,
	}
}


func (t *PythonSTT) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Info("stt: starting subprocess", "state", StateSttStarting, "script", t.scriptPath)

	cmd := exec.CommandContext(context.Background(), t.pythonBin, t.scriptPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stt: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stt: stdout pipe: %w", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("stt: start process: %w", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.scanner = bufio.NewScanner(stdout)

	readyErr := make(chan error, 1)
	go func() {
		if !t.scanner.Scan() {
			readyErr <- fmt.Errorf("stt: process exited before READY: %w", t.scanner.Err())
			return
		}
		if line := t.scanner.Text(); line != "READY" {
			readyErr <- fmt.Errorf("stt: unexpected startup line: %q", line)
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
		t.logger.Info("stt: ready", "state", StateSttReady)
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return fmt.Errorf("stt: startup cancelled: %w", ctx.Err())
	}
}

func (t *PythonSTT) Listen() (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return "", fmt.Errorf("stt: Listen called before Start")
	}

	t.logger.Info("stt: listening", "state", StateListening)

	if _, err := fmt.Fprintln(t.stdin, "listen"); err != nil {
		return "", fmt.Errorf("stt: write to subprocess: %w", err)
	}

	if !t.scanner.Scan() {
		return "", fmt.Errorf("stt: subprocess closed unexpectedly: %w", t.scanner.Err())
	}

	resp := t.scanner.Text()
	if strings.HasPrefix(resp, "ERROR:") {
		return "", fmt.Errorf("stt: %s", strings.TrimSpace(strings.TrimPrefix(resp, "ERROR:")))
	}

	t.logger.Info("stt: transcript received", "state", StateSttReady, "transcript_len", len(resp))
	return resp, nil
}

func (t *PythonSTT) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return nil
	}

	t.logger.Info("stt: shutting down")

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
		return fmt.Errorf("stt: shutdown timed out, process killed: %w", ctx.Err())
	}
}
