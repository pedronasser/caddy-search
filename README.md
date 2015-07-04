# search

Middleware for [Caddy](https://caddyserver.com).

**search** indexes your static and/or dynamic documents then serves a HTTP search endpoint.

### Syntax

```
search [directory|regexp] [endpoint "search/"]
```
* **directory** is the path, relative to site root, to a directory (static content)
* **regexp** is the URL [regular expression] of documents that must be indexed (static and dynamic content)
* **endpoint** is the path, relative to site's root url, of the search endpoint

For more options, use the following syntax:

```
search {
    engine      (default: bleve)
    datadir     (default: index)
    endpoint    (default: /search)

    +path       directory or regexp
    -path       directory or regexp
    ignore      regexp
}
```
* **engine** is the engine for indexing and searching
* **datadir** is the directory's path where the indexer should store all data
* **+path** include a path to be indexed (can be added multiple times)
* **-path** exclude a path from being index (can be added multiple times)
* **ignore** ignore files that matches the regular expression (can be added multiple times)

Each property in the block is optional.

### Supported Engines

* [BleveSearch](http://github.com/blevesearch/bleve)

### Examples

Index every static content in root folder (single line configuration)
```
search /
```

Index every content (single line configuration)
```
search ^/
```

Indexing every dynamic content with a different endpoint (single line configuration)
```
search /(.*) /mySearch
```

Multiple static and dynamic paths
```
search ^/blog/ {
	+path /static/docs/
    -path ^/blog/admin/
    ignore robots.txt
}
```

Different directory for storing the index
```
search {
	datadir /tmp/index
}
```
