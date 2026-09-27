package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mortis/internal/core"
	"mortis/internal/speech"
)

type Module struct {
	handler *Handler
}

func NewModule(client *http.Client) *Module {
	handler, err := NewHandler(client)
	return &Module{handler: handlerWithError(handler, err)}
}

func handlerWithError(handler *Handler, err error) *Handler {
	if err == nil {
		return handler
	}
	return &Handler{client: http.DefaultClient}
}

var _ core.Module = (*Module)(nil)

func (m *Module) Name() string { return "calendar" }

func (m *Module) Execute(cmd core.Command, _ speech.TtsBridge) (string, error) {
	if m.handler.accessToken == "" {
		return "Google Calendar is not configured; set GOOGLE_CALENDAR_ACCESS_TOKEN", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	switch cmd.Action {
	case "list":
		maxResults := 10
		if value := cmd.GetString("max_results"); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				maxResults = parsed
			}
		}
		events, err := m.handler.List(ctx, maxResults)
		if err != nil {
			return "I could not read the calendar", err
		}
		if len(events) == 0 {
			return "There are no upcoming calendar events", nil
		}
		data, _ := json.Marshal(events)
		return string(data), nil
	case "add", "update":
		event := eventFromCommand(cmd)
		if event.Summary == "" || event.Start.DateTime == "" || event.End.DateTime == "" {
			return "Calendar events need a summary, start, and end in RFC3339 format", nil
		}
		var result Event
		var err error
		if cmd.Action == "add" {
			result, err = m.handler.Add(ctx, event)
		} else {
			eventID := cmd.GetString("event_id")
			if eventID == "" {
				return "Updating an event requires event_id", nil
			}
			result, err = m.handler.Update(ctx, eventID, event)
		}
		if err != nil {
			return "I could not save that calendar event", err
		}
		return "Calendar event saved: " + result.Summary, nil
	case "delete":
		eventID := cmd.GetString("event_id")
		if eventID == "" {
			return "Deleting an event requires event_id", nil
		}
		if err := m.handler.Delete(ctx, eventID); err != nil {
			return "I could not delete that calendar event", err
		}
		return "Calendar event deleted", nil
	default:
		return "Unknown calendar action", fmt.Errorf("calendar: unknown action %q", cmd.Action)
	}
}

func eventFromCommand(cmd core.Command) Event {
	return Event{
		Summary:     cmd.GetString("summary"),
		Description: cmd.GetString("description"),
		Start:       DateTime{DateTime: cmd.GetString("start"), TimeZone: cmd.GetString("timezone")},
		End:         DateTime{DateTime: cmd.GetString("end"), TimeZone: cmd.GetString("timezone")},
	}
}

func (m *Module) Description() string {
	return `calendar
- list: params MAY include "max_results" (string number).
- add: params MUST include "summary", "start", and "end" as RFC3339 strings; MAY include "description" and "timezone".
- update: params MUST include "event_id", "summary", "start", and "end" as RFC3339 strings; MAY include "description" and "timezone".
- delete: params MUST include "event_id" (string).
Use module "calendar" for Google Calendar operations. Requires GOOGLE_CALENDAR_ACCESS_TOKEN.`
}
