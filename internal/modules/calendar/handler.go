package calendar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const defaultCalendarAPI = "https://www.googleapis.com/calendar/v3"

type Handler struct {
	client      *http.Client
	accessToken string
	calendarID  string
	baseURL     string
}

func NewHandler(client *http.Client) (*Handler, error) {
	accessToken := strings.TrimSpace(os.Getenv("GOOGLE_CALENDAR_ACCESS_TOKEN"))
	if accessToken == "" {
		return nil, fmt.Errorf("GOOGLE_CALENDAR_ACCESS_TOKEN is not set")
	}
	calendarID := os.Getenv("GOOGLE_CALENDAR_ID")
	if calendarID == "" {
		calendarID = "primary"
	}
	baseURL := strings.TrimRight(os.Getenv("GOOGLE_CALENDAR_API_URL"), "/")
	if baseURL == "" {
		baseURL = defaultCalendarAPI
	}
	return &Handler{client: client, accessToken: accessToken, calendarID: calendarID, baseURL: baseURL}, nil
}

type Event struct {
	ID          string   `json:"id,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Start       DateTime `json:"start"`
	End         DateTime `json:"end"`
}

type DateTime struct {
	DateTime string `json:"dateTime,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
	Date     string `json:"date,omitempty"`
}

type eventList struct {
	Items []Event `json:"items"`
}

func (h *Handler) request(ctx context.Context, method, path string, body any, result any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("google calendar returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

func (h *Handler) List(ctx context.Context, maxResults int) ([]Event, error) {
	if maxResults <= 0 || maxResults > 250 {
		maxResults = 10
	}
	path := "/calendars/" + url.PathEscape(h.calendarID) + "/events?singleEvents=true&orderBy=startTime&maxResults=" + fmt.Sprint(maxResults)
	var result eventList
	if err := h.request(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (h *Handler) Add(ctx context.Context, event Event) (Event, error) {
	var result Event
	err := h.request(ctx, http.MethodPost, "/calendars/"+url.PathEscape(h.calendarID)+"/events", event, &result)
	return result, err
}

func (h *Handler) Update(ctx context.Context, eventID string, event Event) (Event, error) {
	var result Event
	path := "/calendars/" + url.PathEscape(h.calendarID) + "/events/" + url.PathEscape(eventID)
	err := h.request(ctx, http.MethodPut, path, event, &result)
	return result, err
}

func (h *Handler) Delete(ctx context.Context, eventID string) error {
	path := "/calendars/" + url.PathEscape(h.calendarID) + "/events/" + url.PathEscape(eventID)
	return h.request(ctx, http.MethodDelete, path, nil, nil)
}
