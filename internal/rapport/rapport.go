package rapport

import (
	"bufio"
	"encoding/json"
	"os"
)

type Entry struct {
	Note      string `json:"note"`
	Tag       string `json:"tag"` 
	Timestamp string `json:"timestamp"`
}


type Store struct {
	path string
}

func NewStore(path string) (*Store, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	return &Store{path: path}, nil
}

func (s *Store) Add(entry Entry) error {
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

func (s *Store) LoadEntries() ([]Entry, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func (s *Store) Recent(n int) ([]Entry, error) {
	entries, err := s.LoadEntries()
	if err != nil {
		return nil, err
	}
	start := len(entries) - n
	if start < 0 {
		start = 0
	}
	return entries[start:], nil
}
