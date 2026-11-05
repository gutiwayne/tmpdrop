package convertx

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func makeKMZ(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestExtractKML(t *testing.T) {
	kmz := makeKMZ(t, map[string]string{"files/icon.png": "png", "doc.kml": "<kml/>"})
	var out bytes.Buffer
	if err := extractKML(bytes.NewReader(kmz), &out); err != nil {
		t.Fatalf("extractKML: %v", err)
	}
	if out.String() != "<kml/>" {
		t.Fatalf("got %q", out.String())
	}
}

func TestExtractKMLNoKML(t *testing.T) {
	kmz := makeKMZ(t, map[string]string{"a.txt": "x"})
	err := extractKML(bytes.NewReader(kmz), &bytes.Buffer{})
	if !errors.Is(err, ErrConvertFailed) {
		t.Fatalf("want ErrConvertFailed, got %v", err)
	}
}

// A request without an explicit converter (plain API use) must still take the
// local gpsbabel path and never reach ConvertX, which does not know GPS formats
// and would leave the job hanging until the deadline.
func TestConvertGPSWithoutConverterStaysLocal(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "should not be called", http.StatusTeapot)
	}))
	defer srv.Close()

	c := New(srv.URL, "a@b.c", "pw", 5*time.Second)
	src := Source{
		Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("<gpx/>")), nil },
		Size: 6,
	}
	_, _ = c.Convert(context.Background(), src, "track.gpx", "kml", "", &bytes.Buffer{})
	if n := hits.Load(); n != 0 {
		t.Fatalf("ConvertX was called %d times for a gpsbabel conversion", n)
	}
}

func TestExtractKMLNotZip(t *testing.T) {
	err := extractKML(bytes.NewReader([]byte("not a zip")), &bytes.Buffer{})
	if !errors.Is(err, ErrConvertFailed) {
		t.Fatalf("want ErrConvertFailed, got %v", err)
	}
}
