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
	rack model.Rack
}

func New(path string) (*Store, error) {
	s := &Store{path: path}
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
		s.rack = seedRack()
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

func seedRack() model.Rack {
	id := sequentialID()
	ppTpl := "rj45-t568b"

	pp1 := model.Device{
		ID: id(), Name: "Patchpanel A", Kind: "patchpanel", Color: "#2980b9",
		Ports: porttpl.NewPorts(24, ppTpl, "A-", id),
	}
	pp2 := model.Device{
		ID: id(), Name: "Patchpanel B", Kind: "patchpanel", Color: "#16a085",
		Ports: porttpl.NewPorts(24, ppTpl, "B-", id),
	}
	pp3 := model.Device{
		ID: id(), Name: "Patchpanel C", Kind: "patchpanel", Color: "#8e44ad",
		Ports: porttpl.NewPorts(24, ppTpl, "C-", id),
	}
	router := model.Device{
		ID: id(), Name: "Router", Kind: "router", Color: "#c0392b",
		Ports: porttpl.NewPorts(8, ppTpl, "LAN-", id),
	}
	doorbell := model.Device{
		ID: id(), Name: "Dose Eingang (Klingel)", Kind: "outlet", Color: "#d35400",
		Ports: porttpl.NewPorts(1, "doorbell-2pin", "KLG-", id),
	}
	phone := model.Device{
		ID: id(), Name: "Dose Analog Telefon", Kind: "outlet", Color: "#27ae60",
		Ports: porttpl.NewPorts(1, "analog-phone-2", "TEL-", id),
	}

	return model.Rack{
		Name:    "Homelab Rack",
		Devices: []model.Device{pp1, pp2, pp3, router, doorbell, phone},
		Links:   []model.Link{},
	}
}

func sequentialID() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("id_%04d", n)
	}
}
