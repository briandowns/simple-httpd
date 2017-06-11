# micro-httpd

micro-httpd is aimed to be a simple replacement for using `python -M SimpleHTTPServer`.  Like [SimpleHTTPServer](https://docs.python.org/2/library/simplehttpserver.html), micro-httpd supports HTTP GET and HEAD requests.  HTTP HEAD requests adhere to [HTTP/1.1 RFC 2616](https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html) guidelines.  

The HTML output is a mix of the Python module layout and of an Apache directory listing layout. 

## Usage

```
Usage of ./micro_httpd:
  -p int
    	bind port (default 8080)
```

## Contributions 

* File Issue with details of the problem, feature request, etc.
* Submit a pull request and include details of what problem or feature the code is solving or implementing.
