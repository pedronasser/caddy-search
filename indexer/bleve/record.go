package bleve

import (
	"strconv"
	"sync"
	"time"
)

// Record handles indexer's data
type Record struct {
	indexer  *bleveIndexer
	path     string
	title    string
	document map[string]interface{}
	body     []byte
	loaded   bool
	modified time.Time
	mutex    sync.Mutex
}

// Path returns Record's path
func (r *Record) Path() string {
	return r.path
}

// Title returns Record's title
func (r *Record) Title() string {
	return r.title
}

// SetTitle replaces Record's title
func (r *Record) SetTitle(title string) {
	r.title = title
}

// Modified returns Record's Modified
func (r *Record) Modified() time.Time {
	return r.modified
}

// Body returns Record's body
func (r *Record) Body() []byte {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.body
}

// SetBody replaces Record's body
func (r *Record) SetBody(body []byte) {
	r.body = body
}

// Load this record from the indexer.
func (r *Record) Load() bool {
	doc, err := r.indexer.bleve.Document(r.path)
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

	if len(r.body) == 0 {
		r.body = result["Body"].([]byte)
	}

	r.title = string(result["Title"].([]byte))

	r.loaded = true

	return true
}

// Write is the writing method for a Record
func (r *Record) Write(p []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.body = append(r.body, p...)

	return len(r.body), nil
}
