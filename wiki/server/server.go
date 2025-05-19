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
	"time"

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
	jsFileServer := http.FileServer(http.FS(javascriptFS2))
	// Add cache headers for static JS files
	jsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
		jsFileServer.ServeHTTP(w, r)
	})
	s.mux.Handle("GET /static/js/", http.StripPrefix("/static/js/", jsHandler))

	s.mux.HandleFunc("GET /api/v1/page/{id}", s.handlePage)
	s.mux.HandleFunc("POST /api/v1/page/{id}", s.handlePage)
	s.mux.HandleFunc("POST /api/v1/page/new", s.handlePageNew)
	s.mux.HandleFunc("GET /api/v1/pages", s.handlePageList)
	s.mux.HandleFunc("POST /api/v1/data/new", s.handleDataNew)
	s.mux.HandleFunc("GET /api/v1/data/{id}", s.handleData)
	s.mux.HandleFunc("DELETE /api/v1/data/{id}", s.handleDeleteData)
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

		// Add ETag based on revision ID
		etag := fmt.Sprintf("\"%d\"", pr.DataRevision.RevisionID())
		w.Header().Set("ETag", etag)
		w.Header().Set("Revision-ID", strconv.FormatUint(pr.DataRevision.RevisionID(), 10))
		w.Header().Set("Last-Modified", pr.DataRevision.CreationTime().Format(time.RFC1123))

		// Check If-None-Match header for 304 Not Modified response
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		rc, err := pr.DataRevision.NewReadCloser()
		if err != nil {
			http.Error(w, fmt.Sprint(err), 500)
			return
		}
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
		// No caching for POST responses
		w.Header().Set("Cache-Control", "no-store")
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

	// For ETag generation, we'll use a combined string of all revision IDs
	var etagParts []string

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

		// Add to ETag data
		rev, err := LatestRevision(d)
		if err == nil && rev != nil {
			etagParts = append(etagParts, strconv.FormatUint(rev.RevisionID(), 10))
		}
	}

	// Generate ETag from all revision IDs
	etag := fmt.Sprintf("\"%s\"", strconv.FormatUint(uint64(len(ids)), 10))
	if len(etagParts) > 0 {
		etag = fmt.Sprintf("\"%s\"", strconv.Itoa(len(etagParts)))
	}
	w.Header().Set("ETag", etag)

	// Check If-None-Match header
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set cache headers - shorter max-age since this is a list that could change frequently
	w.Header().Set("Cache-Control", "public, must-revalidate, max-age=30") // Cache for 30 seconds, then revalidate
	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(entries)
	if err != nil {
		// probably, the 200 header has already been written, but whatever
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
}

func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	// Set cache headers for SPA - allow client cache but revalidate frequently
	// We use a short cache time because the SPA itself might change
	w.Header().Set("Cache-Control", "public, must-revalidate, max-age=300") // Cache for 5 minutes, then revalidate

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

	// Add ETag based on revision ID if one exists
	if dr != nil {
		etag := fmt.Sprintf("\"%d\"", dr.RevisionID())
		w.Header().Set("ETag", etag)

		// Check If-None-Match header to respond with 304 Not Modified when appropriate
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Set cache control - allow client cache but revalidate
		w.Header().Set("Cache-Control", "public, must-revalidate, max-age=60") // Cache for a minute, then revalidate
	}

	if dr == nil {
		w.WriteHeader(204)
		return
	}
	rc, err := dr.NewReadCloser()
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	defer rc.Close()
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

	// Add ETag based on revision ID if one exists
	if dr != nil {
		etag := fmt.Sprintf("\"%d\"", dr.RevisionID())
		w.Header().Set("ETag", etag)

		// Check If-None-Match header
		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Set cache control - allow client cache but revalidate
		w.Header().Set("Cache-Control", "public, must-revalidate, max-age=60") // Cache for a minute, then revalidate
	}

	if dr == nil {
		http.Error(w, "[]", 200)
		return
	}
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
	if dr == nil {
		http.Error(w, "", 204)
		return
	}

	// Add ETag based on revision ID
	etag := fmt.Sprintf("\"%s-%d\"", className, dr.RevisionID())
	w.Header().Set("ETag", etag)

	// Check If-None-Match header
	if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set cache control - allow client cache but revalidate
	w.Header().Set("Cache-Control", "public, must-revalidate, max-age=60") // Cache for a minute, then revalidate

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
	defer rc.Close()
	_, err = io.Copy(w, rc)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		// probably, the 200 header has already been written, but whatever
		return
	}
}

func (s *Server) handleDeleteData(w http.ResponseWriter, r *http.Request) {
	idRaw := r.PathValue("id")
	id := new(data.ID)
	err := id.UnmarshalText([]byte(idRaw))
	if err != nil {
		http.Error(w, "invalid id", 404)
		return
	}
	err = s.dataStore.DeleteByID(*id)
	if err != nil {
		http.Error(w, fmt.Sprint(err), 500)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
