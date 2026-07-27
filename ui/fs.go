package ui

import "embed"

// Content holds layouts, partials, and static assets.
//
//go:embed layouts/* partials/* static/*
var Content embed.FS
