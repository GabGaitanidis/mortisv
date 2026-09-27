package browser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

func openCommand(target string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{target}, nil
	case "linux":
		return "xdg-open", []string{target}, nil
	case "windows":
		return "cmd", []string{"/c", "start", "", target}, nil
	default:
		return "", nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func activeURL() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		chromeScript := `tell application "Google Chrome"
if (count of windows) is 0 then return ""
return URL of active tab of front window
end tell`
		if output, err := exec.Command("osascript", "-e", chromeScript).Output(); err == nil {
			value := strings.TrimSpace(string(output))
			if value != "" {
				return value, nil
			}
		}
		safariScript := `tell application "Safari"
if (count of windows) is 0 then return ""
return URL of current tab of front window
end tell`
		output, err := exec.Command("osascript", "-e", safariScript).Output()
		if err != nil {
			return "", fmt.Errorf("could not read active browser tab: %w", err)
		}
		value := strings.TrimSpace(string(output))
		if value == "" {
			return "", fmt.Errorf("no active browser tab found")
		}
		return value, nil
	case "linux":
		return "", fmt.Errorf("active browser detection on Linux requires a browser integration")
	default:
		return "", fmt.Errorf("active browser detection is unsupported on %s", runtime.GOOS)
	}
}

func fetchPage(ctx context.Context, pageURL string) (string, error) {
	parsed, err := url.Parse(pageURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("unsupported page URL: %s", pageURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mortis/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("page returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	return cleanHTML(string(body)), nil
}

var tagPattern = regexp.MustCompile(`(?s)<[^>]*>`)
var whitespacePattern = regexp.MustCompile(`\s+`)

func cleanHTML(raw string) string {
	text := tagPattern.ReplaceAllString(raw, " ")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = whitespacePattern.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	if len(text) > 12000 {
		text = text[:12000]
	}
	return text
}
