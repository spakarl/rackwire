package maplayout_test

import (
	"strings"
	"testing"

	"github.com/spakarl/rackwire/internal/maplayout"
	"github.com/spakarl/rackwire/internal/model"
)

func TestBuildPlacesRackLeftFieldRight(t *testing.T) {
	rack := &model.Rack{
		Name: "Test",
		Devices: []model.Device{
			{ID: "pp", Name: "Panel", Kind: "patchpanel", Color: "#00f", Ports: []model.Port{{ID: "p1", Label: "A-01"}}},
			{ID: "out", Name: "Dose", Kind: "outlet", Color: "#f00", Ports: []model.Port{{ID: "o1", Label: "O-01"}}},
		},
		Links: []model.Link{{
			ID: "l1",
			A:  model.Endpoint{DeviceID: "pp", PortID: "p1"},
			B:  model.Endpoint{DeviceID: "out", PortID: "o1"},
		}},
	}
	d := maplayout.Build(rack)
	if len(d.Left) != 1 || len(d.Right) != 1 {
		t.Fatalf("left=%d right=%d", len(d.Left), len(d.Right))
	}
	if d.Left[0].X >= d.Right[0].X {
		t.Fatalf("expected left column before right: %v >= %v", d.Left[0].X, d.Right[0].X)
	}
	if len(d.Curves) != 1 {
		t.Fatalf("curves=%d", len(d.Curves))
	}
	if !strings.HasPrefix(d.Curves[0].Path, "M ") {
		t.Fatalf("path %q", d.Curves[0].Path)
	}
	if !d.HasLinks {
		t.Fatal("HasLinks")
	}
}
