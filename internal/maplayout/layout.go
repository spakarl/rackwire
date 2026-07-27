package maplayout

import (
	"fmt"
	"math"

	"github.com/spakarl/rackwire/internal/model"
)

const (
	colWidth   = 168.0
	colGap     = 36.0
	midGap     = 140.0
	headerH    = 52.0
	portH      = 28.0
	padX       = 24.0
	padY       = 24.0
	colPadTop  = 8.0
	colPadBot  = 12.0
)

// PortView is one port row inside a device column.
type PortView struct {
	ID        string
	Label     string
	Y         float64 // absolute center Y for SVG anchors
	Top       float64 // absolute top of the row
	RelTop    float64 // top relative to column (for HTML)
	Linked    bool
	PeerLabel string
	Key       string // "deviceId:portId"
}

// Column is one device rendered as a vertical strip.
type Column struct {
	DeviceID string
	Name     string
	Kind     string
	Color    string
	X        float64
	Y        float64
	Width    float64
	Height   float64
	Ports    []PortView
	Side     string // "left" | "right"
}

// Curve is one patch cable as an SVG path.
type Curve struct {
	LinkID  string
	Path    string
	Color   string
	FromKey string
	ToKey   string
	Title   string
}

// Diagram is the full patch map layout.
type Diagram struct {
	Width   float64
	Height  float64
	Left    []Column
	Right   []Column
	Curves  []Curve
	HasLinks bool
}

type anchor struct {
	key   string
	leftX float64
	rightX float64
	y     float64
	color string
	label string
}

// Build computes a left/right column layout and SVG curves for all links.
func Build(rack *model.Rack) Diagram {
	if rack == nil {
		return Diagram{}
	}

	var leftDevs, rightDevs []model.Device
	for _, d := range rack.Devices {
		if d.Kind == "patchpanel" || d.Kind == "router" {
			leftDevs = append(leftDevs, d)
		} else {
			rightDevs = append(rightDevs, d)
		}
	}
	leftDevs = model.SortDevices(leftDevs)
	rightDevs = model.SortDevices(rightDevs)

	left := layoutGroup(rack, leftDevs, padX, "left")
	rightStartX := padX
	if len(left) > 0 {
		rightStartX = left[len(left)-1].X + colWidth + midGap
	} else if len(rightDevs) > 0 {
		rightStartX = padX
	}
	// If only right devices, keep them on the right side of a mid gap for visual balance
	if len(left) == 0 && len(rightDevs) > 0 {
		rightStartX = padX + colWidth + midGap
	}
	right := layoutGroup(rack, rightDevs, rightStartX, "right")

	anchors := map[string]anchor{}
	collectAnchors(anchors, left)
	collectAnchors(anchors, right)

	curves := make([]Curve, 0, len(rack.Links))
	for _, link := range rack.Links {
		aKey := link.A.DeviceID + ":" + link.A.PortID
		bKey := link.B.DeviceID + ":" + link.B.PortID
		a, okA := anchors[aKey]
		b, okB := anchors[bKey]
		if !okA || !okB {
			continue
		}
		x1, y1, x2, y2 := pickEnds(a, b)
		color := a.color
		if color == "" {
			color = b.color
		}
		if color == "" {
			color = "#0f5c4c"
		}
		curves = append(curves, Curve{
			LinkID:  link.ID,
			Path:    bezier(x1, y1, x2, y2),
			Color:   color,
			FromKey: aKey,
			ToKey:   bKey,
			Title:   a.label + " ↔ " + b.label,
		})
	}

	width := padX
	height := padY
	for _, cols := range [][]Column{left, right} {
		for _, c := range cols {
			width = math.Max(width, c.X+c.Width+padX)
			height = math.Max(height, c.Y+c.Height+padY)
		}
	}
	if width < 640 {
		width = 640
	}
	if height < 320 {
		height = 320
	}

	return Diagram{
		Width:    width,
		Height:   height,
		Left:     left,
		Right:    right,
		Curves:   curves,
		HasLinks: len(curves) > 0,
	}
}

func layoutGroup(rack *model.Rack, devices []model.Device, startX float64, side string) []Column {
	cols := make([]Column, 0, len(devices))
	x := startX
	for _, d := range devices {
		ports := make([]PortView, 0, len(d.Ports))
		for i, p := range d.Ports {
			label := p.Label
			if label == "" {
				label = p.ID
			}
			relTop := float64(i) * portH
			top := padY + colPadTop + headerH + relTop
			y := top + portH/2
			key := d.ID + ":" + p.ID
			peer := rack.PeerLabel(d.ID, p.ID)
			ports = append(ports, PortView{
				ID:        p.ID,
				Label:     label,
				Y:         y,
				Top:       top,
				RelTop:    relTop,
				Linked:    peer != "",
				PeerLabel: peer,
				Key:       key,
			})
		}
		h := colPadTop + headerH + float64(len(d.Ports))*portH + colPadBot
		if len(d.Ports) == 0 {
			h = colPadTop + headerH + colPadBot
		}
		cols = append(cols, Column{
			DeviceID: d.ID,
			Name:     d.Name,
			Kind:     d.Kind,
			Color:    d.Color,
			X:        x,
			Y:        padY,
			Width:    colWidth,
			Height:   h,
			Ports:    ports,
			Side:     side,
		})
		x += colWidth + colGap
	}
	return cols
}

func collectAnchors(out map[string]anchor, cols []Column) {
	for _, c := range cols {
		for _, p := range c.Ports {
			out[p.Key] = anchor{
				key:    p.Key,
				leftX:  c.X,
				rightX: c.X + c.Width,
				y:      p.Y,
				color:  c.Color,
				label:  c.Name + " · " + p.Label,
			}
		}
	}
}

func pickEnds(a, b anchor) (x1, y1, x2, y2 float64) {
	// Connect outer edges so cables leave toward the peer side.
	aMid := (a.leftX + a.rightX) / 2
	bMid := (b.leftX + b.rightX) / 2
	if aMid <= bMid {
		return a.rightX, a.y, b.leftX, b.y
	}
	return a.leftX, a.y, b.rightX, b.y
}

func bezier(x1, y1, x2, y2 float64) string {
	// Normalize left→right for stable control points.
	if x1 > x2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	dx := x2 - x1
	c := math.Max(40, dx*0.45)
	return fmt.Sprintf("M %.1f %.1f C %.1f %.1f, %.1f %.1f, %.1f %.1f",
		x1, y1, x1+c, y1, x2-c, y2, x2, y2)
}
