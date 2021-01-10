# simple-httpd

[![Build Status](https://travis-ci.org/briandowns/simple-httpd.svg?branch=master)](https://travis-ci.org/briandowns/simple-httpd)

simple-httpd is aimed to be a simple replacement for using `python -m SimpleHTTPServer` to serve local files.  Like [SimpleHTTPServer](https://docs.python.org/2/library/simplehttpserver.html), simple-httpd supports HTTP GET and HEAD requests and adheres to the [HTTP/1.1 RFC 2616](https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html) guidelines.

The HTML output is a mix of the Python module layout and of an Apache directory listing layout.

If you're looking for a full featured or even just more capable web server, take a look at [Caddy](https://caddyserver.com/).

## Features

* HTTP2 with [Let's Encrypt](https://letsencrypt.org/) integration for automatic TLS, if enabled.
* Automatic self signed certificate generation and use, if enabled.
* Multiple language support: English, Italian, Spanish, Irish. ISO 639-1 are given on the CLI.

Certificates are cached in `${HOME}/.autocert` for reuse.

## Installation

```
go get github.com/briandowns/simple-httpd
```
or
```
make install
```
or, on BSD
```
gmake install
```

### Examples

HTTP/1.1 on default port (8000)

```
simple-httpd
```

HTTP/1.1 on the given port

```
simple-httpd -p 8181
```

HTTP/2 with Let's Encrypt on the default port

```
simple-httpd -l some.valid.domain
```

The port assignment is for the HTTP server.  The TLS port will be 8081 and both will respond to requests.

```
simple-httpd -p 8080 -t some.valid.domain
```

Generate a self signed certificate and run the server

```
simple-httpd -g
```

Run server in Spanish

```
simple-httpd -l es

## Contributions

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
