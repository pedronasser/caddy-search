package search

import (
	"testing"
	"time"

	"github.com/mholt/caddy/config/setup"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	defaultPath = convertToRegExp([]string{"^/"})
	configCases = []struct {
		config      string
		expectConf  Config
		expectMsg   string
		expectMatch func(Config, Config)
	}{
		{
			`search`,
			Config{
				Endpoint:     "/search",
				IncludePaths: defaultPath,
			},
			"Should support `search` without any arguments",
			func(expected, result Config) {
				So(expected.Endpoint, ShouldEqual, result.Endpoint)
			},
		},
		{
			`search /path`,
			Config{
				IncludePaths: convertToRegExp([]string{"/path"}),
			},
			"Should support `search` with only one argument",
			func(expected, result Config) {
				So(expected.IncludePaths[0].String(), ShouldEqual, result.IncludePaths[0].String())
			},
		},
		{
			`search / /path`,
			Config{
				Endpoint: "/path",
			},
			"Should support `search` with two arguments",
			func(expected, result Config) {
				So(expected.Endpoint, ShouldEqual, result.Endpoint)
			},
		},
		{
			`search / /search {
				endpoint /search2
			}`,
			Config{
				Endpoint: "/search2",
			},
			"Should support `search` arguments and override configurations",
			func(expected, result Config) {
				So(expected.Endpoint, ShouldEqual, result.Endpoint)
			},
		},
		{
			`search {
				+path /path
				+path /otherPath
				-path /forbidden
			}`,
			Config{
				IncludePaths: convertToRegExp([]string{"/path", "/otherPath"}),
				ExcludePaths: convertToRegExp([]string{"/forbidden"}),
			},
			"Should `search` support multiple include and excludes",
			func(expected, result Config) {
				So(expected.IncludePaths[0].String(), ShouldEqual, result.IncludePaths[0].String())
				So(expected.IncludePaths[1].String(), ShouldEqual, result.IncludePaths[1].String())
				So(expected.ExcludePaths[0].String(), ShouldEqual, result.ExcludePaths[0].String())
			},
		},
		{
			`search {
				expire 1000
			}`,
			Config{
				Expire: 1000 * time.Second,
			},
			"Should `search` support multiple include and excludes",
			func(expected, result Config) {
				So(expected.Expire, ShouldEqual, result.Expire)
			},
		},
	}
)

func TestSearchSetup(t *testing.T) {
	for _, kase := range configCases {
		Convey("Given a Caddy controller with the search middleware", t, func() {
			c := setup.NewTestController(kase.config)
			result, err := parseSearch(c)
			Convey("Should not receive an error when parsing", func() {
				So(err, ShouldBeNil)
			})
			Convey(kase.expectMsg, func() {
				kase.expectMatch(kase.expectConf, *result)
			})
		})
	}
}
