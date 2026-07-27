package wirecolor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Entry is one solid or striped wire color in a palette.
type Entry struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Hex    string `json:"hex,omitempty"`
	Base   string `json:"base,omitempty"`
	Stripe string `json:"stripe,omitempty"`
}

// Palette is a named collection of wire colors (one JSON file).
type Palette struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Colors []Entry `json:"colors"`
}

// Resolved is ready for UI swatches.
type Resolved struct {
	ID       string
	Label    string
	Solid    bool
	Hex      string // solid fill
	BaseHex  string // stripe base
	StripeHex string
	Palette  string
}

// SelectOption is one option in a grouped color dropdown.
type SelectOption struct {
	ID       string
	Label    string
	Group    string
	Resolved Resolved
}

// Catalog loads wire-color palettes from JSON files.
type Catalog struct {
	mu       sync.RWMutex
	dir      string
	palettes map[string]Palette
	byID     map[string]Resolved // flat index across palettes
}

// Open creates a catalog, writing missing seed files.
func Open(dir string) (*Catalog, error) {
	c := &Catalog{dir: dir, palettes: map[string]Palette{}, byID: map[string]Resolved{}}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := c.ensureSeeds(); err != nil {
		return nil, err
	}
	if err := c.reload(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Catalog) Dir() string { return c.dir }

func (c *Catalog) ListPalettes() []Palette {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Palette, 0, len(c.palettes))
	for _, p := range c.palettes {
		out = append(out, clonePalette(p))
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// Options returns dropdown options grouped by palette name.
func (c *Catalog) Options() []SelectOption {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []SelectOption
	pals := make([]Palette, 0, len(c.palettes))
	for _, p := range c.palettes {
		pals = append(pals, p)
	}
	sort.Slice(pals, func(i, j int) bool {
		return strings.ToLower(pals[i].Name) < strings.ToLower(pals[j].Name)
	})
	for _, p := range pals {
		for _, e := range p.Colors {
			r, ok := c.byID[e.ID]
			if !ok {
				continue
			}
			out = append(out, SelectOption{
				ID:       e.ID,
				Label:    e.ID + " — " + e.Label,
				Group:    p.Name,
				Resolved: r,
			})
		}
	}
	return out
}

// Known reports whether code exists in any loaded palette.
func (c *Catalog) Known(code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.byID[code]
	return ok
}

// Resolve looks up a color code. Unknown codes fall back to solid gray with the code as label.
func (c *Catalog) Resolve(code string) Resolved {
	code = strings.TrimSpace(code)
	if code == "" {
		return Resolved{ID: "", Label: "(keine)", Solid: true, Hex: "#888888"}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if r, ok := c.byID[code]; ok {
		return r
	}
	return Resolved{ID: code, Label: code, Solid: true, Hex: "#888888"}
}

// GroupedOptions returns options nested by palette for HTML optgroups.
func (c *Catalog) GroupedOptions() []struct {
	Name    string
	Options []SelectOption
} {
	opts := c.Options()
	var groups []struct {
		Name    string
		Options []SelectOption
	}
	var cur string
	for _, o := range opts {
		if o.Group != cur {
			groups = append(groups, struct {
				Name    string
				Options []SelectOption
			}{Name: o.Group})
			cur = o.Group
		}
		groups[len(groups)-1].Options = append(groups[len(groups)-1].Options, o)
	}
	return groups
}

func (c *Catalog) reload() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	palettes := map[string]Palette{}
	solids := map[string]string{} // id -> hex, filled in first pass

	type pending struct {
		palette string
		entry   Entry
	}
	var striped []pending

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.dir, e.Name()))
		if err != nil {
			return err
		}
		var p Palette
		if err := json.Unmarshal(data, &p); err != nil {
			return fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		if p.ID == "" {
			p.ID = strings.TrimSuffix(e.Name(), ".json")
		}
		if p.Name == "" {
			p.Name = p.ID
		}
		palettes[p.ID] = p
		for _, col := range p.Colors {
			if col.Hex != "" && col.Base == "" {
				solids[col.ID] = col.Hex
			} else if col.Base != "" {
				striped = append(striped, pending{palette: p.Name, entry: col})
			}
		}
	}

	byID := map[string]Resolved{}
	for _, p := range palettes {
		for _, col := range p.Colors {
			if col.Hex != "" && col.Base == "" {
				byID[col.ID] = Resolved{
					ID: col.ID, Label: col.Label, Solid: true, Hex: col.Hex, Palette: p.Name,
				}
			}
		}
	}
	for _, item := range striped {
		col := item.entry
		baseHex := solids[col.Base]
		stripeHex := solids[col.Stripe]
		if baseHex == "" {
			baseHex = "#f5f5f5"
		}
		if stripeHex == "" {
			stripeHex = "#888888"
		}
		byID[col.ID] = Resolved{
			ID: col.ID, Label: col.Label, Solid: false,
			BaseHex: baseHex, StripeHex: stripeHex, Palette: item.palette,
		}
	}

	c.mu.Lock()
	c.palettes = palettes
	c.byID = byID
	c.mu.Unlock()
	return nil
}

func (c *Catalog) ensureSeeds() error {
	for _, p := range seeds() {
		path := filepath.Join(c.dir, p.ID+".json")
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		data, err := json.MarshalIndent(p, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func seeds() []Palette {
	return []Palette{{
		ID:   "iec60757",
		Name: "Netzwerk / IEC 60757",
		Colors: []Entry{
			{ID: "WH", Label: "White", Hex: "#f5f5f5"},
			{ID: "BN", Label: "Brown", Hex: "#8b4513"},
			{ID: "GN", Label: "Green", Hex: "#27ae60"},
			{ID: "YE", Label: "Yellow", Hex: "#f1c40f"},
			{ID: "GY", Label: "Grey", Hex: "#95a5a6"},
			{ID: "PK", Label: "Pink", Hex: "#e91e8c"},
			{ID: "BU", Label: "Blue", Hex: "#3498db"},
			{ID: "RD", Label: "Red", Hex: "#c0392b"},
			{ID: "BK", Label: "Black", Hex: "#2c3e50"},
			{ID: "VT", Label: "Violet", Hex: "#8e44ad"},
			{ID: "OR", Label: "Orange", Hex: "#e67e22"},
			{ID: "TQ", Label: "Turquoise", Hex: "#1abc9c"},
			// T568 striped pairs
			{ID: "WH/OR", Label: "White/Orange", Base: "WH", Stripe: "OR"},
			{ID: "WH/GN", Label: "White/Green", Base: "WH", Stripe: "GN"},
			{ID: "WH/BU", Label: "White/Blue", Base: "WH", Stripe: "BU"},
			{ID: "WH/BN", Label: "White/Brown", Base: "WH", Stripe: "BN"},
		},
	}}
}

func clonePalette(p Palette) Palette {
	cp := p
	if p.Colors != nil {
		cp.Colors = make([]Entry, len(p.Colors))
		copy(cp.Colors, p.Colors)
	}
	return cp
}
