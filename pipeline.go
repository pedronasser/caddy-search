package search

import (
	"bytes"
	"io"
	"path"
	"strings"
	"time"

	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/go-piper"
	"golang.org/x/net/html"
)

// NewPipeline creates a new Pipeline instance
func NewPipeline(config *Config, indexer indexer.Handler) (*Pipeline, error) {
	ppl := &Pipeline{
		config:  config,
		indexer: indexer,
	}

	pipe, err := piper.New(
		piper.P(1, ppl.validate),
		piper.P(1, ppl.parse),
		piper.P(1, ppl.index),
	)

	if err != nil {
		return nil, err
	}

	ppl.pipe = pipe

	go func() {
		tick := time.NewTicker(1 * time.Second)
		out := pipe.Output()
		for {
			select {
			case <-out:
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

// validate is the step of the pipeline that checks if documents are valid for
// being indexed
func (p *Pipeline) validate(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok {
		body := record.Body()
		if len(body) == 0 && !record.Ignored() {
			go p.Pipe(record)
			return nil
		}

		if p.ValidatePath(record.Path()) {
			return in
		}
		return nil
	}
	return nil
}

var titleTag = []byte("title")

// stripHTML returns s without HTML tags. It is fairly
// naive but works for most valid HTML inputs.
func stripHTML(s []byte) []byte {
	var buf bytes.Buffer
	var inTag, inQuotes bool
	var tagStart int
	for i, ch := range s {
		if inTag {
			if ch == '>' && !inQuotes {
				inTag = false
			} else if ch == '<' && !inQuotes {
				// false start
				buf.Write(s[tagStart:i])
				tagStart = i
			} else if ch == '"' {
				inQuotes = !inQuotes
			}
			continue
		}
		if ch == '<' {
			inTag = true
			tagStart = i
			continue
		}
		buf.WriteByte(ch)
	}
	if inTag {
		// false start
		buf.Write(s[tagStart:])
		inTag = false
	}
	return buf.Bytes()
}

// parse is the step of the pipeline that tries to parse documents and get
// important information
func (p *Pipeline) parse(in interface{}) interface{} {
	if record, ok := in.(indexer.Record); ok {
		body := bytes.NewReader(record.Body())
		title, err := getHTMLContent(body, titleTag)
		if err == nil {
			// html file
			record.SetTitle(title)
			record.SetBody(stripHTML(record.Body()))
			return record
		} else if strings.HasSuffix(record.Path(), ".txt") || strings.HasSuffix(record.Path(), ".md") {
			// TODO: We can improve file type detection; this is a very limited subset of indexable file types
			// text or markdown file
			record.SetTitle(path.Base(record.Path()))
			record.SetBody(record.Body())
			return record
		} else {
			// only accept html files
			record.Ignore()
			return err
		}
	}

	return nil
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
