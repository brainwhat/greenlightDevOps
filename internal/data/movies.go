package data

import "time"

type Movie struct {
	ID        int64     `json:"id"` // This just changes output names
	CreatedAt time.Time `json:"-"`  // "-" doen't show field in json response
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"` // omitempty doesn't show field if it's not defined/zero/""/false/etc
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}
