package indexer

import (
	"bytes"
	"io"
	"time"
)

// Handler ...
type Handler interface {
	Record(string) Record
	Search(string) []Record
	Pipe(Record)
}

// Record ...
type Record interface {
	io.ReadWriter
	Name() string
	Body() bytes.Buffer
	Modified() time.Time
	Load() bool
}

// Config ...
type Config struct {
	HostName       string
	IndexDirectory string
}
