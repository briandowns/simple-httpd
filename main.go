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
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/http2"
)

const version = "0.1"
const name = "simple-httpd"
const pathSeperator = "/"
var indexHTMLFiles = []string{
	"index.html",
	"index.htm",
}

const (
	cert    = "cert.pem"
	key     = "key.pem"
	certDir = "/.autocert"
)

// gitSHA is populated at build time from
// `-ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"`
var gitSHA string

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
	Port      int
	template  *template.Template
}

// requestData holds data about the request for logging
type requestData struct {
	Timestamp   string `json:"timestamp,omitempty"`
	Method      string `json:"method,omitempty"`
	HTTPVersion string `json:"http_version,omitempty"`
	RemoteAddr  string `json:"remote_addr,omitempty"`
	Path        string `json:"path,omitempty"`
	Status      int    `json:"status,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	Error       string `json:"error,omitempty,omitempty"`
}

func (r requestData) Format(f fmt.State, c rune) {
	switch c {
	case 'v', 's':
		enc := json.NewEncoder(f)
		enc.Encode(r)
	}
}

// setHeaders sets the base headers for all requests
func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", name+pathSeperator+version)
	w.Header().Add("Date", time.Now().Format(time.RFC822))
}

func isIndexFile(file string) bool {
	for _, s := range indexHTMLFiles {
		if s == file {
			return true
		}
	}
	return false
}

// ServeHTTP handles inbound requests
func (h *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintln(err), http.StatusInternalServerError)
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

	parsedURL, err := url.Parse(req.RequestURI)
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusInternalServerError
		fmt.Println(rd)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	escapedPath := parsedURL.EscapedPath()
	fullpath := filepath.Join(h.Directory, escapedPath[1:])

	file, err := os.Open(fullpath)
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusNotFound
		fmt.Println(rd)
		http.NotFound(w, req)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusInternalServerError
		fmt.Println(rd)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setHeaders(w)

	if stat.IsDir() {
		contents, err := file.Readdir(-1)
		if err != nil {
			rd.Status = http.StatusInternalServerError
			rd.Error = err.Error()
			fmt.Println(rd)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		files := make([]Data, 0, len(contents))
		for _, entry := range contents {
			if isIndexFile(entry.Name()) {
				w.Header().Set("Content-type", "text/html; charset=UTF-8")
				w.Header().Set("Content-Length", fmt.Sprintf("%v", entry.Size()))

				hf, err := os.Open(fullpath + pathSeperator + entry.Name())
				if err != nil {
					fmt.Println(err)
					return
				}
				io.Copy(w, hf)

				rd.Status = http.StatusOK
				fmt.Println(rd)
				return
			}
			file := Data{
				Name:         entry.Name(),
				LastModified: entry.ModTime().Format(time.RFC1123),
				URI:          path.Join(escapedPath, entry.Name()),
			}
			if entry.IsDir() {
				file.Name = entry.Name() + pathSeperator
				file.Size = entry.Size()
			}
			files = append(files, file)
		}

		rd.Status = http.StatusOK

		w.Header().Set("Content-type", "text/html; charset=UTF-8")

		h.template.Execute(w, map[string]interface{}{
			"files":           files,
			"version":         gitSHA,
			"port":            h.Port,
			"relativePath":    escapedPath,
			"goVersion":       runtime.Version(),
			"parentDirectory": path.Dir(escapedPath),
		})

		fmt.Println(rd)

		return
	}

	if mimetype := mime.TypeByExtension(path.Ext(file.Name())); mimetype != "" {
		fmt.Println(mimetype)
		w.Header().Set("Content-type", mimetype)
	} else {
		w.Header().Set("Content-type", "application/octet-stream")
	}

	io.Copy(w, file)

	rd.Status = http.StatusOK
	fmt.Println(rd)
}

// serveTLS
func serveTLS(domain string, port int) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	cacheDir := u.HomeDir + certDir
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

	ln, err := net.ListenTCP("tcp", &net.TCPAddr{
		Port: port,
	})
	if err != nil {
		return err
	}

	return srv.Serve(tls.NewListener(keepAliveListener{ln}, srv.TLSConfig))
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

func getpwd() string {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	return pwd
}

func homeDir() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	return u.HomeDir
}

func main() {
	var port int
	var le string
	var gs bool

	pwd := getpwd()

	flag.IntVar(&port, "p", 8000, "bind port")
	flag.StringVar(&le, "l", "", "enable TLS with Let's Encrypt for the given domain name. Port = port + 1 ")
	flag.BoolVar(&gs, "g", false, "generate and use a self signed certificate")
	flag.Parse()

	if le != "" {
		var srv http.Server
		http2.ConfigureServer(&srv, new(http2.Server))

		tlsPort := port + 1

		go func() {
			fmt.Printf("Serving HTTPS on 0.0.0.0 port %v ...\n", tlsPort)
			log.Fatal(serveTLS(le, tlsPort))
		}()
	}

	h := &httpServer{
		Port:      port,
		Directory: pwd,
		template:  template.Must(template.New("listing").Parse(htmlTemplate)),
	}

	if gs {
		hd := homeDir()

		certPath := hd + certDir + pathSeperator + cert
		keyPath := hd + certDir + pathSeperator + key

		if err := generateCertificates(certPath, keyPath); err != nil {
			log.Fatalln(err)
		}

		fmt.Printf("Serving HTTPS on 0.0.0.0 port %v ...\n", h.Port+1)
		log.Fatal(http.ListenAndServeTLS(fmt.Sprintf("0.0.0.0:%d", port+1), certPath, keyPath, h))
	}

	fmt.Printf("Serving HTTP on 0.0.0.0 port %v ...\n", h.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), h))

}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>simple-httpd</title>
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
    <p>simple-httpd - {{.version}} / {{.goVersion}}</p>
  </footer>
</html>`
