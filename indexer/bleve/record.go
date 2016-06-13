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
	fullPath string
	title    string
	document map[string]interface{}
	body     []byte
	loaded   bool
	modified time.Time
	mutex    sync.RWMutex
	ignored  bool
	indexed  time.Time
}

// Path returns Record's path
func (r *Record) Path() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.path
}

// FullPath returns Record's fullpath
func (r *Record) FullPath() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.fullPath
}

// SetFullPath defines a new fullpath for the record
func (r *Record) SetFullPath(fp string) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	r.fullPath = fp
}

// Title returns Record's title
func (r *Record) Title() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.title
}

// SetTitle replaces Record's title
func (r *Record) SetTitle(title string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.title = title
}

// Modified returns Record's Modified
func (r *Record) Modified() time.Time {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.modified
}

// SetModified defines new modification time for this record
func (r *Record) SetModified(mod time.Time) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.modified = mod
}

// SetBody replaces the actual body
func (r *Record) SetBody(body []byte) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.body = body
}

// Body returns Record's body
func (r *Record) Body() []byte {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.body
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

	strIndexed := string(result["Indexed"].([]byte))
	indexed, err := strconv.Atoi(strIndexed)

	r.indexed = time.Unix(int64(indexed), 0)

	r.document = result

	if len(r.body) == 0 {
		r.Write(result["Body"].([]byte))
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

// Ignore flag this record as ignored
func (r *Record) Ignore() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.ignored = true
}

// Ignored returns if this record is ignored
func (r *Record) Ignored() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.ignored
}

// Indexed returns the indexing time (if indexed)
func (r *Record) Indexed() time.Time {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.indexed
}

// SetIndexed define the time that this record has been indexed
func (r *Record) SetIndexed(index time.Time) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.indexed = index
}
