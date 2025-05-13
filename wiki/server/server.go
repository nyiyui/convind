package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/google/safehtml/template"
	"inaba.kiyuri.ca/2025/convind/data"
	"inaba.kiyuri.ca/2025/convind/wiki"
)

//go:embed javascript
var javascriptFS embed.FS

type Server struct {
	mux       *http.ServeMux
	dataStore data.DataStore
	hops      *wiki.WikiClass
	tps       map[string]*template.Template
}

func New(dataStore data.DataStore) (*Server, error) {
	s := new(Server)
	s.dataStore = dataStore
	s.hops = wiki.NewWikiClass(s.dataStore)
	err := s.hops.Load()
	if err != nil {
		return nil, err
	}
	s.parseTemplates()
	s.setupRoutes()
	return s, nil
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

	s.mux.HandleFunc("GET /api/v1/page/{id}/hop", s.handlePageHop)
	s.mux.HandleFunc("GET /api/v1/page/{id}", s.handlePage)
	s.mux.HandleFunc("POST /api/v1/page/{id}", s.handlePage)
	s.mux.HandleFunc("POST /api/v1/page/new", s.handlePageNew)
	s.mux.HandleFunc("GET /api/v1/pages", s.handlePageList)

	s.mux.HandleFunc("GET /", s.handleSPA)
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
	switch r.Method {
	case "GET":
		pr, err := page.LatestRevision()
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			return
		}
		if pr == nil {
			http.Error(w, "", 204)
			return
		}
		w.Header().Set("Revision-ID", strconv.FormatUint(pr.DataRevision.RevisionID(), 10))
		rc, err := pr.DataRevision.NewReadCloser()
		defer rc.Close()
		_, err = io.Copy(w, rc)
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			// probably, the 200 header has already been written, but whatever
			return
		}
	case "POST":
		dr, err := data.NewRevision(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			return
		}
		w.Header().Set("Revision-ID", strconv.FormatUint(dr.RevisionID(), 10))
		http.Error(w, "", 204)
	}
}

func (s *Server) handlePageNew(w http.ResponseWriter, r *http.Request) {
	data, err := s.dataStore.New("text/markdown")
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	http.Redirect(w, r, filepath.Join("/api/v1/page/", data.ID().String()), 302)
}

func (s *Server) handlePageList(w http.ResponseWriter, r *http.Request) {
	ids, err := s.dataStore.AllIDs()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	datas := make([]data.Data, len(ids))
	for i, id := range ids {
		datas[i], err = s.dataStore.GetDataByID(id)
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(datas)
	if err != nil {
		// probably, the 200 header has already been written, but whatever
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate("spa.html", w, r, map[string]interface{}{})
}

func (s *Server) handlePageHop(w http.ResponseWriter, r *http.Request) {
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
	if pr == nil {
		http.Error(w, "", 204)
		return
	}
	w.Header().Set("Revision-ID", strconv.FormatUint(pr.DataRevision.RevisionID(), 10))
	i, err := s.hops.AttemptInstance(pr.DataRevision)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	rc, err := i.NewReadCloser()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	io.Copy(w, rc)
}
