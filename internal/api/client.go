package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultBaseURL is the address of the Groupie Trackers API.
const DefaultBaseURL = "https://groupietrackers.herokuapp.com/api"

// maxBodySize limits how much data is read from one API response.
const maxBodySize = 10 << 20

// ErrNotFound is returned when an artist does not exist.
var ErrNotFound = errors.New("artist not found")

// Client fetches data from the API and caches it for a while,
// so every page view does not need a new network call.
type Client struct {
	baseURL string
	http    *http.Client
	ttl     time.Duration

	mu        sync.Mutex
	artists   []Artist
	relations map[int]Relation
	fetchedAt time.Time
}

// NewClient returns a Client that caches API data for ttl.
func NewClient(baseURL string, ttl time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
		ttl:     ttl,
	}
}

// Artist returns one artist with their concerts, or ErrNotFound.
func (c *Client) Artist(id int) (ArtistDetail, error) {
	artists, relations, err := c.data()
	if err != nil {
		return ArtistDetail{}, err
	}

	for _, a := range artists {
		if a.ID == id {
			a.FirstAlbum = FormatDate(a.FirstAlbum)
			return ArtistDetail{
				Artist:   a,
				Concerts: buildConcerts(relations[id].DatesLocations),
			}, nil
		}
	}
	return ArtistDetail{}, ErrNotFound
}

// Search returns the artists matching query by name, member, creation
// year, first album date or concert location. An empty query returns
// every artist.
func (c *Client) Search(query string) ([]SearchResult, error) {
	artists, relations, err := c.data()
	if err != nil {
		return nil, err
	}

	q := strings.ToLower(strings.TrimSpace(query))
	results := []SearchResult{}
	for _, a := range artists {
		match, ok := matchArtist(a, relations[a.ID], q)
		if ok {
			results = append(results, SearchResult{ID: a.ID, Name: a.Name, Image: a.Image, Match: match})
		}
	}
	return results, nil
}

// matchArtist reports whether a matches q and describes what matched.
func matchArtist(a Artist, rel Relation, q string) (string, bool) {
	if q == "" || strings.Contains(strings.ToLower(a.Name), q) {
		return "", true
	}
	for _, m := range a.Members {
		if strings.Contains(strings.ToLower(m), q) {
			return "Member: " + m, true
		}
	}
	if strings.Contains(strconv.Itoa(a.CreationDate), q) {
		return "Created: " + strconv.Itoa(a.CreationDate), true
	}
	if strings.Contains(a.FirstAlbum, q) {
		return "First album: " + FormatDate(a.FirstAlbum), true
	}
	for raw := range rel.DatesLocations {
		loc := FormatLocation(raw)
		if strings.Contains(strings.ToLower(loc), q) {
			return "Location: " + loc, true
		}
	}
	return "", false
}

// data returns cached API data, refreshing it when it is older than
// the TTL. If a refresh fails but older data exists, the old data is
// used so the site keeps working while the API is down.
func (c *Client) data() ([]Artist, map[int]Relation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.artists != nil && time.Since(c.fetchedAt) < c.ttl {
		return c.artists, c.relations, nil
	}

	artists, relations, err := c.fetch()
	if err != nil {
		if c.artists != nil {
			log.Printf("api refresh failed, using cached data: %v", err)
			return c.artists, c.relations, nil
		}
		return nil, nil, err
	}

	c.artists, c.relations, c.fetchedAt = artists, relations, time.Now()
	return artists, relations, nil
}

func (c *Client) fetch() ([]Artist, map[int]Relation, error) {
	var artists []Artist
	if err := c.getJSON("/artists", &artists); err != nil {
		return nil, nil, err
	}

	var index relationIndex
	if err := c.getJSON("/relation", &index); err != nil {
		return nil, nil, err
	}

	relations := make(map[int]Relation, len(index.Index))
	for _, r := range index.Index {
		relations[r.ID] = r
	}
	return artists, relations, nil
}

func (c *Client) getJSON(path string, v any) error {
	resp, err := c.http.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("get %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get %s: unexpected status %s", path, resp.Status)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodySize)).Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
