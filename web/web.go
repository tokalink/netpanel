package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed templates/* static/*
var EmbedFS embed.FS

// GetFileSystem returns the embedded file system for static files
func GetFileSystem() http.FileSystem {
	fsys, err := fs.Sub(EmbedFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}
