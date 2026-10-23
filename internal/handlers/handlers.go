// Package handlers serves the website's pages.
package handlers

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"groupie-tracker/internal/api"
)

// DataSource is where the handlers get artist data from.
type DataSource interface {
	Artist(id int) (api.ArtistDetail, error)
	Search(query string) ([]api.SearchResult, error)
}

// App holds what the handlers need to build responses.
type App struct {
	data DataSource
	tmpl *template.Template
}

type homePage struct {
	Query   string
	Results []api.SearchResult
}

type errorPage struct {
	Code    int
	Status  string
	Message string
}

// New parses the HTML templates in templateDir and returns an App.
func New(data DataSource, templateDir string) (*App, error) {
	tmpl, err := template.ParseGlob(filepath.Join(templateDir, "*.html"))
	if err != nil {
		return nil, err
	}
	return &App{data: data, tmpl: tmpl}, nil
}

// Routes returns the site's HTTP handler. Static files are served from staticDir.
func (a *App) Routes(staticDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.home)
	mux.HandleFunc("/artist", a.artist)
	mux.Handle("/static/", a.static(staticDir))
	return a.recoverPanic(mux)
}

// home shows every artist, or only those matching ?q= when given.
func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		a.renderError(w, http.StatusNotFound, "The page you are looking for does not exist.")
		return
	}
	if !allowGet(w, r) {
		a.renderError(w, http.StatusMethodNotAllowed, "Only GET requests are allowed on this page.")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	results, err := a.data.Search(query)
	if err != nil {
		log.Printf("home: %v", err)
		a.renderError(w, http.StatusInternalServerError, "Artist data is unavailable right now. Please try again later.")
		return
	}
	a.render(w, http.StatusOK, "index.html", homePage{Query: query, Results: results})
}

// artist shows one artist's details, chosen by ?id=.
func (a *App) artist(w http.ResponseWriter, r *http.Request) {
	if !allowGet(w, r) {
		a.renderError(w, http.StatusMethodNotAllowed, "Only GET requests are allowed on this page.")
		return
	}

	id, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || id < 1 {
		a.renderError(w, http.StatusBadRequest, "The artist id must be a positive number.")
		return
	}

	detail, err := a.data.Artist(id)
	if errors.Is(err, api.ErrNotFound) {
		a.renderError(w, http.StatusNotFound, "No artist exists with this id.")
		return
	}
	if err != nil {
		log.Printf("artist %d: %v", id, err)
		a.renderError(w, http.StatusInternalServerError, "Artist data is unavailable right now. Please try again later.")
		return
	}
	a.render(w, http.StatusOK, "artist.html", detail)
}

// static serves files from dir, without directory listings.
func (a *App) static(dir string) http.Handler {
	root := http.Dir(dir)
	files := http.StripPrefix("/static/", http.FileServer(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowGet(w, r) {
			a.renderError(w, http.StatusMethodNotAllowed, "Only GET requests are allowed.")
			return
		}

		f, err := root.Open(strings.TrimPrefix(r.URL.Path, "/static"))
		if err != nil {
			a.renderError(w, http.StatusNotFound, "The file you are looking for does not exist.")
			return
		}
		info, err := f.Stat()
		f.Close()
		if err != nil || info.IsDir() {
			a.renderError(w, http.StatusNotFound, "The file you are looking for does not exist.")
			return
		}
		files.ServeHTTP(w, r)
	})
}

// recoverPanic turns an unexpected panic into a 500 page instead of a
// dropped connection.
func (a *App) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic serving %s: %v", r.URL.Path, err)
				a.renderError(w, http.StatusInternalServerError, "Something went wrong on our side.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// render executes a template into a buffer first, so a template error
// becomes a clean 500 page instead of a half-written response.
func (a *App) render(w http.ResponseWriter, code int, name string, data any) {
	var buf bytes.Buffer
	if err := a.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
		a.renderError(w, http.StatusInternalServerError, "Something went wrong on our side.")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

func (a *App) renderError(w http.ResponseWriter, code int, message string) {
	var buf bytes.Buffer
	page := errorPage{Code: code, Status: http.StatusText(code), Message: message}
	if err := a.tmpl.ExecuteTemplate(&buf, "error.html", page); err != nil {
		log.Printf("render error page: %v", err)
		http.Error(w, http.StatusText(code), code)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	buf.WriteTo(w)
}

// allowGet reports whether r is a GET or HEAD request, and sets the
// Allow header when it is not.
func allowGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	return false
}
