package search

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mholt/caddy/middleware"
	"github.com/pedronasser/caddy-search/indexer"
)

// Handler creates a new handler for the search middleware
func Handler(next middleware.Handler, config *Config, index indexer.Handler, pipeline *Pipeline) middleware.Handler {
	return &Search{next, config, index, pipeline}
}

// Search represents this middleware structure
type Search struct {
	Next middleware.Handler
	*Config
	Indexer indexer.Handler
	*Pipeline
}

// ServerHTTP is the HTTP handler for this middleware
func (s *Search) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if middleware.Path(r.URL.Path).Matches(s.Config.Endpoint) {
		if r.Header.Get("Accept") == "application/json" {
			return s.SearchJSON(w, r)
		}
		return s.SearchJSON(w, r)
	}

	record := s.Indexer.Record(r.URL.String())
	go s.Pipeline.Pipe(record)
	return s.Next.ServeHTTP(&searchResponseWriter{w, record}, r)
}

// Result is the structure for the search result
type Result struct {
	Path     string
	Title    string
	Body     string
	Modified time.Time
}

// SearchJSON ...
func (s *Search) SearchJSON(w http.ResponseWriter, r *http.Request) (status int, err error) {
	var jresp []byte

	q := r.URL.Query().Get("q")
	indexResult := s.Indexer.Search(q)

	results := make([]Result, len(indexResult))

	for i, result := range indexResult {
		body := result.Body()
		results[i] = Result{
			Path:     result.Path(),
			Title:    result.Title(),
			Modified: result.Modified(),
			Body:     string(body),
		}
	}

	jresp, err = json.Marshal(results)
	if err != nil {
		return 500, err
	}

	w.Write(jresp)
	return 200, err
}

type searchResponseWriter struct {
	w      http.ResponseWriter
	record indexer.Record
}

func (r *searchResponseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *searchResponseWriter) WriteHeader(code int) {
	if code != 200 {
		r.record.Ignore()
	}
	r.w.WriteHeader(code)
}

func (r *searchResponseWriter) Write(p []byte) (int, error) {
	defer r.record.Write(p)
	n, err := r.w.Write(p)
	return n, err
}
