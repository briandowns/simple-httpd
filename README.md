# micro-httpd

micro-httpd is aimed to be a simple replacement for using `python -m SimpleHTTPServer` to serve local files.  Like [SimpleHTTPServer](https://docs.python.org/2/library/simplehttpserver.html), micro-httpd supports HTTP GET and HEAD requests and adheres to the [HTTP/1.1 RFC 2616](https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html) guidelines.  

The HTML output is a mix of the Python module layout and of an Apache directory listing layout. 

What makes micro-httpd different than it's Python alternative is that it supports HTTP2 with Let's Encrypt integration for automatic TLS.  A valid domain name is needed for the certificate to be generated and used.

## Installation

```
go install github.com/briandowns/micro-httpd
```
or
```
make install
```

## Usage

```
Usage of ./micro_httpd:
  -d string
    	domain name to use with TLS
  -p int
    	bind port (default 8000)
  -t	enable TLS (default :443)
```

### Examples

HTTP/1.1 on default port

`micro-httpd`

HTTP/1.1 on the given port

`micro-httpd -p 8181`

HTTP/2 on the default port

`micro-httpd -t -d some.valid.domain`

## Contributions 

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
