package api

type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
}

type Relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

type relationIndex struct {
	Index []Relation `json:"index"`
}

type Concert struct {
	Location string
	Dates    []string
}

type ArtistDetail struct {
	Artist
	Concerts []Concert
}

type SearchResult struct {
	ID    int
	Name  string
	Image string
	Match string
}
