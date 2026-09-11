package ai

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"strings"
)

type Memory struct {
	path string
}

func NewMemory(path string) (*Memory, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	return &Memory{path: path}, nil
}

func (m *Memory) Add(record ActivityRecord) error {
	f, err := os.OpenFile(m.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}


func (m *Memory) LoadRecords() ([]ActivityRecord, error) {
	f, err := os.Open(m.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var records []ActivityRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var r ActivityRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			continue
		}
		records = append(records, r)
	}
	return records, scanner.Err()
}

func (m *Memory) RecentRecords(n int) ([]ActivityRecord, error) {
	records, err := m.LoadRecords()
	if err != nil {
		return nil, err
	}
	start := len(records) - n
	if start < 0 {
		start = 0
	}
	return records[start:], nil
}

func (m *Memory) SearchRelevant(query string, limit int) ([]ActivityRecord, error) {
	records, err := m.LoadRecords()
	if err != nil {
		return nil, err
	}

	type scored struct {
		record ActivityRecord
		score  float64
	}

	var candidates []scored
	for _, r := range records {
		if score := tokenOverlap(query, r.ActivityName); score > 0.1 {
			candidates = append(candidates, scored{r, score})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })

	if limit > len(candidates) {
		limit = len(candidates)
	}
	out := make([]ActivityRecord, limit)
	for i := 0; i < limit; i++ {
		out[i] = candidates[i].record
	}
	return out, nil
}

func tokenOverlap(a, b string) float64 {
	ta := strings.Fields(strings.ToLower(a))
	tb := strings.Fields(strings.ToLower(b))
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	setB := make(map[string]bool, len(tb))
	for _, t := range tb {
		setB[t] = true
	}
	matches := 0
	for _, t := range ta {
		if setB[t] {
			matches++
		}
	}
	return float64(matches) / float64(len(ta))
}
