package data

import "embed"

//go:embed bootstrap/* manifests/*
var _ embed.FS
