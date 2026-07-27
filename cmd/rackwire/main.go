package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spakarl/rackwire/internal/porttpl"
	"github.com/spakarl/rackwire/internal/store"
	"github.com/spakarl/rackwire/internal/web"
	"github.com/spakarl/rackwire/internal/wirecolor"
	"github.com/spakarl/rackwire/ui"
)

func main() {
	addr := env("ADDR", ":3040")
	dataPath := env("DATA_PATH", "data/rack.json")
	dataDir := filepath.Dir(dataPath)
	templatesDir := env("TEMPLATES_DIR", filepath.Join(dataDir, "templates"))
	colorsDir := env("COLORS_DIR", filepath.Join(dataDir, "colors"))

	cat, err := porttpl.Open(templatesDir)
	if err != nil {
		log.Fatal(err)
	}

	colors, err := wirecolor.Open(colorsDir)
	if err != nil {
		log.Fatal(err)
	}

	st, err := store.New(dataPath, cat)
	if err != nil {
		log.Fatal(err)
	}

	srv, err := web.New(st, cat, colors, ui.Content)
	if err != nil {
		log.Fatal(err)
	}

	absData, _ := filepath.Abs(dataPath)
	absTpl, _ := filepath.Abs(templatesDir)
	absCol, _ := filepath.Abs(colorsDir)
	log.Printf("rackwire listening on %s (data: %s, templates: %s, colors: %s)", addr, absData, absTpl, absCol)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
