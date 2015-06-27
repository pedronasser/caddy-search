package bleve

import (
	"github.com/blevesearch/bleve"
	"github.com/pedronasser/go-piper"
)

// New creates a new instance for this indexer
func New(name string) (*bleveIndexer, error) {
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
	for {
		<-pipe.Output()
	}
}
