package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const version = "0.1"
const name = "micro-httpd"
const indexHTMLFile = "index.html"

type resourceType byte

const (
	directoryResource resourceType = iota
	fileResource
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

// requestData
type requestData struct {
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	UserAgent string `json:"user_agent"`
	Error     string `json:"error,omitempty"`
}

// String stringifies the the requestData struct
func (r requestData) String() string {
	b, err := json.Marshal(r)
	if err != nil {
		return ""
	}

	return string(b)
}

// start starts the server
func (h *httpServer) start() {
	http.Handle("/", h)
	fmt.Printf("Serving HTTP on 0.0.0.0 port %s ...\n", h.Port)
	http.ListenAndServe(":"+h.Port, nil)
}

// setHeaders sets the response headers
func setHeaders(rt resourceType, file *os.File, statInfo os.FileInfo, w http.ResponseWriter) {
	w.Header().Set("Server", name+"/"+version)
	w.Header().Add("Date", time.Now().Format(time.RFC822))

	switch rt {
	case directoryResource:
		w.Header().Set("Content-type", "text/html; charset=UTF-8")
	case fileResource:
		if path.Ext(file.Name()) == "html" || path.Ext(file.Name()) == "htm" {
			fmt.Println(path.Ext(file.Name()))
			w.Header().Set("Content-type", "text/html")
		} else {
			w.Header().Set("Content-type", "application/octet-stream")
		}
		// if mimetype := mime.TypeByExtension(path.Ext(file.Name())); mimetype != "" {
		// 	w.Header().Set("Content-type", "text/html")
		// } else {
		// 	w.Header().Set("Content-type", "application/octet-stream")
		// }
		w.Header().Set("Content-Length", fmt.Sprintf("%v", statInfo.Size()))
	}
}

// ServeHTTP handles inbound requests
func (h *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			log.Printf("recovering from error: %s\n", err)
		}
	}()

	rd := requestData{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Method:    req.Method,
		Path:      req.RequestURI,
		UserAgent: req.UserAgent(),
	}

	queryStr, err := url.QueryUnescape(req.RequestURI)
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusInternalServerError
		fmt.Println(rd.String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullpath := filepath.Join(h.Directory, queryStr[1:])

	file, err := os.Open(fullpath)
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusNotFound
		fmt.Println(rd.String())
		http.NotFound(w, req)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusInternalServerError
		fmt.Println(rd.String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if stat.IsDir() {
		setHeaders(directoryResource, nil, stat, w)

		contents, err := file.Readdir(-1)
		if err != nil {
			rd.Status = http.StatusInternalServerError
			rd.Error = err.Error()
			fmt.Println(rd.String())
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
				file.Name = entry.Name() + "/"
				file.Size = entry.Size()
			}
			files = append(files, file)
		}

		rd.Status = http.StatusOK

		h.template.Execute(w, map[string]interface{}{
			"files":           files,
			"version":         version,
			"port":            h.Port,
			"relativePath":    queryStr,
			"goVersion":       runtime.Version(),
			"parentDirectory": path.Dir(queryStr),
		})

		fmt.Println(rd.String())
		return
	}

	statInfo, err := file.Stat()
	if err != nil {
		rd.Status = http.StatusInternalServerError
		rd.Error = err.Error()
		fmt.Println(rd.String())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setHeaders(fileResource, file, statInfo, w)

	io.Copy(w, file)

	rd.Status = http.StatusOK
	fmt.Println(rd.String())

	return
}

func main() {
	var port int
	var http2 bool

	flag.IntVar(&port, "p", 8080, "bind port")
	flag.BoolVar(&http2, "2", false, "enable HTTP/2")
	flag.Parse()

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h := httpServer{
		Port:      strconv.Itoa(port),
		Directory: pwd,
		template:  template.Must(template.New("listing").Parse(htmlTemplate)),
	}

	h.start()
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>micro-httpd</title>
	<style>
		table, th, td {
    	border: 1px;
	  }
	</style>
  </head>
  <body>
    <h2>Directory listing for {{.relativePath}}</h2>
	<hr>
    <table>
	  <tr>
        <td><b>Name</b></td>
		<td><b>Last Modified</b></td>
		<td><b>Size</b></td>
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
    <p>micro-httpd {{.version}} / {{.goVersion}}</p>
  </footer>
</html>`
