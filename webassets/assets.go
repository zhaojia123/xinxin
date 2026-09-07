package webassets

import (
	"embed"
	"html/template"
	"io/fs"
)

// FS contains all server-rendered pages and public assets in the executable.
//
//go:embed templates/*.html static/*
var FS embed.FS

func Templates() (*template.Template, error) {
	return template.ParseFS(FS, "templates/*.html")
}

func Static() (fs.FS, error) {
	return fs.Sub(FS, "static")
}
