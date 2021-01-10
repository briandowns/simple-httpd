package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

var (
	name    string
	version string
	// gitSHA is populated at build time from
	// `-ldflags "-X main.gitSHA=$(shell git rev-parse HEAD)"`
	gitSHA string
)

var indexHTMLFiles = []string{
	"index.html",
	"index.htm",
}

const (
	cert    = "cert.pem"
	key     = "key.pem"
	certDir = ".autocert"

	pathSeperator = "/"

	defaultPort = 8000
)

var directoryText string

// Data holds the data passed to the template engine.
type Data struct {
	Name         string
	LastModified string
	URI          string
	Size         int64
}

// httpServer holds the relavent info/state.
type httpServer struct {
	Directory string
	Port      int
	TLSPort   int
	HTTPS     bool
	template  *template.Template
}

// requestData holds data about the request for logging.
type requestData struct {
	Timestamp  string `json:"timestamp,omitempty"`
	Method     string `json:"method,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	Path       string `json:"path,omitempty"`
	Status     int    `json:"status,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	Error      string `json:"error,omitempty"`
}

// setHeaders sets the base headers for all requests.
func setHeaders(w http.ResponseWriter) {
	w.Header().Set("Server", name+pathSeperator+version)
	w.Header().Add("Date", time.Now().Format(time.RFC822))
}

// isIndexFile determines if the given file is one
// of the accepted index files.
func isIndexFile(file string) bool {
	for _, s := range indexHTMLFiles {
		if s == file {
			return true
		}
	}
	return false
}

// ServeHTTP handles inbound requests.
func (h *httpServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.HTTPS && req.TLS == nil {
		url := "https://" + strings.Split(req.Host, ":")[0]
		if h.TLSPort != 443 {
			url = url + ":" + strconv.FormatInt(int64(h.TLSPort), 10)
		}
		url += req.URL.String()
		http.Redirect(w, req, url, http.StatusFound)
		return
	}
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, fmt.Sprintln(err), http.StatusInternalServerError)
			log.Error("msg", zap.Error(fmt.Errorf("recovering from error: %s", err)))
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
		log.Error("msg",
			zap.String("method", rd.Method),
			zap.String("remote_addr", rd.RemoteAddr),
			zap.String("path", rd.Path),
			zap.String("user_agent", rd.UserAgent),
			zap.Int("status", rd.Status),
			zap.Error(errors.New(rd.Error)))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check if we have a file with spaces in the name and
	// replace the %20 with an actual space.
	escapedPath := parsedURL.EscapedPath()
	if strings.Contains(escapedPath, "%20") {
		escapedPath = strings.ReplaceAll(escapedPath, "%20", " ")
	}

	fullpath := filepath.Join(h.Directory, escapedPath[1:])

	file, err := os.Open(fullpath)
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusNotFound
		log.Error("msg",
			zap.String("method", rd.Method),
			zap.String("remote_addr", rd.RemoteAddr),
			zap.String("path", rd.Path),
			zap.String("user_agent", rd.UserAgent),
			zap.Int("status", rd.Status),
			zap.Error(errors.New(rd.Error)))
		http.NotFound(w, req)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		rd.Error = err.Error()
		rd.Status = http.StatusInternalServerError
		log.Error("msg",
			zap.String("method", rd.Method),
			zap.String("remote_addr", rd.RemoteAddr),
			zap.String("path", rd.Path),
			zap.String("user_agent", rd.UserAgent),
			zap.Int("status", rd.Status),
			zap.Error(errors.New(rd.Error)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	setHeaders(w)

	if stat.IsDir() {
		if escapedPath[len(escapedPath)-1] != '/' {
			// Redirect all directory requests to ensure they end with a slash
			http.Redirect(w, req, escapedPath+"/", http.StatusFound)
			rd.Status = http.StatusFound
			log.Info("msg",
				zap.String("method", rd.Method),
				zap.String("remote_addr", rd.RemoteAddr),
				zap.String("path", rd.Path),
				zap.String("user_agent", rd.UserAgent),
				zap.Int("status", rd.Status))
			return
		}

		contents, err := file.Readdir(-1)
		if err != nil {
			rd.Status = http.StatusInternalServerError
			rd.Error = err.Error()
			log.Error("msg",
				zap.String("method", rd.Method),
				zap.String("remote_addr", rd.RemoteAddr),
				zap.String("path", rd.Path),
				zap.String("user_agent", rd.UserAgent),
				zap.Int("status", rd.Status),
				zap.Error(errors.New(rd.Error)))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		files := make([]Data, 0, len(contents))
		for _, entry := range contents {
			if isIndexFile(entry.Name()) {
				w.Header().Set("Content-type", "text/html; charset=UTF-8")
				w.Header().Set("Content-Length", fmt.Sprintf("%v", entry.Size()))

				hf, err := os.Open(filepath.Join(fullpath, entry.Name()))
				if err != nil {
					log.Error("msg", zap.Error(err))
					return
				}
				if _, err := io.Copy(w, hf); err != nil {
					log.Error("msg", zap.Error(err))
					return
				}

				rd.Status = http.StatusOK
				log.Info("request",
					zap.String("method", rd.Method),
					zap.String("remote_addr", rd.RemoteAddr),
					zap.String("path", rd.Path),
					zap.String("user_agent", rd.UserAgent),
					zap.Int("status", rd.Status),
					zap.Error(errors.New(rd.Error)))
				return
			}
			file := Data{
				Name:         entry.Name(),
				LastModified: entry.ModTime().Format(time.RFC1123),
				URI:          path.Join(escapedPath, entry.Name()),
				Size:         entry.Size(),
			}
			if entry.IsDir() {
				file.Name = entry.Name() + pathSeperator
			}
			files = append(files, file)
		}

		rd.Status = http.StatusOK

		w.Header().Set("Content-type", "text/html; charset=UTF-8")

		if err := h.template.Execute(w, map[string]interface{}{
			"files":           files,
			"version":         gitSHA,
			"port":            h.Port,
			"relativePath":    escapedPath,
			"goVersion":       runtime.Version(),
			"lang":            lang,
			"directoryText":   directoryText,
			"parentDirectory": path.Clean(escapedPath + "/.."),
		}); err != nil {
			log.Error("msg", zap.Error(err))
			return
		}

		log.Info("request",
			zap.String("method", rd.Method),
			zap.String("remote_addr", rd.RemoteAddr),
			zap.String("path", rd.Path),
			zap.String("user_agent", rd.UserAgent),
			zap.Int("status", rd.Status))
		return
	}

	if mimetype := mime.TypeByExtension(path.Ext(file.Name())); mimetype != "" {
		w.Header().Set("Content-type", mimetype)
	} else {
		w.Header().Set("Content-type", "application/octet-stream")
	}

	if _, err := io.Copy(w, file); err != nil {
		log.Error("msg", zap.Error(err))

		return
	}

	rd.Status = http.StatusOK
	log.Info("request",
		zap.String("method", rd.Method),
		zap.String("remote_addr", rd.RemoteAddr),
		zap.String("path", rd.Path),
		zap.String("user_agent", rd.UserAgent),
		zap.Int("status", rd.Status))
}

const usage = `version: %s

Usage: %[2]s [-p port] [-e domain]

Options:
    -h            help
    -v            show version and exit
    -g            enable TLS/HTTPS generate and use a self signed certificate
    -p port       bind HTTP port (default: 8000)
    -e domain     enable TLS/HTTPS with Let's Encrypt for the given domain name
    -c path       enable TLS/HTTPS with a predefined HTTPS certificate
    -t port       bind HTTPS port (default: 443, 4433 for -g)
    -l lang       language code (ISO 639-1) to for UI. (default: en)

Examples: 
    %[2]s                        start server. http://localhost:8000
    %[2]s -p 80                  use HTTP port 80. http://localhost
    %[2]s -g                     enable HTTPS generated certificate. https://localhost:4433
    %[2]s -p 80 -e example.com   enable HTTPS with Let's Encrypt. https://example.com
`

const warmUpDelay = time.Millisecond * 10

var (
	port    int
	le      string
	gs      bool
	tlsPort int
	tlsCert string
	vers    bool
	lang    string
)

var log *zap.Logger

// setDirectoryText sets the text for the directory listing.
func setDirectoryText(alng string) error {
	switch lang {
	case "it":
		directoryText = "L'elenco di directory per"
	case "es":
		directoryText = "Listado de directorio para"
	case "ga":
		directoryText = "Comhadlann liostú do"
	case "en", "":
		directoryText = "Directory listing for"
	default:
		return fmt.Errorf("error: invalid language: %s", lang)
	}
	return nil
}

func main() {
	var err error
	log, err = zap.NewProduction()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer log.Sync()

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	flag.Usage = func() {
		w := os.Stderr
		for _, arg := range os.Args {
			if arg == "-h" {
				w = os.Stdout
				break
			}
		}
		fmt.Fprintf(w, usage, version, name)
	}

	flag.BoolVar(&vers, "v", false, "")
	flag.IntVar(&port, "p", defaultPort, "")
	flag.StringVar(&le, "e", "", "")
	flag.StringVar(&tlsCert, "c", "", "")
	flag.BoolVar(&gs, "g", false, "")
	flag.IntVar(&tlsPort, "t", -1, "")
	flag.StringVar(&lang, "l", "en", "")
	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s\n", version)
		return
	}

	if tlsPort == -1 {
		if gs {
			tlsPort = 4433
		} else {
			tlsPort = 443
		}
	}

	if err := setDirectoryText(lang); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	h := &httpServer{
		Port:      port,
		TLSPort:   tlsPort,
		Directory: pwd,
		template:  template.Must(template.New("listing").Parse(htmlTemplate)),
	}

	if le != "" || tlsCert != "" || gs {
		h.HTTPS = true

		var tlsServer *http.Server
		var certPath string
		var keyPath string

		u, err := user.Current()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		switch {
		case tlsCert != "":
			if gs {
				fmt.Println(errors.New("cannot specify both -tls-cert and -g"))
				os.Exit(1)
			}
			certPath, keyPath = tlsCert, tlsCert // assume a single PEM format
		case gs:
			hd := u.HomeDir
			certPath = filepath.Join(hd, certDir, cert)
			keyPath = filepath.Join(hd, certDir, key)
			if err := generateCertificates(certPath, keyPath); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		default:
			if tlsPort != 443 {
				fmt.Printf("invalid -tls-port %d. It must be 443 when LetsEncrypt is specified\n", tlsPort)
				os.Exit(1)
			}

			cacheDir := filepath.Join(u.HomeDir, certDir)

			if err := os.MkdirAll(cacheDir, 0700); err != nil {
				fmt.Printf("could not create cache directory: %s\n", cacheDir)
				os.Exit(1)
			}

			certManager := autocert.Manager{
				Cache:      autocert.DirCache(cacheDir),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(le),
			}

			tlsServer = &http.Server{
				Addr: fmt.Sprintf(":%d", tlsPort),
				TLSConfig: &tls.Config{
					GetCertificate: certManager.GetCertificate,
				},
				Handler: h,
			}
		}

		go func() {
			var err error
			if tlsServer == nil {
				err = http.ListenAndServeTLS(fmt.Sprintf("0.0.0.0:%d", tlsPort), certPath, keyPath, h)
			} else {
				err = tlsServer.ListenAndServeTLS("", "")
			}
			if err != nil {
				log.Fatal("msg", zap.Error(err))
			}
		}()

		time.Sleep(warmUpDelay) // give a little warmup time to the TLS
		log.Info("msg", zap.String("Serving HTTP on", "0.0.0.0"), zap.Int("port", h.Port), zap.Int("https", tlsPort))
	} else {
		go func() {
			time.Sleep(warmUpDelay) // give a little warmup time to the HTTP
			log.Info("msg", zap.String("Serving HTTP on", "0.0.0.0"), zap.Int("port", h.Port))
		}()
	}

	log.Fatal("msg", zap.Error(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), h)))
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="{{.lang}}">
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
    <h2>{{.directoryText}} {{.relativePath}}</h2>
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
