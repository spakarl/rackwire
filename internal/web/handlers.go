package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/spakarl/rackwire/internal/maplayout"
	"github.com/spakarl/rackwire/internal/model"
	"github.com/spakarl/rackwire/internal/porttpl"
	"github.com/spakarl/rackwire/internal/store"
	"github.com/spakarl/rackwire/internal/wirecolor"
)

type Server struct {
	store  *store.Store
	cat    *porttpl.Catalog
	colors *wirecolor.Catalog
	tmpl   *template.Template
	static fs.FS
}

func New(st *store.Store, cat *porttpl.Catalog, colors *wirecolor.Catalog, uiFS fs.FS) (*Server, error) {
	s := &Server{store: st, cat: cat, colors: colors}
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"kindLabel":  kindLabel,
		"lower":      strings.ToLower,
		"add":        func(a, b int) int { return a + b },
		"wireSwatch": func(code string) wirecolor.Resolved { return colors.Resolve(code) },
		"colorKnown": func(code string) bool { return colors.Known(code) },
	}).ParseFS(uiFS, "layouts/*.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(uiFS, "static")
	if err != nil {
		return nil, err
	}
	s.tmpl = tmpl
	s.static = static
	return s, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(s.static)))
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /map", s.mapPage)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/health", s.health)

	mux.HandleFunc("POST /devices", s.createDevice)
	mux.HandleFunc("GET /devices/{id}", s.device)
	mux.HandleFunc("POST /devices/{id}", s.updateDevice)
	mux.HandleFunc("POST /devices/{id}/delete", s.deleteDevice)
	mux.HandleFunc("GET /devices/{id}/ports/{portId}", s.port)
	mux.HandleFunc("GET /devices/{id}/ports/{portId}/preview", s.previewPort)
	mux.HandleFunc("POST /devices/{id}/ports/{portId}", s.updatePort)
	mux.HandleFunc("POST /devices/{id}/ports", s.addPorts)

	mux.HandleFunc("POST /links", s.createLink)
	mux.HandleFunc("POST /links/{id}/delete", s.deleteLink)

	mux.HandleFunc("GET /templates", s.templatesList)
	mux.HandleFunc("POST /templates", s.templatesCreate)
	mux.HandleFunc("GET /templates/{id}", s.templateEdit)
	mux.HandleFunc("POST /templates/{id}", s.templateSave)
	mux.HandleFunc("POST /templates/{id}/delete", s.templateDelete)
	mux.HandleFunc("GET /colors", s.colorsPage)
	mux.HandleFunc("GET /rooms", s.roomsList)
	mux.HandleFunc("POST /rooms", s.roomsCreate)
	mux.HandleFunc("POST /rooms/{id}", s.roomsUpdate)
	mux.HandleFunc("POST /rooms/{id}/delete", s.roomsDelete)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

type pageData struct {
	Title        string
	Rack         *model.Rack
	Device       *model.Device
	Port         *model.Port
	Templates    []model.PortTemplate
	Template     *model.PortTemplate
	Map          maplayout.Diagram
	ColorGroups  []struct {
		Name    string
		Options []wirecolor.SelectOption
	}
	ColorPalettes []wirecolor.Palette
	Peer          string
	Flash         string
	Error         string
	IsSeed        bool
	DefaultID     string
	PortPrefix    string
	PortMapJSON   template.JS
}

func (s *Server) withColors(d pageData) pageData {
	d.ColorGroups = s.colors.GroupedOptions()
	d.ColorPalettes = s.colors.ListPalettes()
	return d
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	rack := s.store.Get()
	s.render(w, "home.html", pageData{
		Title:       rack.Name,
		Rack:        &rack,
		Templates:   s.cat.List(),
		DefaultID:   porttpl.DefaultID,
		PortMapJSON: portMapJSON(&rack),
	})
}

func portMapJSON(rack *model.Rack) template.JS {
	type portEntry struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	m := map[string][]portEntry{}
	for _, d := range rack.Devices {
		ports := make([]portEntry, 0, len(d.Ports))
		for _, p := range d.Ports {
			ports = append(ports, portEntry{ID: p.ID, Label: p.Label})
		}
		m[d.ID] = ports
	}
	b, err := json.Marshal(m)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b)
}

func (s *Server) mapPage(w http.ResponseWriter, r *http.Request) {
	rack := s.store.Get()
	s.render(w, "map.html", pageData{
		Title: "Verbindungskarte",
		Rack:  &rack,
		Map:   maplayout.Build(&rack),
	})
}

func (s *Server) createDevice(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	kind := r.FormValue("kind")
	color := r.FormValue("color")
	tpl := r.FormValue("templateId")
	count, _ := strconv.Atoi(r.FormValue("portCount"))
	position, _ := strconv.Atoi(r.FormValue("position"))
	if position < 0 {
		position = 0
	}
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if kind == "" {
		kind = "other"
	}
	if color == "" {
		color = "#555555"
	}
	if tpl == "" {
		tpl = porttpl.DefaultID
	}
	if count < 1 {
		count = 1
	}
	if count > 48 {
		count = 48
	}
	prefix := strings.TrimSpace(r.FormValue("portPrefix"))
	if prefix == "" {
		prefix = defaultPortPrefix(kind)
	}
	roomID := strings.TrimSpace(r.FormValue("roomId"))

	err := s.store.Update(func(rack *model.Rack) error {
		if roomID != "" && rack.RoomByID(roomID) == nil {
			return fmt.Errorf("room not found")
		}
		dev := model.Device{
			ID:       newID(),
			Name:     name,
			Kind:     kind,
			Color:    color,
			Position: position,
			RoomID:   roomID,
			Ports:    s.cat.NewPorts(count, tpl, prefix, newID),
		}
		rack.Devices = append(rack.Devices, dev)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) device(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rack := s.store.Get()
	dev := rack.DeviceByID(id)
	if dev == nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, "device.html", pageData{
		Title:      dev.Name,
		Rack:       &rack,
		Device:     dev,
		Templates:  s.cat.List(),
		DefaultID:  porttpl.DefaultID,
		PortPrefix: inferPortPrefix(dev.Ports),
	})
}

func (s *Server) updateDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	color := r.FormValue("color")
	position, _ := strconv.Atoi(r.FormValue("position"))
	roomID := strings.TrimSpace(r.FormValue("roomId"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if color == "" {
		color = "#555555"
	}
	if position < 0 {
		position = 0
	}
	err := s.store.Update(func(rack *model.Rack) error {
		dev := rack.DeviceByID(id)
		if dev == nil {
			return fmt.Errorf("device not found")
		}
		if roomID != "" && rack.RoomByID(roomID) == nil {
			return fmt.Errorf("room not found")
		}
		dev.Name = name
		dev.Color = color
		dev.Position = position
		dev.RoomID = roomID
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/devices/"+id, http.StatusSeeOther)
}

func (s *Server) deleteDevice(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.store.Update(func(rack *model.Rack) error {
		out := rack.Devices[:0]
		for _, d := range rack.Devices {
			if d.ID != id {
				out = append(out, d)
			}
		}
		rack.Devices = out
		links := rack.Links[:0]
		for _, l := range rack.Links {
			if l.A.DeviceID != id && l.B.DeviceID != id {
				links = append(links, l)
			}
		}
		rack.Links = links
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) port(w http.ResponseWriter, r *http.Request) {
	devID := r.PathValue("id")
	portID := r.PathValue("portId")
	rack := s.store.Get()
	dev := rack.DeviceByID(devID)
	if dev == nil {
		http.NotFound(w, r)
		return
	}
	port := dev.PortByID(portID)
	if port == nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, "port.html", s.withColors(pageData{
		Title:     port.Label,
		Rack:      &rack,
		Device:    dev,
		Port:      port,
		Templates: s.cat.List(),
		Peer:      rack.PeerLabel(devID, portID),
	}))
}

// previewPort applies a template in-memory for the editor UI without persisting.
func (s *Server) previewPort(w http.ResponseWriter, r *http.Request) {
	devID := r.PathValue("id")
	portID := r.PathValue("portId")
	_ = r.ParseForm()
	tplID := strings.TrimSpace(r.FormValue("templateId"))

	rack := s.store.Get()
	dev := rack.DeviceByID(devID)
	if dev == nil {
		http.NotFound(w, r)
		return
	}
	port := dev.PortByID(portID)
	if port == nil {
		http.NotFound(w, r)
		return
	}

	preview := *port
	if label := strings.TrimSpace(r.FormValue("label")); label != "" {
		preview.Label = label
	}
	if tplID != "" {
		s.cat.ApplyTemplate(&preview, tplID)
	}

	s.render(w, "port_editor.html", s.withColors(pageData{
		Rack:      &rack,
		Device:    dev,
		Port:      &preview,
		Templates: s.cat.List(),
		Peer:      rack.PeerLabel(devID, portID),
		Flash:     "Vorschau — noch nicht gespeichert",
	}))
}

func (s *Server) updatePort(w http.ResponseWriter, r *http.Request) {
	devID := r.PathValue("id")
	portID := r.PathValue("portId")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	intent := r.FormValue("intent")
	if intent == "" {
		intent = "save"
	}

	var flash string
	err := s.store.Update(func(rack *model.Rack) error {
		dev := rack.DeviceByID(devID)
		if dev == nil {
			return fmt.Errorf("device not found")
		}
		port := dev.PortByID(portID)
		if port == nil {
			return fmt.Errorf("port not found")
		}

		port.Label = strings.TrimSpace(r.FormValue("label"))
		newTpl := r.FormValue("templateId")

		if strings.HasPrefix(intent, "remove_pin_") {
			idx, _ := strconv.Atoi(strings.TrimPrefix(intent, "remove_pin_"))
			pins := s.parsePinsFromForm(r)
			if idx >= 0 && idx < len(pins) {
				pins = append(pins[:idx], pins[idx+1:]...)
			}
			port.Pins = pins
			if newTpl != "" {
				port.TemplateID = newTpl
			}
			flash = "Pin entfernt"
			return nil
		}

		switch intent {
		case "add_pin":
			pins := s.parsePinsFromForm(r)
			next := 1
			for _, p := range pins {
				if p.Number >= next {
					next = p.Number + 1
				}
			}
			pins = append(pins, model.Pin{Number: next, ColorHex: "#888888"})
			port.Pins = pins
			if newTpl != "" {
				port.TemplateID = newTpl
			}
			flash = "Pin hinzugefügt"
		case "remove_pin":
			pins := s.parsePinsFromForm(r)
			if len(pins) > 0 {
				pins = pins[:len(pins)-1]
			}
			port.Pins = pins
			if newTpl != "" {
				port.TemplateID = newTpl
			}
			flash = "Pin entfernt"
		case "save_template":
			pins := s.parsePinsFromForm(r)
			port.Pins = pins
			if newTpl != "" {
				port.TemplateID = newTpl
			}
			name := strings.TrimSpace(r.FormValue("templateName"))
			if name == "" {
				return fmt.Errorf("template name required")
			}
			t := model.PortTemplate{Name: name, Pins: pins}
			t.ID = porttpl.Slug(name)
			if t.ID == "" {
				return fmt.Errorf("invalid template name")
			}
			if err := s.cat.Save(t); err != nil {
				return err
			}
			port.TemplateID = t.ID
			flash = "Template gespeichert"
		default: // save
			port.Pins = s.parsePinsFromForm(r)
			if newTpl != "" {
				port.TemplateID = newTpl
			}
			flash = "Gespeichert"
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	rack := s.store.Get()
	dev := rack.DeviceByID(devID)
	port := dev.PortByID(portID)
	if r.Header.Get("HX-Request") == "true" {
		s.render(w, "port_editor.html", s.withColors(pageData{
			Rack: &rack, Device: dev, Port: port,
			Templates: s.cat.List(),
			Peer:      rack.PeerLabel(devID, portID),
			Flash:     flash,
		}))
		return
	}
	http.Redirect(w, r, "/devices/"+devID+"/ports/"+portID, http.StatusSeeOther)
}

func (s *Server) parsePinsFromForm(r *http.Request) []model.Pin {
	count, _ := strconv.Atoi(r.FormValue("pinCount"))
	if count < 0 {
		count = 0
	}
	if count > 64 {
		count = 64
	}
	pins := make([]model.Pin, 0, count)
	for i := 0; i < count; i++ {
		prefix := fmt.Sprintf("pin_%d_", i)
		num, _ := strconv.Atoi(r.FormValue(prefix + "number"))
		color := r.FormValue(prefix + "color")
		hex := ""
		if s.colors != nil {
			res := s.colors.Resolve(color)
			if res.Solid {
				hex = res.Hex
			} else {
				hex = res.BaseHex
			}
		}
		if hex == "" {
			hex = "#888888"
		}
		pins = append(pins, model.Pin{
			Number:   num,
			Signal:   r.FormValue(prefix + "signal"),
			Color:    color,
			ColorHex: hex,
		})
	}
	return pins
}

func (s *Server) addPorts(w http.ResponseWriter, r *http.Request) {
	devID := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	count, _ := strconv.Atoi(r.FormValue("count"))
	tpl := r.FormValue("templateId")
	if count < 1 {
		count = 1
	}
	if tpl == "" {
		tpl = porttpl.DefaultID
	}
	err := s.store.Update(func(rack *model.Rack) error {
		dev := rack.DeviceByID(devID)
		if dev == nil {
			return fmt.Errorf("device not found")
		}
		start := len(dev.Ports) + 1
		prefix := strings.TrimSpace(r.FormValue("portPrefix"))
		if prefix == "" {
			prefix = inferPortPrefix(dev.Ports)
		}
		added := s.cat.NewPorts(count, tpl, prefix, newID)
		for i := range added {
			added[i].Label = prefix + fmt.Sprintf("%02d", start+i)
		}
		dev.Ports = append(dev.Ports, added...)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/devices/"+devID, http.StatusSeeOther)
}

func (s *Server) createLink(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	aDev := r.FormValue("aDeviceId")
	aPort := r.FormValue("aPortId")
	bDev := r.FormValue("bDeviceId")
	bPort := r.FormValue("bPortId")
	if aDev == "" || aPort == "" || bDev == "" || bPort == "" {
		http.Error(w, "both endpoints required", 400)
		return
	}
	if aDev == bDev && aPort == bPort {
		http.Error(w, "cannot link a port to itself", 400)
		return
	}

	err := s.store.Update(func(rack *model.Rack) error {
		if rack.DeviceByID(aDev) == nil || rack.DeviceByID(bDev) == nil {
			return fmt.Errorf("device not found")
		}
		if rack.DeviceByID(aDev).PortByID(aPort) == nil || rack.DeviceByID(bDev).PortByID(bPort) == nil {
			return fmt.Errorf("port not found")
		}
		links := rack.Links[:0]
		for _, l := range rack.Links {
			touchA := (l.A.DeviceID == aDev && l.A.PortID == aPort) || (l.B.DeviceID == aDev && l.B.PortID == aPort)
			touchB := (l.A.DeviceID == bDev && l.A.PortID == bPort) || (l.B.DeviceID == bDev && l.B.PortID == bPort)
			if !touchA && !touchB {
				links = append(links, l)
			}
		}
		rack.Links = append(links, model.Link{
			ID: newID(),
			A:  model.Endpoint{DeviceID: aDev, PortID: aPort},
			B:  model.Endpoint{DeviceID: bDev, PortID: bPort},
		})
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) deleteLink(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.store.Update(func(rack *model.Rack) error {
		out := rack.Links[:0]
		for _, l := range rack.Links {
			if l.ID != id {
				out = append(out, l)
			}
		}
		rack.Links = out
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) templatesList(w http.ResponseWriter, r *http.Request) {
	s.render(w, "templates.html", pageData{
		Title:     "Templates",
		Templates: s.cat.List(),
		DefaultID: porttpl.DefaultID,
	})
}

func (s *Server) templatesCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	base := r.FormValue("fromId")
	var pins []model.Pin
	if base != "" {
		if t := s.cat.ByID(base); t != nil {
			pins = append([]model.Pin(nil), t.Pins...)
		}
	}
	t := model.PortTemplate{Name: name, Pins: pins}
	t.ID = porttpl.Slug(name)
	if t.ID == "" {
		http.Error(w, "invalid name", 400)
		return
	}
	if existing := s.cat.ByID(t.ID); existing != nil {
		t.ID = t.ID + "-" + newID()[3:7]
	}
	if err := s.cat.Save(t); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/templates/"+t.ID, http.StatusSeeOther)
}

func (s *Server) templateEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t := s.cat.ByID(id)
	if t == nil {
		http.NotFound(w, r)
		return
	}
	s.render(w, "template_edit.html", s.withColors(pageData{
		Title:    t.Name,
		Template: t,
		IsSeed:   s.cat.IsSeed(id),
	}))
}

func (s *Server) templateSave(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	existing := s.cat.ByID(id)
	if existing == nil {
		http.NotFound(w, r)
		return
	}
	intent := r.FormValue("intent")
	pins := s.parsePinsFromForm(r)
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = existing.Name
	}

	if strings.HasPrefix(intent, "remove_pin_") {
		idx, _ := strconv.Atoi(strings.TrimPrefix(intent, "remove_pin_"))
		if idx >= 0 && idx < len(pins) {
			pins = append(pins[:idx], pins[idx+1:]...)
		}
	} else {
		switch intent {
		case "add_pin":
			next := 1
			for _, p := range pins {
				if p.Number >= next {
					next = p.Number + 1
				}
			}
			pins = append(pins, model.Pin{Number: next, ColorHex: "#888888"})
		case "remove_pin":
			if len(pins) > 0 {
				pins = pins[:len(pins)-1]
			}
		}
	}

	t := model.PortTemplate{ID: id, Name: name, Pins: pins}
	if err := s.cat.Save(t); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/templates/"+id, http.StatusSeeOther)
}

func (s *Server) templateDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.cat.Delete(id); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/templates", http.StatusSeeOther)
}

func (s *Server) colorsPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, "colors.html", pageData{
		Title:         "Aderfarben",
		ColorPalettes: s.colors.ListPalettes(),
	})
}

func (s *Server) roomsList(w http.ResponseWriter, r *http.Request) {
	rack := s.store.Get()
	s.render(w, "rooms.html", pageData{
		Title: "Räume",
		Rack:  &rack,
	})
}

func (s *Server) roomsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	id := porttpl.Slug(name)
	if id == "" {
		http.Error(w, "invalid name", 400)
		return
	}
	err := s.store.Update(func(rack *model.Rack) error {
		if rack.RoomByID(id) != nil {
			id = id + "-" + newID()[3:7]
		}
		rack.Rooms = append(rack.Rooms, model.Room{ID: id, Name: name})
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/rooms", http.StatusSeeOther)
}

func (s *Server) roomsUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "name required", 400)
		return
	}
	err := s.store.Update(func(rack *model.Rack) error {
		rm := rack.RoomByID(id)
		if rm == nil {
			return fmt.Errorf("room not found")
		}
		rm.Name = name
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/rooms", http.StatusSeeOther)
}

func (s *Server) roomsDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.store.Update(func(rack *model.Rack) error {
		if rack.RoomByID(id) == nil {
			return fmt.Errorf("room not found")
		}
		out := rack.Rooms[:0]
		for _, rm := range rack.Rooms {
			if rm.ID != id {
				out = append(out, rm)
			}
		}
		rack.Rooms = out
		rack.ClearRoomID(id)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/rooms", http.StatusSeeOther)
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "id_" + hex.EncodeToString(b[:])
}

func defaultPortPrefix(kind string) string {
	switch kind {
	case "patchpanel":
		return "P-"
	case "outlet":
		return "O-"
	case "router":
		return "R-"
	default:
		if kind == "" {
			return "X-"
		}
		return strings.ToUpper(kind[:1]) + "-"
	}
}

// inferPortPrefix strips trailing digits from the first port label (e.g. "LAN-03" → "LAN-").
func inferPortPrefix(ports []model.Port) string {
	if len(ports) == 0 {
		return "P-"
	}
	label := ports[0].Label
	i := len(label)
	for i > 0 && label[i-1] >= '0' && label[i-1] <= '9' {
		i--
	}
	if i == 0 {
		return "P-"
	}
	return label[:i]
}

func kindLabel(kind string) string {
	switch kind {
	case "patchpanel":
		return "Patchpanel"
	case "router":
		return "Router"
	case "outlet":
		return "Dose"
	default:
		return "Gerät"
	}
}
