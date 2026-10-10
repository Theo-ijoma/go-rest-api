package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"example.com/go-rest-api/internal/todos"
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

	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		return errors.New("MONGODB_URI must be set in .env or the environment")
	}

	// Create one MongoDB client and reuse its built-in connection pool.
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("failed to disconnect MongoDB client: %v", err)
		}
	}()

	// Ping once at startup so a bad URI or unreachable cluster fails clearly.
	pingContext, cancelPing := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPing()
	if err := client.Ping(pingContext, nil); err != nil {
		return err
	}

	databaseName := os.Getenv("MONGODB_DATABASE")
	if databaseName == "" {
		databaseName = "go_rest_api"
	}
	store := todos.NewStore(client.Database(databaseName).Collection("todos"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// The Handler receives the MongoDB repository; it does not need to know
	// about connection setup or which database driver is being used.
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           todos.NewHandler(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("API listening on http://localhost:%s (MongoDB database: %s)", port, databaseName)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
