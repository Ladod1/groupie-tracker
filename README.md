# Groupie Tracker

A website, written in Go, that shows bands and artists and their concerts,
using data from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api).

Only the Go standard library is used. There is no JavaScript: every action is
a normal request to the Go server, which answers with a new HTML page.

## Run

```sh
go run .
```

Then open <http://localhost:8080>. Set `PORT` to use another port.

## Pages

| Route | What it shows |
|---|---|
| `/` | Every artist as a card |
| `/?q=...` | Search: artists matching a name, member, creation year, first album date or concert location |
| `/artist?id=N` | One artist: members, creation year, first album and a table of concert locations with their dates |

Errors get their own page with the right status code: `400` for an invalid
id, `404` for unknown pages or artists, `405` for a method other than GET,
and `500` when the API cannot be reached.

## Event (client → server)

The search form sends `GET /?q=...` to the server. The server filters the
artists and sends back the page with the results.

## Structure

```
main.go                  starts the server
internal/api/            fetches, caches and formats API data
internal/handlers/       HTTP handlers and error pages
templates/               HTML templates
static/                  CSS
```

API data is cached for 10 minutes. If the API goes down after data was
loaded once, the cached data keeps being shown.
