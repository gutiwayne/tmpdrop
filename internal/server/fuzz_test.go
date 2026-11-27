package server

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func FuzzRequestByID(f *testing.F) {
	for _, s := range []string{
		"..", "../manifest.json", "..%2fmanifest.json", "%2e%2e", "/etc/passwd", "files", "x\x00y", "",
		strings.Repeat("f", 24),
	} {
		f.Add(s)
	}
	dir := f.TempDir()
	srv := newTestServer(f, dir, testConfig())
	keptID := uploadForFuzz(f, srv.URL, "kept.txt")
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	f.Fuzz(func(t *testing.T, id string) {
		if id == "" || id == keptID {
			return
		}
		esc := url.PathEscape(id)
		for _, req := range []struct{ method, path string }{
			{http.MethodGet, "/d/" + esc},
			{http.MethodGet, "/api/files/" + esc},
			{http.MethodGet, "/api/files/" + esc + "/targets"},
			{http.MethodDelete, "/api/files/" + esc},
		} {
			r, err := http.NewRequest(req.method, srv.URL+req.path, nil)
			if err != nil {
				return
			}
			res, err := noRedirect.Do(r)
			if err != nil {
				t.Fatalf("%s %s: %v", req.method, req.path, err)
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			if res.StatusCode >= 500 {
				t.Fatalf("%s %s = %d", req.method, req.path, res.StatusCode)
			}
			if res.StatusCode < 300 {
				t.Fatalf("%s %s = %d for an id that does not exist", req.method, req.path, res.StatusCode)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
			t.Fatalf("manifest is gone after id %q: %v", id, err)
		}
		res, err := http.Get(srv.URL + "/d/" + keptID)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("stored file answers %d after id %q", res.StatusCode, id)
		}
	})
}

func FuzzUploadFileName(f *testing.F) {
	for _, s := range []string{
		"report.pdf", "../../etc/passwd", `..\evil.exe`, "/", "a\"b.txt", "ñandú.png", "x\r\nSet-Cookie: a=b",
		"page.html", "%2e%2e%2fx", strings.Repeat("ü", 200),
	} {
		f.Add(s)
	}
	cfg := testConfig()
	cfg.MaxFilesClient = 1 << 30
	srv := newTestServer(f, f.TempDir(), cfg)

	f.Fuzz(func(t *testing.T, name string) {
		if name == "" || strings.ContainsAny(name, "\r\n") {
			return
		}
		res := uploadFile(t, srv.URL, name, "data", nil)
		defer res.Body.Close()
		if res.StatusCode >= 500 {
			t.Fatalf("upload %q = %d", name, res.StatusCode)
		}
		if res.StatusCode != http.StatusCreated {
			return
		}
		var body map[string]any
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		id, _ := body["id"].(string)
		dl, err := http.Get(srv.URL + "/d/" + id)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, dl.Body)
		dl.Body.Close()
		cd := dl.Header.Get("Content-Disposition")
		disp, params, err := mime.ParseMediaType(cd)
		if err != nil {
			t.Fatalf("upload %q: unparsable Content-Disposition %q: %v", name, cd, err)
		}
		if disp != "attachment" {
			t.Fatalf("upload %q: disposition %q, want attachment", name, disp)
		}
		if fn := params["filename"]; fn == "" || strings.ContainsAny(fn, "/\\\x00\"") {
			t.Fatalf("upload %q: served filename %q", name, fn)
		}
		if dl.Header.Get("Content-Security-Policy") == "" {
			t.Fatalf("upload %q: download without a Content-Security-Policy", name)
		}
		r, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/files/"+id, nil)
		if res, err := http.DefaultClient.Do(r); err == nil {
			res.Body.Close()
		}
	})
}

func uploadForFuzz(f *testing.F, base, name string) string {
	res := uploadFile(f, base, name, "keep", nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		f.Fatalf("seed upload = %d", res.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		f.Fatal(err)
	}
	id, _ := body["id"].(string)
	return id
}
