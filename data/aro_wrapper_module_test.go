package data

import (
	"embed"
)

//go:embed data/bootstrap/* data/manifests/*
var _ embed.FS
