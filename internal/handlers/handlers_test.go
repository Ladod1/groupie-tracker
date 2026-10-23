package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"groupie-tracker/internal/api"
)

// fakeData is a DataSource that answers from memory.
type fakeData struct {
	err error
}

func (f fakeData) Artist(id int) (api.ArtistDetail, error) {
	if f.err != nil {
		return api.ArtistDetail{}, f.err
	}
	if id != 1 {
		return api.ArtistDetail{}, api.ErrNotFound
	}
	return api.ArtistDetail{
		Artist:   api.Artist{ID: 1, Name: "Queen", Members: []string{"Freddie Mercury"}, CreationDate: 1970},
		Concerts: []api.Concert{{Location: "London, UK", Dates: []string{"23 Aug 2019"}}},
	}, nil
}

func (f fakeData) Search(query string) ([]api.SearchResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if query == "" || strings.Contains("queen", strings.ToLower(query)) {
		return []api.SearchResult{{ID: 1, Name: "Queen"}}, nil
	}
	return nil, nil
}

func newTestServer(t *testing.T, data DataSource) http.Handler {
	t.Helper()
	app, err := New(data, "../../templates")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return app.Routes("../../static")
}

func TestRoutes(t *testing.T) {
	ok := newTestServer(t, fakeData{})
	broken := newTestServer(t, fakeData{err: errors.New("api down")})

	tests := []struct {
		name     string
		handler  http.Handler
		method   string
		target   string
		wantCode int
		wantBody string
	}{
		{"home", ok, http.MethodGet, "/", http.StatusOK, "Queen"},
		{"search hit", ok, http.MethodGet, "/?q=que", http.StatusOK, "Queen"},
		{"search miss", ok, http.MethodGet, "/?q=zzz", http.StatusOK, "No artists match"},
		{"home wrong method", ok, http.MethodPost, "/", http.StatusMethodNotAllowed, "Method Not Allowed"},
		{"unknown page", ok, http.MethodGet, "/nope", http.StatusNotFound, "Not Found"},
		{"artist", ok, http.MethodGet, "/artist?id=1", http.StatusOK, "London, UK"},
		{"artist missing id", ok, http.MethodGet, "/artist", http.StatusBadRequest, "Bad Request"},
		{"artist bad id", ok, http.MethodGet, "/artist?id=abc", http.StatusBadRequest, "Bad Request"},
		{"artist negative id", ok, http.MethodGet, "/artist?id=-3", http.StatusBadRequest, "Bad Request"},
		{"artist unknown id", ok, http.MethodGet, "/artist?id=999", http.StatusNotFound, "Not Found"},
		{"artist wrong method", ok, http.MethodDelete, "/artist?id=1", http.StatusMethodNotAllowed, "Method Not Allowed"},
		{"stylesheet", ok, http.MethodGet, "/static/style.css", http.StatusOK, "--accent"},
		{"static missing file", ok, http.MethodGet, "/static/missing.css", http.StatusNotFound, "Not Found"},
		{"static no listing", ok, http.MethodGet, "/static/", http.StatusNotFound, "Not Found"},
		{"home api down", broken, http.MethodGet, "/", http.StatusInternalServerError, "Internal Server Error"},
		{"artist api down", broken, http.MethodGet, "/artist?id=1", http.StatusInternalServerError, "Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.target, nil))

			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body does not contain %q", tt.wantBody)
			}
		})
	}
}

func TestRecoverPanic(t *testing.T) {
	app, err := New(fakeData{}, "../../templates")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := app.recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
