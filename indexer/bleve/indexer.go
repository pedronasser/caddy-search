package bleve

import (
	"bytes"
	"strconv"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/go-piper"
)

type bleveIndexer struct {
	pipeline piper.Handler
	bleve    bleve.Index
}

type indexRecord struct {
	Name     string
	Body     string
	Modified string
}

// Record method get existent or creates a new Record to be saved/updated in the indexer
func (i *bleveIndexer) Record(name string) indexer.Record {
	record := &Record{i, name, nil, bytes.NewBuffer(nil), false, time.Now()}
	return record
}

// Search method lookup for records using a query
func (i *bleveIndexer) Search(q string) (records []indexer.Record) {
	query := bleve.NewQueryStringQuery(q)
	request := bleve.NewSearchRequest(query)
	result, _ := i.bleve.Search(request)

	for _, match := range result.Hits {
		rec := i.Record(match.ID)
		loaded := rec.Load()

		if !loaded {
			continue
		}

		records = append(records, rec)
	}

	return
}

// Pipe sends the new record to the pipeline
func (i *bleveIndexer) Pipe(r indexer.Record) {
	i.pipeline.Input() <- r
}

// index is the pipeline step that indexes the document
func (i *bleveIndexer) index(in interface{}) interface{} {
	var rec *Record

	if _, ok := in.(*Record); ok {
		rec = in.(*Record)
	}

	if rec != nil && rec.body.Len() > 0 {
		r := indexRecord{
			Name:     rec.Name(),
			Body:     rec.body.String(),
			Modified: strconv.Itoa(int(time.Now().Unix())),
		}

		i.bleve.Index(rec.Name(), r)
	}

	return nil
}
