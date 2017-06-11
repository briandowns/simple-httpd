package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

const version = "0.1"

const (
	httpContentType        = "Content-type"
	textHTMLContentType    = "text/html"
	octetStreamContentType = "application/octet-stream"
	charsetUTF8            = "charset=UTF-8"
)

// Data holds the data passed to the template engine
type Data struct {
	Name         string
	LastModified string
	URI          string
	Size         int64
}

// httpServer holds the relavent info/state
type httpServer struct {
	Directory string
	Port      string
	template  *template.Template
}

// start starts the server
func (h *httpServer) start() {
	http.Handle("/", h)
	log.Printf("Listening on port %s\n", h.Port)
	http.ListenAndServe(":"+h.Port, nil)
}

// ServeHTTP handles inbound requests
func (h *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			log.Printf("recovering from error: %s\n", err)
		}
	}()

	if req.RequestURI == "/favicon.ico" {
		return
	}

	queryStr, err := url.QueryUnescape(req.RequestURI)
	if err != nil {
		log.Println(err)
		return
	}

	fullpath := filepath.Join(h.Directory, queryStr[1:])

	file, err := os.Open(fullpath)
	if err != nil {
		log.Println(err)
		http.NotFound(w, req)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		w.Header().Set(httpContentType, textHTMLContentType+"; "+charsetUTF8)

		contents, err := file.Readdir(-1)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		files := make([]Data, 0, len(contents))
		for _, entry := range contents {
			file := Data{
				Name:         entry.Name(),
				LastModified: entry.ModTime().Format(time.RFC1123),
				URI:          path.Join(queryStr, entry.Name()),
			}
			if entry.IsDir() {
				file.Size = entry.Size()
			}
			files = append(files, file)
		}

		h.template.Execute(w, map[string]interface{}{
			"files":           files,
			"version":         version,
			"port":            h.Port,
			"relativePath":    queryStr,
			"parentDirectory": path.Dir(queryStr),
		})

		return
	}

	if mimetype := mime.TypeByExtension(path.Ext(file.Name())); mimetype != "" {
		w.Header().Set(httpContentType, mimetype)
	} else {
		w.Header().Set(httpContentType, octetStreamContentType)
	}

	statinfo, err := file.Stat()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%v", statinfo.Size()))
	io.Copy(w, file)

	return
}

func main() {
	var port int

	flag.IntVar(&port, "p", 8080, "bind port")
	flag.Parse()

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h := httpServer{
		Port:      strconv.Itoa(port),
		Directory: pwd,
		template:  template.Must(template.New("index").Parse(tmpl)),
	}

	h.start()
}

const tmpl = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>micro-httpd</title>
  </head>
  <body>
    <h1>Index of {{.relativePath}}</h1>
    <table style="text-align: left;">
	  <tr>
        <th>Name</th>
		<th>Last Modified</th>
		<th>Size</th>
	  </tr>
	  <tr>
	  	<td><hr></td>
	    <td><hr></td>
		<td><hr></td>
	  </tr>
	  <tr>
	    <td><a href="{{.parentDirectory}}">{{.parentDirectory}}</td>
		<td></td>
		<td></td>
	  </td>
      {{range .files}}
      <tr>
	    <td><a href="{{.URI}}">{{.Name}}</td>
		<td>{{.LastModified}}</td>
		<td>{{.Size}}</td>
	  </tr>
      {{end}}
    <table>
  </body>
  <hr>
  <footer>
    <p>micro-httpd / {{.version}} on port {{.port}}</p>
  </footer>
</html>`
