package search

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"text/template"
	"time"

	"github.com/mholt/caddy/caddy/setup"
	"github.com/mholt/caddy/middleware"
	"github.com/pedronasser/caddy-search/indexer"
	"github.com/pedronasser/caddy-search/indexer/bleve"
)

// Setup creates a new middleware with the given configuration
func Setup(c *setup.Controller) (mid middleware.Middleware, err error) {
	var config *Config

	config, err = parseSearch(c)
	if err != nil {
		return nil, err
	}

	if c.ServerBlockHostIndex == 0 {
		index, err := NewIndexer(config.Engine, indexer.Config{
			HostName:       config.HostName,
			IndexDirectory: config.IndexDirectory,
		})

		if err != nil {
			return nil, err
		}

		ppl, err := NewPipeline(config, index)

		if err != nil {
			return nil, err
		}

		expire := time.NewTicker(config.Expire)
		go func() {
			var lastScanned indexer.Record
			lastScanned = ScanToPipe(c.Root, ppl, index)

			for {
				select {
				case <-expire.C:
					if lastScanned != nil && (!lastScanned.Indexed().IsZero() || lastScanned.Ignored()) {
						lastScanned = ScanToPipe(c.Root, ppl, index)
					}
				}
			}
		}()

		c.ServerBlockStorage = &Search{
			Config:   config,
			Indexer:  index,
			Pipeline: ppl,
		}
	}

	if s, ok := c.ServerBlockStorage.(*Search); ok {
		mid = func(next middleware.Handler) middleware.Handler {
			s.Next = next
			return s
		}
	} else {
		return nil, errors.New("Could not create the search middleware")
	}

	return
}

// ScanToPipe ...
func ScanToPipe(fp string, pipeline *Pipeline, index indexer.Handler) indexer.Record {
	var last indexer.Record
	filepath.Walk(fp, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "." {
			return nil
		}

		if info.Name() == "" || info.Name()[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			reqPath, err := filepath.Rel(fp, path)
			if err != nil {
				return err
			}
			reqPath = "/" + reqPath

			if pipeline.ValidatePath(reqPath) {
				go func(url string) {
					http.Get("http://" + pipeline.config.HostName + url)
				}(reqPath)
			}
		}

		return nil
	})

	return last
}

// NewIndexer creates a new Indexer with the received config
func NewIndexer(engine string, config indexer.Config) (index indexer.Handler, err error) {
	switch engine {
	case "bleve":
		index, err = bleve.New(config)
		break
	default:
		index, err = bleve.New(config)
		break
	}
	return
}

// Config represents this middleware configuration structure
type Config struct {
	HostName       string
	Engine         string
	Path           string
	IncludePaths   []*regexp.Regexp
	ExcludePaths   []*regexp.Regexp
	Endpoint       string
	IndexDirectory string
	Template       *template.Template
	Expire         time.Duration
	SiteRoot       string
	Crawl          bool
}

// parseSearch controller information to create a IndexSearch config
func parseSearch(c *setup.Controller) (*Config, error) {
	conf := &Config{
		HostName:       c.ServerBlockHosts[0],
		Engine:         `bleve`,
		IndexDirectory: `/tmp/caddyIndex`,
		IncludePaths:   []*regexp.Regexp{},
		ExcludePaths:   []*regexp.Regexp{},
		Endpoint:       `/search`,
		SiteRoot:       c.Root,
		Expire:         60 * time.Second,
		Template:       nil,
		Crawl:          false,
	}

	if conf.HostName != "" && string(conf.HostName[len(conf.HostName)-1]) == ":" {
		conf.HostName = conf.HostName + "2015"
	}

	incPaths := []string{}
	excPaths := []string{}

	for c.Next() {
		args := c.RemainingArgs()

		switch len(args) {
		case 2:
			conf.Endpoint = args[1]
			fallthrough
		case 1:
			incPaths = append(incPaths, args[0])
		}

		for c.NextBlock() {
			switch c.Val() {
			case "engine":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				conf.Engine = c.Val()
			case "+path":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				incPaths = append(incPaths, c.Val())
				incPaths = append(incPaths, c.RemainingArgs()...)
			case "-path":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				excPaths = append(excPaths, c.Val())
				excPaths = append(excPaths, c.RemainingArgs()...)
			case "endpoint":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				conf.Endpoint = c.Val()
			case "expire":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				exp, err := strconv.Atoi(c.Val())
				if err != nil {
					return nil, err
				}
				conf.Expire = time.Duration(exp) * time.Second
			case "datadir":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				conf.IndexDirectory = c.Val()
			case "template":
				var err error
				if c.NextArg() {
					conf.Template, err = template.ParseFiles(filepath.Join(conf.SiteRoot, c.Val()))
					if err != nil {
						return nil, err
					}
				}
			case "crawl":
				conf.Crawl = true
			}
		}
	}

	if len(incPaths) == 0 {
		incPaths = append(incPaths, "^/")
	}

	conf.IncludePaths = convertToRegExp(incPaths)
	conf.ExcludePaths = convertToRegExp(excPaths)

	dir := conf.IndexDirectory
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, c.Err("Given 'datadir' not a valid path.")
		}
	}

	if conf.Template == nil {
		var err error
		conf.Template, err = template.New("search-results").Parse(defaultTemplate)
		if err != nil {
			return nil, err
		}
	}

	return conf, nil
}

func convertToRegExp(rexp []string) (r []*regexp.Regexp) {
	r = make([]*regexp.Regexp, 0)
	for _, exp := range rexp {
		var rule *regexp.Regexp
		var err error
		rule, err = regexp.Compile(exp)
		if err != nil {
			continue
		}
		r = append(r, rule)
	}
	return
}

// The default template to use when serving up HTML search results
const defaultTemplate = `<!DOCTYPE html>
<html>
	<head>
		<title>Search results for: {{.Query}}</title>
		<meta charset="utf-8">
<style>
body {
	padding: 1% 2%;
	font: 16px Arial;
}

form {
	margin-bottom: 3em;
}

input {
	font-size: 14px;
	border: 1px solid #CCC;
	background: #FFF;
	line-height: 1.5em;
	padding: 5px;
}

input[name=q] {
	width: 100%;
	max-width: 350px;
}

input[type=submit] {
	border-radius: 5px;
	padding: 5px 10px;
}

p,
li {
	max-width: 600px;
}

.result-title {
	font-size: 18px;
}

.result-url {
	font-size: 14px;
	margin-bottom: 5px;
	color: #777;
}

li {
	margin-top: 15px;
}
</style>
	</head>
	<body>
		<h1>Site Search</h1>

		<form method="GET" action="{{.URL.Path}}">
			<input type="text" name="q" value="{{.Query}}"> <input type="submit" value="Search">
		</form>

		{{if .Query}}
		<p>
			Found <b>{{len .Results}}</b> result{{if ne (len .Results) 1}}s{{end}} for <b>{{.Query}}</b>
		</p>

		<ol>
			{{range .Results}}
			<li>
				<div class="result-title"><a href="{{.Path}}">{{.Title}}</a></div>
				<div class="result-url">{{$.Req.Host}}{{.Path}}</div>
				{{.Body}}
			</li>
			{{end}}
		</ol>
		{{end}}
	</body>
</html>`
