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
	Title() string
	SetTitle(string)
	Body() []byte
	SetBody([]byte)
	Modified() time.Time
	Load() bool
}
