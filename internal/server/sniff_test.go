package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSniffGPSFormats(t *testing.T) {
	fit := []byte{14, 0x10, 0x5c, 0x08, 0, 0, 0, 0, '.', 'F', 'I', 'T', 0, 0}
	cases := []struct {
		name string
		body []byte
		want string
	}{
		{"fit", fit, "application/vnd.garmin.fit"},
		{"gpx", []byte(`<?xml version="1.0"?><gpx version="1.1">`), "application/gpx+xml"},
		{"kml", []byte(`<?xml version="1.0"?><kml xmlns="x">`), "application/vnd.google-earth.kml+xml"},
		{"geojson", []byte("{\n  \"features\": [\n    {\n      \"geometry\": {\n        \"coordinates\": [[-3.7, 40.4]]"), "application/geo+json"},
		{"json", []byte(`{"name": "not geo"}`), "text/plain; charset=utf-8"},
		{"short", []byte{14, 0x10}, "application/octet-stream"},
	}
	for _, c := range cases {
		p := filepath.Join(t.TempDir(), c.name)
		if err := os.WriteFile(p, c.body, 0o600); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := sniff(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
