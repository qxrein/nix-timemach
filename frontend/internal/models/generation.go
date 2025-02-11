package models

import "time"

type Generation struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Profiles    []string  `json:"profiles"`
	Selected    bool      `json:"-"`
}

type GenerationDiff struct {
	Added    []string
	Removed  []string
	Modified []string
}
