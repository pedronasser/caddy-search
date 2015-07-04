package search

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mholt/caddy/middleware"
	"github.com/pedronasser/caddy-search/indexer"
)

// Handler creates a new handler for the search middleware
func Handler(next middleware.Handler, config *Config) middleware.Handler {
	if len(config.HostName) == 0 {
		return nil
	}

	index, err := NewIndexer(config.Engine, indexer.Config{
		HostName:       config.HostName,
		IndexDirectory: config.IndexDirectory,
	})

	if err != nil {
		panic(err)
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
	s.Pipeline.Pipe(record)
	return s.Next.ServeHTTP(&searchResponseWriter{record, w}, r)
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
	record indexer.Record
	http.ResponseWriter
}

func (r *searchResponseWriter) WriteHeader(code int) {
	r.ResponseWriter.WriteHeader(code)
}

func (r *searchResponseWriter) Write(p []byte) (n int, err error) {
	log.Println("Writing...")
	go r.record.Write(p)
	n, err = r.ResponseWriter.Write(p)
	return n, err
}
