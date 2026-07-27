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

func TestDevicesGroupedByRoom(t *testing.T) {
	rack := model.Rack{
		Rooms: []model.Room{
			{ID: "keller", Name: "Keller"},
			{ID: "wohnzimmer", Name: "Wohnzimmer"},
		},
		Devices: []model.Device{
			{ID: "1", Name: "Panel", RoomID: "wohnzimmer", Position: 2},
			{ID: "2", Name: "Dose", RoomID: "wohnzimmer", Position: 1},
			{ID: "3", Name: "Rack", RoomID: "keller"},
			{ID: "4", Name: "Orphan", RoomID: ""},
			{ID: "5", Name: "Ghost", RoomID: "missing"},
		},
	}
	groups := rack.DevicesGroupedByRoom()
	if len(groups) != 3 {
		t.Fatalf("groups=%d %#v", len(groups), groups)
	}
	if groups[0].Name != "Keller" || len(groups[0].Devices) != 1 {
		t.Fatalf("first: %#v", groups[0])
	}
	if groups[1].Name != "Wohnzimmer" || groups[1].Devices[0].Name != "Dose" {
		t.Fatalf("second: %#v", groups[1])
	}
	if groups[2].Name != "Ohne Raum" || len(groups[2].Devices) != 2 {
		t.Fatalf("orphan: %#v", groups[2])
	}
	rack.ClearRoomID("wohnzimmer")
	if rack.DeviceByID("1").RoomID != "" || rack.DeviceByID("2").RoomID != "" {
		t.Fatal("ClearRoomID failed")
	}
}
