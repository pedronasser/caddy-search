package search

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mholt/caddy/middleware"
	"github.com/pedronasser/caddy-search/indexer"
)

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
		if r.Header.Get("Accept") == "application/json" || s.Config.Template == nil {
			return s.SearchJSON(w, r)
		}
		return s.SearchHTML(w, r)
	}

	record := s.Indexer.Record(r.URL.String())

	status, err := s.Next.ServeHTTP(&searchResponseWriter{w, record}, r)

	modif := w.Header().Get("Last-Modified")
	if len(modif) > 0 {
		modTime, err := time.Parse(`Mon, 2 Jan 2006 15:04:05 MST`, modif)
		if err == nil {
			record.SetModified(modTime)
		}
	}

	if status != http.StatusOK {
		record.Ignore()
	}

	go s.Pipeline.Pipe(record)

	return status, err
}

// Result is the structure for the search result
type Result struct {
	Path     string
	Title    string
	Body     string
	Modified time.Time
	Indexed  time.Time
}

// SearchJSON renders the search results in JSON format
func (s *Search) SearchJSON(w http.ResponseWriter, r *http.Request) (int, error) {
	q := r.URL.Query().Get("q")
	indexResult := s.Indexer.Search(q)

	results := make([]Result, len(indexResult))

	for i, result := range indexResult {
		body := result.Body()
		results[i] = Result{
			Path:     result.Path(),
			Title:    result.Title(),
			Modified: result.Modified(),
			Indexed:  result.Indexed(),
			Body:     string(body),
		}
	}

	jresp, err := json.Marshal(results)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	w.Write(jresp)
	return http.StatusOK, err
}

// SearchHTML renders the search results in the HTML template
func (s *Search) SearchHTML(w http.ResponseWriter, r *http.Request) (int, error) {
	q := r.URL.Query().Get("q")
	indexResult := s.Indexer.Search(q)

	results := make([]Result, len(indexResult))

	for i, result := range indexResult {
		results[i] = Result{
			Path:     result.Path(),
			Title:    result.Title(),
			Modified: result.Modified(),
			Body:     string(result.Body()),
		}
	}

	qresults := QueryResults{
		Context: middleware.Context{
			Root: http.Dir(s.SiteRoot),
			Req:  r,
			URL:  r.URL,
		},
		Query:   q,
		Results: results,
	}

	var buf bytes.Buffer
	err := s.Config.Template.Execute(&buf, qresults)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	buf.WriteTo(w)
	return http.StatusOK, nil
}

type QueryResults struct {
	middleware.Context
	Query   string
	Results []Result
}

type searchResponseWriter struct {
	w      http.ResponseWriter
	record indexer.Record
}

func (r *searchResponseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *searchResponseWriter) WriteHeader(code int) {
	if code != http.StatusOK {
		r.record.Ignore()
	}
	r.w.WriteHeader(code)
}

func (r *searchResponseWriter) Write(p []byte) (int, error) {
	defer r.record.Write(p)
	n, err := r.w.Write(p)
	return n, err
}
