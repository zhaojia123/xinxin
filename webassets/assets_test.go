package webassets

import (
	"io/fs"
	"testing"
)

func TestEmbeddedAssets(t *testing.T) {
	if _, err := Templates(); err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	staticFiles, err := Static()
	if err != nil {
		t.Fatalf("open static assets: %v", err)
	}
	if _, err := fs.ReadFile(staticFiles, "app.css"); err != nil {
		t.Fatalf("read embedded stylesheet: %v", err)
	}
}
