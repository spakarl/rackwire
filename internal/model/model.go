package model

import (
	"sort"
	"strings"
)

// Pin is one physical contact on a port.
type Pin struct {
	Number   int    `json:"number"`
	Signal   string `json:"signal"`
	Color    string `json:"color"`    // human label, e.g. "WH/OR"
	ColorHex string `json:"colorHex"` // display swatch
}

// PortTemplate is a reusable pinout (RJ45 T568B, doorbell, etc.).
type PortTemplate struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Pins []Pin  `json:"pins"`
}

// Port is a labeled jack on a device, based on a template with optional overrides.
type Port struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	TemplateID string `json:"templateId"`
	Pins       []Pin  `json:"pins"`
}

// Room is a physical location that devices can be assigned to.
type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Device is a patch panel, router, wall outlet, etc.
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"` // patchpanel | router | outlet | other
	Color    string `json:"color"`
	Position int    `json:"position"` // rack unit / sort index; smaller = earlier
	RoomID   string `json:"roomId,omitempty"`
	Ports    []Port `json:"ports"`
}

// Endpoint identifies one side of a patch link.
type Endpoint struct {
	DeviceID string `json:"deviceId"`
	PortID   string `json:"portId"`
}

// Link is a cable between two ports.
type Link struct {
	ID string   `json:"id"`
	A  Endpoint `json:"a"`
	B  Endpoint `json:"b"`
}

// Rack is the full persisted document.
type Rack struct {
	Name    string   `json:"name"`
	Rooms   []Room   `json:"rooms"`
	Devices []Device `json:"devices"`
	Links   []Link   `json:"links"`
}

// RoomGroup is devices under one room (or unassigned) for the home page.
type RoomGroup struct {
	Room    *Room
	Name    string
	Devices []Device
}

func (r *Rack) DeviceByID(id string) *Device {
	for i := range r.Devices {
		if r.Devices[i].ID == id {
			return &r.Devices[i]
		}
	}
	return nil
}

func (r *Rack) RoomByID(id string) *Room {
	if id == "" {
		return nil
	}
	for i := range r.Rooms {
		if r.Rooms[i].ID == id {
			return &r.Rooms[i]
		}
	}
	return nil
}

// RoomName returns the display name for roomID, or empty if unknown/unset.
func (r *Rack) RoomName(roomID string) string {
	if rm := r.RoomByID(roomID); rm != nil {
		return rm.Name
	}
	return ""
}

// RoomsSorted returns rooms ordered by Name.
func (r *Rack) RoomsSorted() []Room {
	out := append([]Room(nil), r.Rooms...)
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// ClearRoomID removes roomID from all devices (after room delete).
func (r *Rack) ClearRoomID(roomID string) {
	for i := range r.Devices {
		if r.Devices[i].RoomID == roomID {
			r.Devices[i].RoomID = ""
		}
	}
}

// DevicesSorted returns devices ordered by Position, then Name.
func (r *Rack) DevicesSorted() []Device {
	return SortDevices(r.Devices)
}

// SortDevices returns a copy of devices ordered by Position, then Name.
func SortDevices(devices []Device) []Device {
	out := append([]Device(nil), devices...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Position != out[j].Position {
			return out[i].Position < out[j].Position
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// DeviceCountInRoom returns how many devices reference roomID.
func (r *Rack) DeviceCountInRoom(roomID string) int {
	n := 0
	for _, d := range r.Devices {
		if d.RoomID == roomID {
			n++
		}
	}
	return n
}

// DevicesGroupedByRoom returns devices grouped by room (named rooms first, then unassigned).
func (r *Rack) DevicesGroupedByRoom() []RoomGroup {
	sorted := r.DevicesSorted()
	byRoom := map[string][]Device{}
	var orphan []Device
	for _, d := range sorted {
		if d.RoomID == "" || r.RoomByID(d.RoomID) == nil {
			orphan = append(orphan, d)
			continue
		}
		byRoom[d.RoomID] = append(byRoom[d.RoomID], d)
	}
	var groups []RoomGroup
	for _, room := range r.RoomsSorted() {
		devs := byRoom[room.ID]
		if len(devs) == 0 {
			continue
		}
		rm := room
		groups = append(groups, RoomGroup{Room: &rm, Name: room.Name, Devices: devs})
	}
	if len(orphan) > 0 {
		groups = append(groups, RoomGroup{Name: "Ohne Raum", Devices: orphan})
	}
	return groups
}

func (d *Device) PortByID(id string) *Port {
	for i := range d.Ports {
		if d.Ports[i].ID == id {
			return &d.Ports[i]
		}
	}
	return nil
}

func (r *Rack) LinkForPort(deviceID, portID string) *Link {
	for i := range r.Links {
		l := &r.Links[i]
		if (l.A.DeviceID == deviceID && l.A.PortID == portID) ||
			(l.B.DeviceID == deviceID && l.B.PortID == portID) {
			return l
		}
	}
	return nil
}

func (r *Rack) EndpointLabel(deviceID, portID string) string {
	dev := r.DeviceByID(deviceID)
	if dev == nil {
		return deviceID
	}
	port := dev.PortByID(portID)
	if port == nil {
		return dev.Name
	}
	label := port.Label
	if label == "" {
		label = port.ID
	}
	return dev.Name + " · " + label
}

func (r *Rack) PeerLabel(deviceID, portID string) string {
	link := r.LinkForPort(deviceID, portID)
	if link == nil {
		return ""
	}
	other := link.B
	if link.B.DeviceID == deviceID && link.B.PortID == portID {
		other = link.A
	}
	return r.EndpointLabel(other.DeviceID, other.PortID)
}
