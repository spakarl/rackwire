package model_test

import (
	"testing"

	"github.com/spakarl/rackwire/internal/model"
)

func TestSortDevicesByPositionThenName(t *testing.T) {
	devs := []model.Device{
		{ID: "c", Name: "Charlie", Position: 2},
		{ID: "a", Name: "Alpha", Position: 1},
		{ID: "b", Name: "Bravo", Position: 1},
		{ID: "z", Name: "Zero", Position: 0},
	}
	sorted := model.SortDevices(devs)
	want := []string{"Zero", "Alpha", "Bravo", "Charlie"}
	if len(sorted) != len(want) {
		t.Fatalf("len=%d", len(sorted))
	}
	for i, name := range want {
		if sorted[i].Name != name {
			t.Fatalf("i=%d got %q want %q", i, sorted[i].Name, name)
		}
	}
	// original unchanged
	if devs[0].Name != "Charlie" {
		t.Fatal("SortDevices mutated input")
	}
}
