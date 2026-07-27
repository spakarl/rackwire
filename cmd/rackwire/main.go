package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spakarl/rackwire/internal/store"
	"github.com/spakarl/rackwire/internal/web"
	"github.com/spakarl/rackwire/ui"
)

func main() {
	addr := env("ADDR", ":3040")
	dataPath := env("DATA_PATH", "data/rack.json")

	st, err := store.New(dataPath)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := web.New(st, ui.Content)
	if err != nil {
		log.Fatal(err)
	}

	abs, _ := filepath.Abs(dataPath)
	log.Printf("rackwire listening on %s (data: %s)", addr, abs)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
