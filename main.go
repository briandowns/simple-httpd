package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
)

const version = "0.1"
const name = "micro-httpd"
const indexHTMLFile = "index.html"
const pathSeperator = "/"

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
	Timestamp  string `json:"timestamp"`
	Method     string `json:"method"`
	RemoteAddr string `json:"remote_addr"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	UserAgent  string `json:"user_agent"`
	Error      string `json:"error,omitempty"`
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

// ServeHTTP handles inbound requests
func (h *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintf("%v", err), http.StatusInternalServerError)
			log.Printf("recovering from error: %s\n", err)
		}
	}()

	rd := requestData{
		Timestamp:  time.Now().Format("2006-01-02 15:04:05"),
		RemoteAddr: req.RemoteAddr,
		Method:     req.Method,
		Path:       req.RequestURI,
		UserAgent:  req.UserAgent(),
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
			if entry.Name() == indexHTMLFile {
				w.Header().Set("Content-type", "text/html; charset=UTF-8")
				w.Header().Set("Content-Length", fmt.Sprintf("%v", entry.Size()))

				hf, err := os.Open(fullpath + pathSeperator + entry.Name())
				if err != nil {
					fmt.Println(err)
					return
				}
				io.Copy(w, hf)

				rd.Status = http.StatusOK
				fmt.Println(rd.String())
				return
			}
			file := Data{
				Name:         entry.Name(),
				LastModified: entry.ModTime().Format(time.RFC1123),
				URI:          path.Join(queryStr, entry.Name()),
			}
			if entry.IsDir() {
				file.Name = entry.Name() + pathSeperator
				file.Size = entry.Size()
			}
			files = append(files, file)
		}

		rd.Status = http.StatusOK

		w.Header().Set("Server", name+pathSeperator+version)
		w.Header().Add("Date", time.Now().Format(time.RFC822))
		w.Header().Set("Content-type", "text/html; charset=UTF-8")

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

	w.Header().Set("Server", name+pathSeperator+version)
	w.Header().Add("Date", time.Now().Format(time.RFC822))

	if mimetype := mime.TypeByExtension(path.Ext(file.Name())); mimetype != "" {
		fmt.Println(mimetype)
		w.Header().Set("Content-type", "text/html; charset=UTF-8")
	} else {
		w.Header().Set("Content-type", "application/octet-stream")
	}

	io.Copy(w, file)

	rd.Status = http.StatusOK
	fmt.Println(rd.String())

	return
}

// serveTLS
func serveTLS(domain string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	cacheDir := u.HomeDir + "/.autocert"
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return err
	}

	m := autocert.Manager{
		Cache:      autocert.DirCache(cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(domain),
	}

	srv := &http.Server{
		TLSConfig: &tls.Config{
			GetCertificate: m.GetCertificate,
		},
	}

	http2.ConfigureServer(srv, &http2.Server{
		NewWriteScheduler: func() http2.WriteScheduler {
			return http2.NewPriorityWriteScheduler(nil)
		},
	})

	ln, err := net.Listen("tcp", ":443")
	if err != nil {
		return err
	}

	return srv.Serve(tls.NewListener(keepAliveListener{ln.(*net.TCPListener)}, srv.TLSConfig))
}

// keepAliveListener
type keepAliveListener struct {
	*net.TCPListener
}

// Accept
func (k keepAliveListener) Accept() (net.Conn, error) {
	tc, err := k.AcceptTCP()
	if err != nil {
		return nil, err
	}

	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(time.Minute * 3)

	return tc, nil
}

func main() {
	var port int
	var tls bool
	var domain string

	flag.IntVar(&port, "p", 8080, "bind port")
	flag.BoolVar(&tls, "t", false, "enable TLS (default :443)")
	flag.StringVar(&domain, "d", "", "domain name to use with TLS")
	flag.Parse()

	errChan := make(chan error, 2)

	var srv http.Server

	if tls {
		if domain == "" {
			flag.Usage()
			os.Exit(1)
		}

		http2.ConfigureServer(&srv, &http2.Server{})

		go func() {
			fmt.Printf("Serving HTTPS on 0.0.0.0 port 443 ...\n")
			errChan <- serveTLS(domain)
		}()
	}

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h := &httpServer{
		Port:      strconv.Itoa(port),
		Directory: pwd,
		template:  template.Must(template.New("listing").Parse(htmlTemplate)),
	}

	go func() {
		http.Handle("/", h)
		fmt.Printf("Serving HTTP on 0.0.0.0 port %s ...\n", h.Port)
		errChan <- http.ListenAndServe(":"+h.Port, nil)
	}()

	log.Fatalln(<-errChan)

}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>micro-httpd</title>
	<style>
		table, td {
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
