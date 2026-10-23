package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const artistsJSON = `[
	{"id": 1, "image": "queen.jpeg", "name": "Queen",
	 "members": ["Freddie Mercury", "Brian May"],
	 "creationDate": 1970, "firstAlbum": "14-12-1973"},
	{"id": 2, "image": "soja.jpeg", "name": "SOJA",
	 "members": ["Jacob Hemphill"],
	 "creationDate": 1997, "firstAlbum": "05-06-2002"}
]`

const relationJSON = `{"index": [
	{"id": 1, "datesLocations": {"london-uk": ["24-08-2019", "23-08-2019"]}},
	{"id": 2, "datesLocations": {"playa_del_carmen-mexico": ["05-12-2019"]}}
]}`

// newTestAPI starts a fake Groupie Trackers API. While *down is true
// it answers every request with a 500 error.
func newTestAPI(t *testing.T, down *bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *down {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		switch r.URL.Path {
		case "/artists":
			w.Write([]byte(artistsJSON))
		case "/relation":
			w.Write([]byte(relationJSON))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestArtist(t *testing.T) {
	down := false
	c := NewClient(newTestAPI(t, &down).URL, time.Minute)

	got, err := c.Artist(1)
	if err != nil {
		t.Fatalf("Artist(1) error: %v", err)
	}
	if got.Name != "Queen" || got.FirstAlbum != "14 Dec 1973" {
		t.Errorf("Artist(1) = %q / %q, want Queen / 14 Dec 1973", got.Name, got.FirstAlbum)
	}
	if len(got.Concerts) != 1 || got.Concerts[0].Location != "London, UK" {
		t.Fatalf("Artist(1) concerts = %+v", got.Concerts)
	}
	if got.Concerts[0].Dates[0] != "23 Aug 2019" {
		t.Errorf("dates not sorted: %v", got.Concerts[0].Dates)
	}

	if _, err := c.Artist(99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Artist(99) error = %v, want ErrNotFound", err)
	}
}

func TestSearch(t *testing.T) {
	down := false
	c := NewClient(newTestAPI(t, &down).URL, time.Minute)

	tests := []struct {
		query     string
		wantNames []string
		wantMatch string
	}{
		{"", []string{"Queen", "SOJA"}, ""},
		{"  qUeEn ", []string{"Queen"}, ""},
		{"freddie", []string{"Queen"}, "Member: Freddie Mercury"},
		{"1997", []string{"SOJA"}, "Created: 1997"},
		{"mexico", []string{"SOJA"}, "Location: Playa Del Carmen, Mexico"},
		{"nothing matches this", nil, ""},
	}
	for _, tt := range tests {
		results, err := c.Search(tt.query)
		if err != nil {
			t.Fatalf("Search(%q) error: %v", tt.query, err)
		}
		if len(results) != len(tt.wantNames) {
			t.Errorf("Search(%q) returned %d results, want %d", tt.query, len(results), len(tt.wantNames))
			continue
		}
		for i, r := range results {
			if r.Name != tt.wantNames[i] {
				t.Errorf("Search(%q)[%d] = %q, want %q", tt.query, i, r.Name, tt.wantNames[i])
			}
		}
		if len(results) == 1 && results[0].Match != tt.wantMatch {
			t.Errorf("Search(%q) match = %q, want %q", tt.query, results[0].Match, tt.wantMatch)
		}
	}
}

func TestAPIDownWithoutCache(t *testing.T) {
	down := true
	c := NewClient(newTestAPI(t, &down).URL, time.Minute)

	if _, err := c.Search(""); err == nil {
		t.Error("Search with API down: want error, got nil")
	}
}

func TestAPIDownUsesCachedData(t *testing.T) {
	down := false
	c := NewClient(newTestAPI(t, &down).URL, 0) // TTL 0: refresh every call

	if _, err := c.Search(""); err != nil {
		t.Fatalf("first Search error: %v", err)
	}

	down = true
	results, err := c.Search("")
	if err != nil {
		t.Fatalf("Search with API down after caching: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d cached results, want 2", len(results))
	}
}
