package porttpl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spakarl/rackwire/internal/model"
	"github.com/spakarl/rackwire/internal/porttpl"
)

func TestCatalogSeedsAndSave(t *testing.T) {
	dir := t.TempDir()
	cat, err := porttpl.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	list := cat.List()
	if len(list) < 5 {
		t.Fatalf("expected seeds, got %d", len(list))
	}
	def := cat.ByID(porttpl.DefaultID)
	if def == nil || len(def.Pins) != 8 {
		t.Fatalf("default template: %#v", def)
	}
	if _, err := os.Stat(filepath.Join(dir, "rj45-t568a.json")); err != nil {
		t.Fatal(err)
	}

	err = cat.Save(model.PortTemplate{
		Name: "Custom ISDN",
		Pins: []model.Pin{{Number: 3, Signal: "A", Color: "BU", ColorHex: "#3498db"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := cat.ByID("custom-isdn")
	if got == nil || got.Name != "Custom ISDN" {
		t.Fatalf("saved: %#v", got)
	}
	if err := cat.Delete("custom-isdn"); err != nil {
		t.Fatal(err)
	}
	if err := cat.Delete(porttpl.DefaultID); err == nil {
		t.Fatal("expected seed delete to fail")
	}
}
