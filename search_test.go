package search_test

import (
	"testing"

	"net/http"

	"net/http/httptest"

	"github.com/mholt/caddy"
	"github.com/pedronasser/caddy-search"
)

func BenchmarkSearch(b *testing.B) {
	c := caddy.NewTestController(configCases[0].config)
	search.Setup(c)

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/search?q=test", nil)
		resp := httptest.NewRecorder()
		c.ServerBlockStorage.(*search.Search).ServeHTTP(resp, req)
	}

	c.ServerBlockStorage.(*search.Search).Pipeline.Piper().Close()
}
