package wirecolor_test

import (
	"path/filepath"
	"testing"

	"github.com/spakarl/rackwire/internal/wirecolor"
)

func TestCatalogResolvesStripes(t *testing.T) {
	dir := t.TempDir()
	cat, err := wirecolor.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := filepath.Glob(filepath.Join(dir, "*.json")); err != nil {
		t.Fatal(err)
	}
	solid := cat.Resolve("OR")
	if !solid.Solid || solid.Hex == "" {
		t.Fatalf("OR: %#v", solid)
	}
	stripe := cat.Resolve("WH/OR")
	if stripe.Solid || stripe.BaseHex == "" || stripe.StripeHex == "" {
		t.Fatalf("WH/OR: %#v", stripe)
	}
	opts := cat.Options()
	if len(opts) < 10 {
		t.Fatalf("options=%d", len(opts))
	}
	unknown := cat.Resolve("CUSTOM")
	if !unknown.Solid || unknown.ID != "CUSTOM" {
		t.Fatalf("unknown: %#v", unknown)
	}
	if cat.Known("CUSTOM") {
		t.Fatal("CUSTOM should be unknown")
	}
	if !cat.Known("WH/OR") {
		t.Fatal("WH/OR should be known")
	}
	groups := cat.GroupedOptions()
	if len(groups) < 1 || len(groups[0].Options) < 10 {
		t.Fatalf("groups=%#v", groups)
	}
}
