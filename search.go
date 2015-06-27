package search

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"regexp"
	"time"

	"github.com/mholt/caddy/middleware"
	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/caddy-search/indexer/bleve"
)

// Handler creates a new handler for the search middleware
func Handler(next middleware.Handler, config *Config) middleware.Handler {
	if len(config.HostName) == 0 {
		return nil
	}

	index, err := NewIndexer(config.Engine, indexer.Config{Name: filepath.Clean(config.IndexDirectory + string(filepath.Separator) + config.HostName)})

	if err != nil {
		panic(err)
	}

	return &Search{next, config, index}
}

// Search represents this middleware structure
type Search struct {
	Next middleware.Handler
	*Config
	Indexer indexer.Handler
}

// ServerHTTP is the HTTP handler for this middleware
func (s *Search) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path == s.Config.JSONRoute {
		return s.ServeJSON(w, r)
	}

	// if url == s.Config.HTMLRoute {
	// 	return s.SearchHTML(w, r)
	// }

	if s.validatePath(r.URL.String()) {
		record := s.Indexer.Record(r.URL.String())

		code, err := s.Next.ServeHTTP(&searchResponseWriter{record, w, s, http.StatusOK}, r)
		if s.validateCode(code) {
			go s.Indexer.Pipe(record)
		}
		return code, err
	}
	return s.Next.ServeHTTP(w, r)
}

// Result is the structure for the search result
type Result struct {
	Path     string
	Modified time.Time
}

// ServeJSON ...
func (s *Search) ServeJSON(w http.ResponseWriter, r *http.Request) (status int, err error) {
	var jresp []byte

	q := r.URL.Query().Get("q")
	indexResult := s.Indexer.Search(q)

	results := make([]Result, len(indexResult))

	for i, result := range indexResult {
		results[i] = Result{result.Name(), result.Modified()}
	}

	jresp, err = json.Marshal(results)
	if err != nil {
		return 500, err
	}

	w.Write(jresp)
	return 200, err
}

// validatePath is the method that checks if the target page can be indexed
func (s *Search) validatePath(path string) bool {
	for _, p := range s.Config.ExcludePaths {
		if p.MatchString(path) {
			return false
		}
	}

	for _, p := range s.Config.IncludePaths {
		if p.MatchString(path) {
			return true
		}
	}

	return false
}

// validateCode is the method that checks if the response code can be indexed
func (s *Search) validateCode(code int) bool {
	if code != 200 {
		return false
	}

	return true
}

type searchResponseWriter struct {
	record indexer.Record
	http.ResponseWriter
	search *Search
	code   int
}

func (r *searchResponseWriter) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *searchResponseWriter) Write(p []byte) (n int, err error) {
	if r.search.validateCode(r.code) {
		go r.record.Write(p)
	}
	n, err = r.ResponseWriter.Write(p)
	return n, err
}

// NewIndexer creates a new Indexer with the received config
func NewIndexer(engine string, config indexer.Config) (index indexer.Handler, err error) {
	switch engine {
	case "bleve":
		index, err = bleve.New(config.Name)
		break
	default:
		index, err = bleve.New(config.Name)
		break
	}
	return
}

// Config represents this middleware configuration structure
type Config struct {
	HostName       string
	Engine         string
	Path           string
	IncludeCodes   []int
	ExcludeCodes   []int
	IncludePaths   []*regexp.Regexp
	ExcludePaths   []*regexp.Regexp
	HTMLRoute      string
	JSONRoute      string
	IndexDirectory string
}
