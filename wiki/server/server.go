package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"slices"
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
	wikiClass *wiki.WikiClass
	classes   []data.Class
	tps       map[string]*template.Template
}

func New(dataStore data.DataStore) (*Server, error) {
	s := new(Server)
	s.dataStore = dataStore
	s.wikiClass = wiki.NewWikiClass(s.dataStore)
	err := s.wikiClass.Load()
	if err != nil {
		return nil, err
	}
	s.classes = append(s.classes, s.wikiClass)
	s.parseTemplates()
	s.setupRoutes()
	return s, nil
}

func (s *Server) AddClass(class data.Class) {
	s.classes = append(s.classes, class)
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
	s.mux.HandleFunc("POST /api/v1/page/{id}", s.handlePage)
	s.mux.HandleFunc("POST /api/v1/page/new", s.handlePageNew)
	s.mux.HandleFunc("GET /api/v1/pages", s.handlePageList)
	s.mux.HandleFunc("POST /api/v1/data/new", s.handleDataNew)
	s.mux.HandleFunc("GET /api/v1/data/{id}", s.handleData)
	s.mux.HandleFunc("GET /api/v1/data/{id}/instances", s.handleDataInstances)
	s.mux.HandleFunc("GET /api/v1/data/{id}/instance/{className}", s.handleDataInstance)

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
	if data.MIMEType() != "text/markdown" {
		if err != nil {
			http.Error(w, "MIME type is not text/markdown", 404)
			return
		}
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

type pageEntry struct {
	Data                data.Data
	LatestRevisionTitle string
}

func (s *Server) handlePageList(w http.ResponseWriter, r *http.Request) {
	ids, err := s.dataStore.AllIDs()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	entries := make([]pageEntry, len(ids))
	for i, id := range ids {
		d, err := s.dataStore.GetDataByID(id)
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			return
		}
		title, err := (&wiki.Page{d}).LatestRevisionTitle()
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			return
		}
		entries[i] = pageEntry{Data: d, LatestRevisionTitle: title}
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entries)
	if err != nil {
		// probably, the 200 header has already been written, but whatever
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	s.renderTemplate("spa.html", w, r, map[string]interface{}{})
}

func (s *Server) handleDataNew(w http.ResponseWriter, r *http.Request) {
	mimeType := r.Header.Get("Content-Type")
	d, err := s.dataStore.New(mimeType)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	dr, err := d.NewRevision(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	http.Redirect(w, r, filepath.Join("/api/v1/data", d.ID().String())+"?revision-id="+strconv.FormatUint(dr.RevisionID(), 10), 302)
}

func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
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
	dr, err := LatestRevision(data)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	w.Header().Set("Content-Type", data.MIMEType())
	rc, err := dr.NewReadCloser()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	_, err = io.Copy(w, rc)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		// probably, the 200 header has already been written, but whatever
		return
	}
}

// LatestRevision returns the latest revision if available, and nil is there are no revisions at all.
func LatestRevision(d data.Data) (data.DataRevision, error) {
	revisions, err := d.Revisions()
	if err != nil {
		return nil, err
	}
	var latestRevision data.DataRevision
	for _, revision := range revisions {
		if latestRevision == nil || revision.CreationTime().After(latestRevision.CreationTime()) {
			latestRevision = revision
		}
	}
	return latestRevision, nil
}

func (s *Server) handleDataInstances(w http.ResponseWriter, r *http.Request) {
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
	dr, err := getDataRevision(r, data)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	availableClassNames := make([]string, 0)
	for _, class := range s.classes {
		_, err := class.AttemptInstance(dr)
		if err == nil {
			availableClassNames = append(availableClassNames, class.Name())
		}
	}
	err = json.NewEncoder(w).Encode(availableClassNames)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		// probably, the 200 header has already been written, but whatever
		return
	}
}

func classWithName(name string) func(data.Class) bool {
	return func(c data.Class) bool { return c.Name() == name }
}

func (s *Server) handleDataInstance(w http.ResponseWriter, r *http.Request) {
	className := r.PathValue("className")
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
	dr, err := getDataRevision(r, data)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	classIndex := slices.IndexFunc(s.classes, classWithName(className))
	if classIndex == -1 {
		http.Error(w, "no such class", 404)
		return
	}
	class := s.classes[classIndex]
	instance, err := class.AttemptInstance(dr)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	w.Header().Set("Content-Type", instance.MIMEType())
	rc, err := instance.NewReadCloser()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	_, err = io.Copy(w, rc)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		// probably, the 200 header has already been written, but whatever
		return
	}
}

// getDataRevision returns the revision requested by the revision-id query parameter, or the latest revision otherwise.
// Errors are only returned when [LatestRevision] or [data.Data.Revisions] fails.
// That is, no malformed user input will cause an error, and thus it is correct to return a 500 response on an error from getDataRevision.
func getDataRevision(r *http.Request, d data.Data) (data.DataRevision, error) {
	var revisionID uint64
	var revisions []data.DataRevision
	var err error
	revisionIDRaw := r.URL.Query().Get("revision-id")
	if revisionIDRaw == "" {
		goto LatestRevision
	}
	revisionID, err = strconv.ParseUint(revisionIDRaw, 10, 64)
	if err != nil {
		goto LatestRevision
	}
	revisions, err = d.Revisions()
	if err != nil {
		return nil, err
	}
	for _, revision := range revisions {
		if revision.RevisionID() == revisionID {
			return revision, nil
		}
	}
	// revision ID not found, so fall back to latest
LatestRevision:
	return LatestRevision(d)
}
