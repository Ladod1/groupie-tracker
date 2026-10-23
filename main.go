package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := api.NewClient(api.DefaultBaseURL, 10*time.Minute)
	app, err := handlers.New(client, "templates")
	if err != nil {
		log.Fatalf("load templates: %v", err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           app.Routes("static"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("listening on http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
