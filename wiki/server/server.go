package server

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	"github.com/google/safehtml/template"
	"inaba.kiyuri.ca/2025/convind/data"
	"inaba.kiyuri.ca/2025/convind/wiki"
)

//go:embed javascript
var javascriptFS embed.FS

type Server struct {
	mux       *http.ServeMux
	dataStore data.DataStore
	tps       map[string]*template.Template
}

func New(dataStore data.DataStore) *Server {
	s := new(Server)
	s.dataStore = dataStore
	s.parseTemplates()
	s.setupRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) setupRoutes() {
	s.mux = http.NewServeMux()
	javascriptFS2, err := fs.Sub(javascriptFS, "javascript")
	if err != nil {
		panic(err)
	}
	s.mux.Handle("GET /static/js/", http.StripPrefix("/static/js/", http.FileServer(http.FS(javascriptFS2))))

	s.mux.HandleFunc("GET /api/v1/page/{id}", s.handlePage)

	s.mux.HandleFunc("GET /wiki", s.handleWiki)
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	idRaw := r.PathValue("id")
	id := new(data.ID)
	err := id.UnmarshalText([]byte(idRaw))
	if err != nil {
		http.Error(w, "invalid id", 404)
		return
	}
	data, err := s.dataStore.GetDataByID(*id)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	page := wiki.Page{data}
	pr, err := page.LatestRevision()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	rc, err := pr.DataRevision.NewReadCloser()
	defer rc.Close()
	_, err = io.Copy(w, rc)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		// probably, the 200 header has already been written, but whatever
		return
	}
}

func (s *Server) handleWiki(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate("wiki.html", w, r, map[string]interface{}{})
}
