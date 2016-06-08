package indexer

import (
	"io"
	"time"
)

// Handler ...
type Handler interface {
	Record(string) Record
	Search(string) []Record
	Pipe(Record)
}

// Config ...
type Config struct {
	HostName       string
	IndexDirectory string
}

// Record ...
type Record interface {
	io.Writer
	Path() string
	FullPath() string
	SetFullPath(string)
	Title() string
	SetTitle(string)
	Body() []byte
	SetBody([]byte)
	SetModified(time.Time)
	Modified() time.Time
	Load() bool
	Ignore()
	Ignored() bool
	Indexed() time.Time
}
