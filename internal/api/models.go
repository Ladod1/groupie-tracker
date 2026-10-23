// Package api fetches and organizes data from the Groupie Trackers API.
package api

// Artist is one band or solo artist as returned by the /artists endpoint.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

// Relation links an artist's concert locations to their dates.
type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// relationIndex is the wrapper object returned by the /relation endpoint.
type relationIndex struct {
	Index []Relation `json:"index"`
}

// Concert is one location and every date the artist plays there.
type Concert struct {
	Location string
	Dates    []string
}

// ArtistDetail combines an artist with their concerts, ready for display.
type ArtistDetail struct {
	Artist
	Concerts []Concert
}

// SearchResult is one artist matching a search query.
type SearchResult struct {
	ID    int
	Name  string
	Image string
	Match string
}
