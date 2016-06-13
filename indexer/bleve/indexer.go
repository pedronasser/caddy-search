package bleve

import (
	"fmt"
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
	record := recordPool.Get().(*Record)
	record.path = path
	record.fullPath = ""
	record.title = ""
	record.document = make(map[string]interface{})
	record.ignored = false
	record.loaded = false
	record.body = bufPool.Get().([]byte)
	record.indexed = time.Time{}
	record.modified = time.Time{}
	record.indexer = i
	return record
}

func (i *bleveIndexer) Kill(r indexer.Record) {
	bufPool.Put(r.Body())
	recordPool.Put(r)
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
	if rec, ok := in.(*Record); ok {

		if rec != nil && len(rec.body) > 0 && !rec.Ignored() {
			rec.SetIndexed(time.Now())
			fmt.Println(rec.FullPath())

			r := indexRecord{
				Path:     rec.Path(),
				Title:    rec.Title(),
				Body:     string(rec.body),
				Modified: strconv.Itoa(int(rec.Modified().Unix())),
				Indexed:  strconv.Itoa(int(rec.Indexed().Unix())),
			}

			i.bleve.Index(rec.Path(), r)
		}

		i.Kill(rec)
	}

	return in
}
