package browser

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
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

func (m *Module) Name() string { return "browser" }

func (m *Module) Execute(cmd core.Command, _ speech.TtsBridge) (string, error) {
	if cmd.Action == "current" {
		pageURL, err := activeURL()
		if err != nil {
			return "I could not read the active browser page", err
		}
		return pageURL, nil
	}
	if cmd.Action == "explain" {
		pageURL, err := activeURL()
		if err != nil {
			return "I could not read the active browser page", err
		}
		content, err := fetchPage(context.Background(), pageURL)
		if err != nil {
			return "I could not read that page", err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return m.client.Complete(ctx, "Explain the following webpage plainly and accurately. Mention when the page content could not be fully loaded. Return concise plain text only.", "URL: "+pageURL+"\n\nPAGE TEXT:\n"+content)
	}
	if cmd.Action != "open" {
		return "Unknown browser action", fmt.Errorf("browser: unknown action %q", cmd.Action)
	}

	target := cmd.GetString("url")
	if target == "" {
		target = cmd.GetString("query")
	}
	if target == "" {
		return "I need a URL or search query", nil
	}

	if parsed, err := url.Parse(target); err != nil || parsed.Scheme == "" {
		target = "https://www.google.com/search?q=" + url.QueryEscape(target)
	}

	command, args, err := openCommand(target)
	if err != nil {
		return "I cannot open the browser", err
	}
	if err := exec.Command(command, args...).Start(); err != nil {
		return "I cannot open the browser", err
	}
	return "Opened " + target, nil
}

func (m *Module) Description() string {
	return `browser
- open: params MUST include "url" (string) or "query" (string). Opens a URL or searches the web.
- current: returns the URL of the active tab in Google Chrome or Safari on macOS.
- explain: reads the active page's available HTML text and summarizes it. It cannot see screenshots, canvas content, or content requiring login.
Use module "browser" for websites, web searches, and explaining the active page.`
}
