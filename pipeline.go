package search

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/go-piper"
	"golang.org/x/net/html"
)

var bm = bluemonday.UGCPolicy()

// NewPipeline creates a new Pipeline instance
func NewPipeline(config *Config, indxr indexer.Handler) (*Pipeline, error) {
	ppl := &Pipeline{
		config:  config,
		indexer: indxr,
	}

	pipe, err := piper.New(
		piper.P(1, ppl.read),
		piper.P(1, ppl.validate),
		piper.P(1, ppl.parse),
		piper.P(1, ppl.index),
	)

	if err != nil {
		return nil, err
	}

	ppl.pipe = pipe

	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		out := pipe.Output()
		for {
			select {
			case in := <-out:
				if record, ok := in.(indexer.Record); ok {
					if record.Ignored() {
						ppl.indexer.Kill(record)
					}
				}
			case <-tick.C:
			}
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

// Piper is a func that returns the piper.Handler
func (p *Pipeline) Piper() piper.Handler {
	return p.pipe
}

// validate is the step of the pipeline that reads the file content
func (p *Pipeline) read(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok && !record.Ignored() {
		in, err := os.Open(record.FullPath())
		defer in.Close()

		if err != nil {
			record.Ignore()
		} else {
			io.Copy(record, in)
		}
	}

	return in
}

// validate is the step of the pipeline that checks if documents are valid for
// being indexed
func (p *Pipeline) validate(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok && !record.Ignored() {
		if !p.ValidatePath(record.Path()) {
			record.Ignore()
		}
	}

	return in
}

var titleTag = []byte("title")

// parse is the step of the pipeline that tries to parse documents and get
// important information
func (p *Pipeline) parse(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok && !record.Ignored() {
		if strings.HasSuffix(record.Path(), ".txt") || strings.HasSuffix(record.Path(), ".md") {
			// TODO: We can improve file type detection; this is a very limited subset of indexable file types
			// text or markdown file
			record.SetTitle(path.Base(record.Path()))
		} else {
			body := bytes.NewReader(record.Body())
			title, err := getHTMLContent(body, titleTag)
			if err == nil {
				// html file
				record.SetTitle(title)
				stripped := bm.SanitizeBytes(record.Body())
				record.SetBody(stripped)
			} else {
				record.Ignore()
			}
		}
	}

	return in
}

func getHTMLContent(r io.Reader, tag []byte) (result string, err error) {
	z := html.NewTokenizer(r)
	result = ""
	valid := 0
	cacheLen := len(tag)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			err = z.Err()
			return
		case html.TextToken:
			if valid == 1 {
				return string(z.Text()), nil
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			if len(tn) == cacheLen && bytes.Equal(tn[0:cacheLen], tag) {
				if tt == html.StartTagToken {
					valid = 1
				} else {
					valid = 0
				}
			}
		}
	}
}

// index is the step of the pipeline that pipes valid documents to the indexer.
func (p *Pipeline) index(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok {
		if !record.Ignored() {
			p.indexer.Pipe(record)
		}
	}
	return in
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
