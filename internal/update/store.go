package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// Record is what the updater remembers between runs.
type Record struct {
	CheckedAt time.Time     `json:"checkedAt,omitempty"`
	ETag      string        `json:"etag,omitempty"`
	Latest    *SavedRelease `json:"latest,omitempty"`
	Pending   *Pending      `json:"pending,omitempty"`
}

// SavedRelease is the last release seen, kept so a conditional check can reuse it.
type SavedRelease struct {
	Tag   string `json:"tag"`
	Notes string `json:"notes,omitempty"`
	URL   string `json:"url,omitempty"`
}

// Pending is an update handed to the installer. The next launch reads it to
// greet the user and remove the download.
type Pending struct {
	From string `json:"from"`
	To   string `json:"to"`
	Dir  string `json:"dir,omitempty"`
}

func saveRelease(r Release) *SavedRelease {
	return &SavedRelease{Tag: r.Tag, Notes: r.Notes, URL: r.URL}
}

// Release returns the saved release, or false when it is missing or unreadable.
func (s *SavedRelease) Release() (Release, bool) {
	if s == nil {
		return Release{}, false
	}
	v, ok := ParseVersion(s.Tag)
	if !ok {
		return Release{}, false
	}
	return Release{Version: v, Tag: s.Tag, Notes: s.Notes, URL: s.URL}, true
}

// Store keeps the Record in a JSON file beside the settings.
type Store struct{ Path string }

// Load reads the record. A missing or damaged file is an empty record.
func (s Store) Load() Record {
	var r Record
	b, err := os.ReadFile(s.Path)
	if err != nil {
		return Record{}
	}
	if json.Unmarshal(b, &r) != nil {
		return Record{}
	}
	return r
}

// Save writes the record.
func (s Store) Save(r Record) error {
	if s.Path == "" {
		return errors.New("no update state path")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}
