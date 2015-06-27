package bleve

import (
	"bytes"
	"time"
)

// Record handles indexer's data
type Record struct {
	name     string
	document map[string]interface{}
	body     *bytes.Buffer
	loaded   bool
	modified time.Time
}

// Name returns Record's name
func (r *Record) Name() string {
	return r.name
}

// Modified returns Record's Modified
func (r *Record) Modified() time.Time {
	return r.modified
}

// Write is the writing method for a Record
func (r *Record) Write(p []byte) (n int, err error) {
	if r.body.Len() == 0 {
		r.body = bytes.NewBuffer(p)
		return r.body.Len(), nil
	}
	return
}

// Read is the reading method for a Record
func (r *Record) Read(p []byte) (n int, err error) {
	return r.body.Read(p)
}
