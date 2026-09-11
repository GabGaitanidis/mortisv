package userprofile

import (
	"encoding/json"
	"os"
)


type ProfileStore struct {
	path string
}

func NewProfileStore(path string) (*ProfileStore, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(`{"fields":{}}`), 0o644); err != nil {
			return nil, err
		}
	}
	return &ProfileStore{path: path}, nil
}

func (s *ProfileStore) Load() (*UserProfile, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	profile := NewUserProfile()
	if err := json.Unmarshal(data, profile); err != nil {
		return nil, err
	}
	if profile.Fields == nil {
		profile.Fields = map[string][]string{}
	}
	return profile, nil
}

func (s *ProfileStore) Save(profile *UserProfile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
