package main

import "embed"

//go:embed bootstrap/* manifests/*
var InstallerData embed.FS
