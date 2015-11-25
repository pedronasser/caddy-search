package bleve

import (
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

// Bleve's record data struct
type indexRecord struct {
	Path     string
	Title    string
	Body     string
	Modified string
	Indexed  string
}

// Record method get existent or creates a new Record to be saved/updated in the indexer
func (i *bleveIndexer) Record(path string) indexer.Record {
	record := &Record{
		indexer:  i,
		path:     path,
		title:    "",
		document: nil,
		body:     []byte{},
		loaded:   false,
		indexed:  time.Time{},
	}
	return record
}

// Search method lookup for records using a query
func (i *bleveIndexer) Search(q string) (records []indexer.Record) {
	query := bleve.NewQueryStringQuery(q)
	request := bleve.NewSearchRequest(query)
	request.Highlight = bleve.NewHighlight()
	result, err := i.bleve.Search(request)
	if err != nil { // an empty query would cause this
		return
	}

	for _, match := range result.Hits {
		rec := i.Record(match.ID)
		loaded := rec.Load()

		if !loaded {
			continue
		}

		if len(match.Fragments["Body"]) > 0 {
			rec.SetBody([]byte(match.Fragments["Body"][0]))
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

	if rec != nil && len(rec.body) > 0 {
		rec.SetIndexed(time.Now())

		r := indexRecord{
			Path:     rec.Path(),
			Title:    rec.Title(),
			Body:     string(rec.body),
			Modified: strconv.Itoa(int(rec.Modified().Unix())),
			Indexed:  strconv.Itoa(int(rec.Indexed().Unix())),
		}

		i.bleve.Index(rec.Path(), r)
	}

	return nil
}
