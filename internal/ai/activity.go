package ai

import (
	"fmt"
	"strings"
	"sync"
)

// ActivityRecord mirrors the Java record — struct tags replace Jackson's
// field-name mapping.
type ActivityRecord struct {
	ActivityName string `json:"activityName"`
	Module       string `json:"module"`
	Action       string `json:"action"`
	Target       string `json:"target"`
	Timestamp    string `json:"timestamp"`
}

const maxActivitySize = 15

// ActivityMemory mirrors the ArrayDeque-based ring buffer. It's mutated
// from the async profile-update goroutine and read from the main flow,
// so it needs a mutex — Java's single-threaded-per-request Deque usage
// got that for free, Go doesn't.
type ActivityMemory struct {
	mu         sync.Mutex
	activities []ActivityRecord // index 0 = most recent
}

func NewActivityMemory() *ActivityMemory {
	return &ActivityMemory{activities: make([]ActivityRecord, 0, maxActivitySize)}
}

func (m *ActivityMemory) Add(record ActivityRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activities = append([]ActivityRecord{record}, m.activities...)
	if len(m.activities) > maxActivitySize {
		m.activities = m.activities[:maxActivitySize]
	}
}

func (m *ActivityMemory) Recent() []ActivityRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ActivityRecord, len(m.activities))
	copy(out, m.activities)
	return out
}

func (m *ActivityMemory) ToPromptContext() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sb strings.Builder
	for _, r := range m.activities {
		sb.WriteString(fmt.Sprintf("- %s (%s.%s): %s\n", r.ActivityName, r.Module, r.Action, r.Target))
	}
	return sb.String()
}
