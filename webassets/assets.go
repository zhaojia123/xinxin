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

func Templates(filingNumbers ...string) (*template.Template, error) {
	filingNumber := ""
	if len(filingNumbers) > 0 {
		filingNumber = filingNumbers[0]
	}
	return template.New("templates").Funcs(template.FuncMap{
		"filingNumber": func() string { return filingNumber },
	}).ParseFS(FS, "templates/*.html")
}

func Static() (fs.FS, error) {
	return fs.Sub(FS, "static")
}
