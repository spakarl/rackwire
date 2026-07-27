package porttpl

import "github.com/spakarl/rackwire/internal/model"

// Builtin returns the stock port templates.
func Builtin() []model.PortTemplate {
	return []model.PortTemplate{
		{
			ID:   "rj45-t568b",
			Name: "RJ45 T568B",
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
			ID:   "rj45-t568a",
			Name: "RJ45 T568A",
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
			ID:   "doorbell-2pin",
			Name: "Doorbell (2 pins)",
			Pins: []model.Pin{
				{Number: 1, Signal: "Bell+", Color: "RD", ColorHex: "#c0392b"},
				{Number: 2, Signal: "Bell-", Color: "BK", ColorHex: "#2c3e50"},
			},
		},
		{
			ID:   "analog-phone-2",
			Name: "Analog phone (2 pins)",
			Pins: []model.Pin{
				{Number: 1, Signal: "Tip", Color: "GN", ColorHex: "#27ae60"},
				{Number: 2, Signal: "Ring", Color: "RD", ColorHex: "#c0392b"},
			},
		},
		{
			ID:   "blank",
			Name: "Blank (no pins)",
			Pins: nil,
		},
	}
}

func ByID(id string) *model.PortTemplate {
	for _, t := range Builtin() {
		if t.ID == id {
			cp := t
			return &cp
		}
	}
	return nil
}

// ApplyTemplate copies template pins onto the port (replaces pin list).
func ApplyTemplate(port *model.Port, templateID string) {
	t := ByID(templateID)
	if t == nil {
		return
	}
	port.TemplateID = t.ID
	pins := make([]model.Pin, len(t.Pins))
	copy(pins, t.Pins)
	port.Pins = pins
}

// NewPorts creates n ports from a template with sequential labels.
func NewPorts(count int, templateID, labelPrefix string, idFn func() string) []model.Port {
	ports := make([]model.Port, 0, count)
	for i := 1; i <= count; i++ {
		p := model.Port{
			ID:    idFn(),
			Label: labelPrefix + pad3(i),
		}
		ApplyTemplate(&p, templateID)
		ports = append(ports, p)
	}
	return ports
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
