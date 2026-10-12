package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	"example.com/go-rest-api/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run returns errors to main so deferred cleanup runs before the process exits.
func run() error {
	// Go does not load .env files automatically. Existing environment variables
	// still work, and godotenv does not overwrite values already set by the OS.
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	handler, disconnect, err := app.NewHandler()
	if err != nil {
		return err
	}
	defer func() {
		if err := disconnect(); err != nil {
			log.Printf("failed to disconnect MongoDB client: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API listening on http://localhost:%s", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
