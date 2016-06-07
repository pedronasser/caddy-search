package search_test

import (
	"testing"
	"time"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
	"github.com/pedronasser/caddy-search"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	defaultPath = search.ConvertToRegExp([]string{"^/"})
	configCases = []struct {
		config      string
		expectConf  search.Config
		expectMsg   string
		expectMatch func(search.Config, search.Config)
	}{
		{
			`search`,
			search.Config{
				Endpoint:     "/search",
				IncludePaths: defaultPath,
			},
			"Should support `search` without any arguments",
			func(expected, result search.Config) {
				So(expected.Endpoint, ShouldEqual, result.Endpoint)
			},
		},
		{
			`search /path`,
			search.Config{
				IncludePaths: search.ConvertToRegExp([]string{"/path"}),
			},
			"Should support `search` with only one argument",
			func(expected, result search.Config) {
				So(expected.IncludePaths[0].String(), ShouldEqual, result.IncludePaths[0].String())
			},
		},
		{
			`search / /path`,
			search.Config{
				Endpoint: "/path",
			},
			"Should support `search` with two arguments",
			func(expected, result search.Config) {
				So(expected.Endpoint, ShouldEqual, result.Endpoint)
			},
		},
		{
			`search / /search {
				endpoint /search2
			}`,
			search.Config{
				Endpoint: "/search2",
			},
			"Should support `search` arguments and override configurations",
			func(expected, result search.Config) {
				So(expected.Endpoint, ShouldEqual, result.Endpoint)
			},
		},
		{
			`search {
				+path /path
				+path /otherPath
				-path /forbidden
			}`,
			search.Config{
				IncludePaths: search.ConvertToRegExp([]string{"/path", "/otherPath"}),
				ExcludePaths: search.ConvertToRegExp([]string{"/forbidden"}),
			},
			"Should `search` support multiple include and excludes",
			func(expected, result search.Config) {
				So(expected.IncludePaths[0].String(), ShouldEqual, result.IncludePaths[0].String())
				So(expected.IncludePaths[1].String(), ShouldEqual, result.IncludePaths[1].String())
				So(expected.ExcludePaths[0].String(), ShouldEqual, result.ExcludePaths[0].String())
			},
		},
		{
			`search {
				expire 1000
			}`,
			search.Config{
				Expire: 1000 * time.Second,
			},
			"Should `search` support multiple include and excludes",
			func(expected, result search.Config) {
				So(expected.Expire, ShouldEqual, result.Expire)
			},
		},
	}
)

func TestSearchSetup(t *testing.T) {
	for _, kase := range configCases {
		Convey("Given a Caddy controller with the search middleware", t, func() {
			c := caddy.NewTestController(kase.config)
			cnf := httpserver.GetConfig("")
			result, err := search.ParseSearchConfig(c, cnf)
			Convey("Should not receive an error when parsing", func() {
				So(err, ShouldBeNil)
			})
			Convey(kase.expectMsg, func() {
				kase.expectMatch(kase.expectConf, *result)
			})
		})
	}
}
