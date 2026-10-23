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

const DefaultBaseURL = "https://groupietrackers.herokuapp.com/api"

const maxBodySize = 10 << 20

var ErrNotFound = errors.New("artist not found")

type Client struct {
	baseURL string
	http    *http.Client
	ttl     time.Duration

	mu        sync.Mutex
	artists   []Artist
	relations map[int]Relation
	fetchedAt time.Time
}

func NewClient(baseURL string, ttl time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
		ttl:     ttl,
	}
}

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
