package console

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed ui/dist/*
var embedFS embed.FS
var UIFS = &uiFS{}

type uiFS struct {
	Nt bool
}

func (fs *uiFS) Open(name string) (fs.File, error) {
	if fs.Nt {
		return embedFS.Open(path.Join("ui", "dist", "prod-nt", name))
	}
	return embedFS.Open(path.Join("ui", "dist", "prod", name))
}

var UI = http.FileServer(http.FS(UIFS))
