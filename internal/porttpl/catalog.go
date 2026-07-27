package porttpl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/spakarl/rackwire/internal/model"
)

// DefaultID is applied when creating devices/ports without an explicit template.
const DefaultID = "rj45-t568a"

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Catalog loads and persists port templates as JSON files.
type Catalog struct {
	mu   sync.RWMutex
	dir  string
	byID map[string]model.PortTemplate
}

// Open creates a catalog at dir, writing missing seed templates.
func Open(dir string) (*Catalog, error) {
	c := &Catalog{dir: dir, byID: map[string]model.PortTemplate{}}
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

func (c *Catalog) List() []model.PortTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]model.PortTemplate, 0, len(c.byID))
	for _, t := range c.byID {
		out = append(out, cloneTemplate(t))
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (c *Catalog) ByID(id string) *model.PortTemplate {
	c.mu.RLock()
	defer c.mu.RUnlock()
	t, ok := c.byID[id]
	if !ok {
		return nil
	}
	cp := cloneTemplate(t)
	return &cp
}

func (c *Catalog) ApplyTemplate(port *model.Port, templateID string) {
	t := c.ByID(templateID)
	if t == nil {
		return
	}
	port.TemplateID = t.ID
	pins := make([]model.Pin, len(t.Pins))
	copy(pins, t.Pins)
	port.Pins = pins
}

// NewPorts creates n ports from a template with sequential labels.
func (c *Catalog) NewPorts(count int, templateID, labelPrefix string, idFn func() string) []model.Port {
	if templateID == "" {
		templateID = DefaultID
	}
	ports := make([]model.Port, 0, count)
	for i := 1; i <= count; i++ {
		p := model.Port{
			ID:    idFn(),
			Label: labelPrefix + pad3(i),
		}
		c.ApplyTemplate(&p, templateID)
		ports = append(ports, p)
	}
	return ports
}

func (c *Catalog) Save(t model.PortTemplate) error {
	t.ID = strings.TrimSpace(t.ID)
	t.Name = strings.TrimSpace(t.Name)
	if t.ID == "" {
		t.ID = Slug(t.Name)
	}
	if t.ID == "" {
		return fmt.Errorf("template id required")
	}
	if t.Name == "" {
		t.Name = t.ID
	}
	if err := c.writeFile(t); err != nil {
		return err
	}
	c.mu.Lock()
	c.byID[t.ID] = cloneTemplate(t)
	c.mu.Unlock()
	return nil
}

func (c *Catalog) Delete(id string) error {
	if isSeedID(id) {
		return fmt.Errorf("cannot delete built-in template %q", id)
	}
	path := filepath.Join(c.dir, id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	c.mu.Lock()
	delete(c.byID, id)
	c.mu.Unlock()
	return nil
}

func (c *Catalog) IsSeed(id string) bool { return isSeedID(id) }

func (c *Catalog) reload() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return err
	}
	next := map[string]model.PortTemplate{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.dir, e.Name()))
		if err != nil {
			return err
		}
		var t model.PortTemplate
		if err := json.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		if t.ID == "" {
			t.ID = strings.TrimSuffix(e.Name(), ".json")
		}
		if t.Name == "" {
			t.Name = t.ID
		}
		next[t.ID] = t
	}
	c.mu.Lock()
	c.byID = next
	c.mu.Unlock()
	return nil
}

func (c *Catalog) writeFile(t model.PortTemplate) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, t.ID+".json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (c *Catalog) ensureSeeds() error {
	for _, t := range seeds() {
		path := filepath.Join(c.dir, t.ID+".json")
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := c.writeFile(t); err != nil {
			return err
		}
	}
	return nil
}

func seeds() []model.PortTemplate {
	return []model.PortTemplate{
		{
			ID:   "rj45-t568a",
			Name: "Netzwerk T568A",
			Pins: []model.Pin{
				{Number: 1, Signal: "TX+", Color: "WH/GN", ColorHex: "#eafaf1"},
				{Number: 2, Signal: "TX-", Color: "GN", ColorHex: "#27ae60"},
				{Number: 3, Signal: "RX+", Color: "WH/OR", ColorHex: "#f5f0e6"},
				{Number: 4, Signal: "", Color: "BU", ColorHex: "#3498db"},
				{Number: 5, Signal: "", Color: "WH/BU", ColorHex: "#ebf5fb"},
				{Number: 6, Signal: "RX-", Color: "OR", ColorHex: "#e67e22"},
				{Number: 7, Signal: "", Color: "WH/BN", ColorHex: "#f5e6d3"},
				{Number: 8, Signal: "", Color: "BN", ColorHex: "#8b4513"},
			},
		},
		{
			ID:   "rj45-t568b",
			Name: "Netzwerk T568B",
			Pins: []model.Pin{
				{Number: 1, Signal: "TX+", Color: "WH/OR", ColorHex: "#f5f0e6"},
				{Number: 2, Signal: "TX-", Color: "OR", ColorHex: "#e67e22"},
				{Number: 3, Signal: "RX+", Color: "WH/GN", ColorHex: "#eafaf1"},
				{Number: 4, Signal: "", Color: "BU", ColorHex: "#3498db"},
				{Number: 5, Signal: "", Color: "WH/BU", ColorHex: "#ebf5fb"},
				{Number: 6, Signal: "RX-", Color: "GN", ColorHex: "#27ae60"},
				{Number: 7, Signal: "", Color: "WH/BN", ColorHex: "#f5e6d3"},
				{Number: 8, Signal: "", Color: "BN", ColorHex: "#8b4513"},
			},
		},
		{
			ID:   "isdn-4pin",
			Name: "ISDN (4 Pins)",
			Pins: []model.Pin{
				{Number: 3, Signal: "TX+", Color: "WH/OR", ColorHex: "#f5f0e6"},
				{Number: 4, Signal: "RX+", Color: "BU", ColorHex: "#3498db"},
				{Number: 5, Signal: "RX-", Color: "WH/BU", ColorHex: "#ebf5fb"},
				{Number: 6, Signal: "TX-", Color: "OR", ColorHex: "#e67e22"},
			},
		},
		{
			ID:   "doorbell-2pin",
			Name: "Klingel (2 Pins)",
			Pins: []model.Pin{
				{Number: 1, Signal: "Bell+", Color: "RD", ColorHex: "#c0392b"},
				{Number: 2, Signal: "Bell-", Color: "BK", ColorHex: "#2c3e50"},
			},
		},
		{
			ID:   "analog-phone-2",
			Name: "Analog Telefon (2 Pins)",
			Pins: []model.Pin{
				{Number: 1, Signal: "Tip", Color: "GN", ColorHex: "#27ae60"},
				{Number: 2, Signal: "Ring", Color: "RD", ColorHex: "#c0392b"},
			},
		},
		{
			ID:   "blank",
			Name: "Leer (keine Pins)",
			Pins: nil,
		},
	}
}

func isSeedID(id string) bool {
	for _, t := range seeds() {
		if t.ID == id {
			return true
		}
	}
	return false
}

// Slug turns a display name into a filesystem-safe template id.
func Slug(name string) string {
	s := strings.Map(func(r rune) rune {
		r = unicode.ToLower(r)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, strings.TrimSpace(name))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 48 {
		s = s[:48]
		s = strings.Trim(s, "-")
	}
	return s
}

func cloneTemplate(t model.PortTemplate) model.PortTemplate {
	cp := t
	if t.Pins != nil {
		cp.Pins = make([]model.Pin, len(t.Pins))
		copy(cp.Pins, t.Pins)
	}
	return cp
}

func pad3(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	if n < 100 {
		return itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
