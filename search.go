package search

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mholt/caddy/middleware"
	"github.com/pedronasser/caddy-search/indexer"
)

// Handler creates a new handler for the search middleware
func Handler(next middleware.Handler, config *Config, index indexer.Handler) middleware.Handler {
	if len(config.HostName) == 0 {
		return nil
	}

	ppl, err := NewPipeline(config, index)

	if err != nil {
		panic(err)
	}

	return &Search{next, config, ppl, index}
}

// Search represents this middleware structure
type Search struct {
	Next middleware.Handler
	*Config
	*Pipeline
	Indexer indexer.Handler
}

// ServerHTTP is the HTTP handler for this middleware
func (s *Search) ServeHTTP(w http.ResponseWriter, r *http.Request) (int, error) {
	if r.URL.Path == s.Config.Endpoint {
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
	Modified time.Time
}

// SearchJSON ...
func (s *Search) SearchJSON(w http.ResponseWriter, r *http.Request) (status int, err error) {
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

type searchResponseWriter struct {
	w      http.ResponseWriter
	record indexer.Record
}

func (r *searchResponseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *searchResponseWriter) WriteHeader(code int) {
	r.w.WriteHeader(code)
}

func (r *searchResponseWriter) Write(p []byte) (int, error) {
	go r.record.Write(p)
	n, err := r.w.Write(p)
	return n, err
}
