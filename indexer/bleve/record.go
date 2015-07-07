package bleve

import (
	"bytes"
	"strconv"
	"sync"
	"time"
)

// Record handles indexer's data
type Record struct {
	indexer  *bleveIndexer
	name     string
	document map[string]interface{}
	body     *bytes.Buffer
	loaded   bool
	modified time.Time
	mutex    sync.Mutex
}

// Name returns Record's name
func (r *Record) Name() string {
	return r.name
}

// Modified returns Record's Modified
func (r *Record) Modified() time.Time {
	return r.modified
}

// Body returns Record's body
func (r *Record) Body() bytes.Buffer {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return *r.body
}

// Load this record from the indexer.
func (r *Record) Load() bool {
	doc, err := r.indexer.bleve.Document(r.name)
	if err != nil || doc == nil {
		r.loaded = true
		return false
	}

	result := make(map[string]interface{})

	for _, field := range doc.Fields {
		name := field.Name()
		value := field.Value()
		result[name] = value
	}

	strModified := string(result["Modified"].([]byte))
	modified, err := strconv.Atoi(strModified)

	r.modified = time.Unix(int64(modified), 0)
	r.document = result

	if r.body.Len() == 0 {
		body := result["Body"].([]byte)
		if len(body) > 0 {
			r.body = bytes.NewBuffer(body)
		}
	}

	r.loaded = true

	return true
}

// Write is the writing method for a Record
func (r *Record) Write(p []byte) (n int, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
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
