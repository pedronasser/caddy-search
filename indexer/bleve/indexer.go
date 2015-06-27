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

func (i *bleveIndexer) newRecord(name string) *Record {
	return &Record{name, nil, bytes.NewBuffer(nil), false, time.Now()}
}

func (i *bleveIndexer) loadRecord(record *Record) bool {
	doc, err := i.bleve.Document(record.name)
	if err != nil || doc == nil {
		record.loaded = true
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

	record.modified = time.Unix(int64(modified), 0)
	record.document = result

	if record.body.Len() == 0 {
		body := result["Body"].([]byte)
		if len(body) > 0 {
			record.body = bytes.NewBuffer(body)
		}
	}

	record.loaded = true

	return true
}

// Record method get existent or creates a new Record to be saved/updated in the indexer
func (i *bleveIndexer) Record(name string) indexer.Record {
	record := i.newRecord(name)
	go i.loadRecord(record)
	return record
}

// Search method lookup for records using a query
func (i *bleveIndexer) Search(q string) (records []indexer.Record) {
	query := bleve.NewQueryStringQuery(q)
	request := bleve.NewSearchRequest(query)
	result, _ := i.bleve.Search(request)

	for _, match := range result.Hits {
		rec := i.newRecord(match.ID)
		loaded := i.loadRecord(rec)

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
		if rec.loaded == false {
			go func() {
				i.pipeline.Input() <- rec
			}()
			return rec
		}

		r := indexRecord{
			Name:     rec.Name(),
			Body:     rec.body.String(),
			Modified: strconv.Itoa(int(time.Now().Unix())),
		}

		i.bleve.Index(rec.Name(), r)
	}

	return nil
}
