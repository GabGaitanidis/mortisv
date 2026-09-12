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
	StateStarting = "tts_starting"
	StateReady    = "tts_ready"
	StateSpeaking = "tts_speaking"
	StateStopped  = "tts_stopped"
)

type PythonTTS struct {
	pythonBin  string
	scriptPath string
	logger     *slog.Logger

	mu      sync.Mutex
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

var _ TtsBridge = (*PythonTTS)(nil)

func NewPythonTTS(scriptPath string, logger *slog.Logger) *PythonTTS {
	var pythonBin = "/home/gabz/Desktop/projects/mortisgo/mortis-go/venv/bin/python3"

	return &PythonTTS{
		pythonBin:  pythonBin,
		scriptPath: scriptPath,
		logger:     logger,
	}
}


func (t *PythonTTS) Start(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.logger.Info("tts: starting subprocess", "state", StateStarting, "script", t.scriptPath)

	cmd := exec.CommandContext(context.Background(), t.pythonBin, t.scriptPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("tts: stdin pipe: ", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("tts: stdout pipe: ", err)
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("tts: start process: ", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.scanner = bufio.NewScanner(stdout)

	readyErr := make(chan error, 1)
	go func() {
		if !t.scanner.Scan() {
			readyErr <- fmt.Errorf("tts: process exited before READY: ", t.scanner.Err())
			return
		}
		if line := t.scanner.Text(); line != "READY" {
			readyErr <- fmt.Errorf("tts: unexpected startup line: ", line)
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
		t.logger.Info("tts: ready", "state", StateReady)
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return fmt.Errorf("tts: startup cancelled: ", ctx.Err())
	}
}


func (t *PythonTTS) Speak(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return fmt.Errorf("tts: Speak called before Start")
	}

	line := strings.ReplaceAll(text, "\n", " ")

	t.logger.Info("tts: speaking", "state", StateSpeaking, "text_len", len(line))

	if _, err := fmt.Fprintln(t.stdin, line); err != nil {
		return fmt.Errorf("tts: write to subprocess: ", err)
	}

	if !t.scanner.Scan() {
		return fmt.Errorf("tts: subprocess closed unexpectedly: ", t.scanner.Err())
	}

	resp := t.scanner.Text()
	if strings.HasPrefix(resp, "ERROR:") {
		return fmt.Errorf("tts: ", strings.TrimSpace(strings.TrimPrefix(resp, "ERROR:")))
	}
	if resp != "DONE" {
		return fmt.Errorf("tts: unexpected response: ", resp)
	}

	t.logger.Info("tts: done speaking", "state", StateReady)
	return nil
}

func (t *PythonTTS) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cmd == nil {
		return nil
	}

	t.logger.Info("tts: shutting down", "state", StateStopped)

	
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
		return fmt.Errorf("tts: shutdown timed out, process killed: ", ctx.Err())
	}
}
