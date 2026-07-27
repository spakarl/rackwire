package web

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/spakarl/rackwire/internal/model"
	"github.com/spakarl/rackwire/internal/porttpl"
	"github.com/spakarl/rackwire/internal/store"
)

type Server struct {
	store *store.Store
	tmpl  *template.Template
	static fs.FS
}

func New(st *store.Store, uiFS fs.FS) (*Server, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"kindLabel": kindLabel,
		"lower":     strings.ToLower,
	}).ParseFS(uiFS, "layouts/*.html", "partials/*.html")
	if err != nil {
		return nil, err
	}
	static, err := fs.Sub(uiFS, "static")
	if err != nil {
		return nil, err
	}
	return &Server{store: st, tmpl: tmpl, static: static}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(s.static)))
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /devices", s.createDevice)
	mux.HandleFunc("GET /devices/{id}", s.device)
	mux.HandleFunc("POST /devices/{id}/delete", s.deleteDevice)
	mux.HandleFunc("GET /devices/{id}/ports/{portId}", s.port)
	mux.HandleFunc("POST /devices/{id}/ports/{portId}", s.updatePort)
	mux.HandleFunc("POST /devices/{id}/ports", s.addPorts)
	mux.HandleFunc("POST /links", s.createLink)
	mux.HandleFunc("POST /links/{id}/delete", s.deleteLink)
	mux.HandleFunc("GET /api/health", s.health)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

type pageData struct {
	Title     string
	Rack      *model.Rack
	Device    *model.Device
	Port      *model.Port
	Templates []model.PortTemplate
	Peer      string
	Flash     string
	Error     string
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	rack := s.store.Get()
	s.render(w, "home.html", pageData{Title: rack.Name, Rack: &rack, Templates: porttpl.Builtin()})
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
		tpl = "rj45-t568b"
	}
	if count < 1 {
		count = 1
	}
	if count > 48 {
		count = 48
	}
	prefix := strings.ToUpper(kind[:1]) + "-"
	if kind == "patchpanel" {
		prefix = "P-"
	}
	if kind == "outlet" {
		prefix = "O-"
	}
	if kind == "router" {
		prefix = "R-"
	}

	err := s.store.Update(func(rack *model.Rack) error {
		dev := model.Device{
			ID:    newID(),
			Name:  name,
			Kind:  kind,
			Color: color,
			Ports: porttpl.NewPorts(count, tpl, prefix, newID),
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
		Title:     dev.Name,
		Rack:      &rack,
		Device:    dev,
		Templates: porttpl.Builtin(),
	})
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
	s.render(w, "port.html", pageData{
		Title:     port.Label,
		Rack:      &rack,
		Device:    dev,
		Port:      port,
		Templates: porttpl.Builtin(),
		Peer:      rack.PeerLabel(devID, portID),
	})
}

func (s *Server) updatePort(w http.ResponseWriter, r *http.Request) {
	devID := r.PathValue("id")
	portID := r.PathValue("portId")
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

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
		if newTpl != "" && newTpl != port.TemplateID {
			porttpl.ApplyTemplate(port, newTpl)
		}

		// Pin overrides from form: pin_N_signal, pin_N_color, pin_N_hex
		for i := range port.Pins {
			n := strconv.Itoa(port.Pins[i].Number)
			if v := r.FormValue("pin_" + n + "_signal"); r.Form.Has("pin_" + n + "_signal") {
				port.Pins[i].Signal = v
			}
			if v := r.FormValue("pin_" + n + "_color"); r.Form.Has("pin_" + n + "_color") {
				port.Pins[i].Color = v
			}
			if v := r.FormValue("pin_" + n + "_hex"); r.Form.Has("pin_" + n + "_hex") {
				port.Pins[i].ColorHex = v
			}
		}
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		rack := s.store.Get()
		dev := rack.DeviceByID(devID)
		port := dev.PortByID(portID)
		s.render(w, "port_editor.html", pageData{
			Rack: &rack, Device: dev, Port: port,
			Templates: porttpl.Builtin(),
			Peer:      rack.PeerLabel(devID, portID),
			Flash:     "Saved",
		})
		return
	}
	http.Redirect(w, r, "/devices/"+devID+"/ports/"+portID, http.StatusSeeOther)
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
		tpl = "rj45-t568b"
	}
	err := s.store.Update(func(rack *model.Rack) error {
		dev := rack.DeviceByID(devID)
		if dev == nil {
			return fmt.Errorf("device not found")
		}
		start := len(dev.Ports) + 1
		prefix := "P-"
		added := porttpl.NewPorts(count, tpl, prefix, newID)
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
		// replace existing links on either endpoint
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

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "id_" + hex.EncodeToString(b[:])
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
