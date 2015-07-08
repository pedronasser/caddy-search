package bleve

import (
	"path/filepath"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/go-piper"
)

// New creates a new instance for this indexer
func New(config indexer.Config) (*bleveIndexer, error) {
	name := filepath.Clean(config.IndexDirectory + string(filepath.Separator) + config.HostName)

	blv, err := openIndex(name)
	if err != nil {
		return nil, err
	}

	indexer := &bleveIndexer{}

	pipe, err := piper.New(
		piper.P(1, indexer.index),
	)

	if err != nil {
		return nil, err
	}

	indexer.pipeline = pipe
	indexer.bleve = blv

	go consumeOutput(pipe)

	return indexer, nil
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
