package search_test

import (
	"os"
	"testing"

	"github.com/pedronasser/caddy-search"
	"github.com/pedronasser/caddy-search/indexer/bleve"
)

func BenchmarkPipeline(b *testing.B) {
	b.ReportAllocs()

	os.RemoveAll("/tmp/caddyIndexTest")
	indxr, err := bleve.New("/tmp/caddyIndexTest")

	if err != nil {
		b.Fatal(err)
	}

	pipeline, err2 := search.NewPipeline(&search.Config{}, indxr)
	if err2 != nil {
		b.Fatal(err)
	}

	cwd, _ := os.Getwd()
	path := cwd + "/README.md"

	for i := 0; i < b.N; i++ {
		rec := indxr.Record(path)
		rec.SetFullPath(path)
		pipeline.Pipe(rec)
	}
}
