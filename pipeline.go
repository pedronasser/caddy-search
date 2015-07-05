package search

import (
	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/go-piper"
)

// NewPipeline creates a new Pipeline instance
func NewPipeline(config *Config, indexer indexer.Handler) (*Pipeline, error) {
	ppl := &Pipeline{
		config:  config,
		indexer: indexer,
	}

	pipe, err := piper.New(
		piper.P(1, ppl.validate),
		piper.P(1, ppl.index),
	)

	if err != nil {
		return nil, err
	}

	ppl.pipe = pipe

	go func() {
		out := pipe.Output()
		for {
			<-out
		}
	}()

	return ppl, nil
}

// Pipeline is the structure that holds search's pipeline infos and methods
type Pipeline struct {
	config  *Config
	indexer indexer.Handler
	pipe    piper.Handler
}

// Pipe is the step of the pipeline that pipes valid documents to the indexer.
func (p *Pipeline) Pipe(record indexer.Record) {
	p.pipe.Input() <- record
}

// Validate is the step of the pipeline that checks if documents are valid for
// being indexed
func (p *Pipeline) validate(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok {
		if p.ValidatePath(record.Name()) {
			return in
		}
		return nil
	}
	return nil
}

// Pipe is the step of the pipeline that pipes valid documents to the indexer.
func (p *Pipeline) index(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok {
		body := record.Body()
		if body.Len() == 0 {
			p.Pipe(record)
			return nil
		}
		go p.indexer.Pipe(record)
		return in
	}
	return nil
}

// ValidatePath is the method that checks if the target page can be indexed
func (p *Pipeline) ValidatePath(path string) bool {
	for _, pa := range p.config.ExcludePaths {
		if pa.MatchString(path) {
			return false
		}
	}

	for _, pa := range p.config.IncludePaths {
		if pa.MatchString(path) {
			return true
		}
	}

	return false
}
