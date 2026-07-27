# rackwire

Dokumentiere dein Rack: Patchpanels, Router und Dosen — inkl. **Port-Labels**, **Pinanzahl** und **Aderfarben**, mit Templates für Standardbelegungen (T568B, Klingel, Analog-Telefon).

## Stack

- Go (`net/http` + `html/template`)
- HTMX für Partial-Updates am Port-Editor
- JSON-Persistenz (`data/rack.json`)
- Docker / Compose / Makefile

## Quick start

```bash
make up
# http://localhost:3040
```

Lokal ohne Docker:

```bash
make run
```

## Features (MVP)

- Geräte: Patchpanel, Router, Dose
- **JSON-Templates** unter `data/templates/` (T568A Standard, T568B, ISDN-4, Klingel, Telefon)
- Pro Port: Label, Template, editierbare Pinnummern/Farben; „Als Template speichern“
- Patchverbindungen + Verbindungskarte (`/map`)
- Seed-Daten: 3 Patchpanels, Router, Spezialdosen

## Image

```bash
docker pull ghcr.io/spakarl/rackwire:latest
```
