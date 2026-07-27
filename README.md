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
- Builtin-Templates: RJ45 T568A/B, 2-Pin Klingel, Analog-Telefon, Blank
- Pro Port: Label, Template, Pin-Overrides (Signal + Farbe)
- Patchverbindungen zwischen Ports
- Seed-Daten: 3 Patchpanels, Router, Klingel- und Telefon-Dose

## Image

```bash
docker pull ghcr.io/spakarl/rackwire:latest
```
