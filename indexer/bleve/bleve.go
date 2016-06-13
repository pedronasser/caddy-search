package bleve

import (
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/pedronasser/go-piper"
)

var recordPool = sync.Pool{
	New: func() interface{} {
		return &Record{}
	},
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0)
	},
}

// New creates a new instance for this indexer
func New(name string) (*bleveIndexer, error) {
	blv, err := openIndex(name)
	if err != nil {
		return nil, err
	}

	indxr := &bleveIndexer{}

	pipe, err := piper.New(
		piper.P(1, indxr.index),
	)

	if err != nil {
		return nil, err
	}

	indxr.pipeline = pipe
	indxr.bleve = blv

	go consumeOutput(pipe)

	return indxr, nil
}

func openIndex(name string) (bleve.Index, error) {
	textFieldMapping := bleve.NewTextFieldMapping()

	doc := bleve.NewDocumentMapping()
	doc.AddFieldMappingsAt("name", textFieldMapping)
	doc.AddFieldMappingsAt("body", textFieldMapping)
	doc.AddFieldMappingsAt("modied", textFieldMapping)

	indexMap := bleve.NewIndexMapping()
	indexMap.AddDocumentMapping("document", doc)

	blv, err := bleve.New(name, indexMap)

	if err != nil {
		blv, err = bleve.Open(name)
		if err != nil {
			return nil, err
		}
	}

	return blv, nil
}

func consumeOutput(pipe piper.Handler) {
	tick := time.NewTicker(1 * time.Second)
	out := pipe.Output()
	for {
		select {
		case <-out:
		case <-tick.C:
		}
	}
}
