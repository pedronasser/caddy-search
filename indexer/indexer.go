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

// Record ...
type Record interface {
	io.ReadWriter
	Name() string
	Modified() time.Time
	Load() bool
}

// Config ...
type Config struct {
	HostName       string
	IndexDirectory string
}
