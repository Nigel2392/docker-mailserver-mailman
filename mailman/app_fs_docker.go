//go:build docker
// +build docker

package main

import (
	"embed"
	"io/fs"
	"path/filepath"

	"github.com/Nigel2392/go-django/src/core/filesystem"
)

//go:embed assets/**
var assetsFS embed.FS

var RUNNING_IN_DOCKER = true

func joinRootPath(p ...string) string {
	return filepath.Join(append([]string{"/mailman"}, p...)...)
}

func initAppFS() (fs.FS, fs.FS) {
	var tplFS = filesystem.Sub(assetsFS, "assets/templates")
	var staticFS = filesystem.Sub(assetsFS, "assets/static")
	return tplFS, staticFS
}
