package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/spakarl/rackwire/internal/model"
	"github.com/spakarl/rackwire/internal/porttpl"
)

type Store struct {
	mu   sync.Mutex
	path string
	cat  *porttpl.Catalog
	rack model.Rack
}

func New(path string, cat *porttpl.Catalog) (*Store, error) {
	s := &Store{path: path, cat: cat}
	if err := s.loadOrSeed(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) loadOrSeed() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		s.rack = seedRack(s.cat)
		return s.saveLocked()
	}
	if err := json.Unmarshal(data, &s.rack); err != nil {
		return fmt.Errorf("parse %s: %w", s.path, err)
	}
	return nil
}

func (s *Store) Get() model.Rack {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneRack(s.rack)
}

func (s *Store) Update(fn func(r *model.Rack) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.rack); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.rack, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func cloneRack(r model.Rack) model.Rack {
	b, _ := json.Marshal(r)
	var out model.Rack
	_ = json.Unmarshal(b, &out)
	return out
}

func seedRack(cat *porttpl.Catalog) model.Rack {
	id := sequentialID()
	ppTpl := porttpl.DefaultID

	pp1 := model.Device{
		ID: id(), Name: "Patchpanel A", Kind: "patchpanel", Color: "#2980b9",
		Ports: cat.NewPorts(24, ppTpl, "A-", id),
	}
	pp2 := model.Device{
		ID: id(), Name: "Patchpanel B", Kind: "patchpanel", Color: "#16a085",
		Ports: cat.NewPorts(24, ppTpl, "B-", id),
	}
	pp3 := model.Device{
		ID: id(), Name: "Patchpanel C", Kind: "patchpanel", Color: "#8e44ad",
		Ports: cat.NewPorts(24, ppTpl, "C-", id),
	}
	router := model.Device{
		ID: id(), Name: "Router", Kind: "router", Color: "#c0392b",
		Ports: cat.NewPorts(8, ppTpl, "LAN-", id),
	}
	doorbell := model.Device{
		ID: id(), Name: "Dose Eingang (Klingel)", Kind: "outlet", Color: "#d35400",
		Ports: cat.NewPorts(1, "doorbell-2pin", "KLG-", id),
	}
	phone := model.Device{
		ID: id(), Name: "Dose Analog Telefon", Kind: "outlet", Color: "#27ae60",
		Ports: cat.NewPorts(1, "analog-phone-2", "TEL-", id),
	}
	isdn := model.Device{
		ID: id(), Name: "Dose ISDN", Kind: "outlet", Color: "#7f8c8d",
		Ports: cat.NewPorts(1, "isdn-4pin", "ISDN-", id),
	}

	return model.Rack{
		Name:    "Homelab Rack",
		Devices: []model.Device{pp1, pp2, pp3, router, doorbell, phone, isdn},
		Links: []model.Link{
			{
				ID: id(),
				A:  model.Endpoint{DeviceID: pp1.ID, PortID: pp1.Ports[0].ID},
				B:  model.Endpoint{DeviceID: router.ID, PortID: router.Ports[0].ID},
			},
			{
				ID: id(),
				A:  model.Endpoint{DeviceID: pp1.ID, PortID: pp1.Ports[1].ID},
				B:  model.Endpoint{DeviceID: doorbell.ID, PortID: doorbell.Ports[0].ID},
			},
			{
				ID: id(),
				A:  model.Endpoint{DeviceID: pp2.ID, PortID: pp2.Ports[0].ID},
				B:  model.Endpoint{DeviceID: phone.ID, PortID: phone.Ports[0].ID},
			},
			{
				ID: id(),
				A:  model.Endpoint{DeviceID: pp3.ID, PortID: pp3.Ports[0].ID},
				B:  model.Endpoint{DeviceID: isdn.ID, PortID: isdn.Ports[0].ID},
			},
		},
	}
}

func sequentialID() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("id_%04d", n)
	}
}
