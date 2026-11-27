package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzSanitizeName(f *testing.F) {
	for _, s := range []string{
		"report.pdf", "../../etc/passwd", `..\..\windows\win.ini`, "/abs/path/x.txt", "a\x00b.txt",
		"line\nbreak\r.txt", `quote".txt`, ".", "..", "", "   ", strings.Repeat("ñ", 150), "dir/",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, in string) {
		out := sanitizeName(in)
		if out == "" || out == "." || out == ".." {
			t.Fatalf("sanitizeName(%q) = %q", in, out)
		}
		if strings.ContainsAny(out, "/\\\x00\n\r\t\"") {
			t.Fatalf("sanitizeName(%q) = %q keeps a separator or control character", in, out)
		}
		if len(out) > maxNameLength {
			t.Fatalf("sanitizeName(%q) is %d bytes, limit %d", in, len(out), maxNameLength)
		}
		if utf8.ValidString(in) && !utf8.ValidString(out) {
			t.Fatalf("sanitizeName(%q) = %q is not valid UTF-8", in, out)
		}
		if filepath.Base(out) != out {
			t.Fatalf("sanitizeName(%q) = %q is not a single path element", in, out)
		}
	})
}

func FuzzLookupByID(f *testing.F) {
	for _, s := range []string{
		"", ".", "..", "../manifest.json", "../../outside", "/etc/passwd", "files", "%2e%2e%2f",
		strings.Repeat("a", 24), "0123456789abcdef01234567", "x\x00y",
	} {
		f.Add(s)
	}
	root := f.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		f.Fatal(err)
	}
	s, err := Open(filepath.Join(root, "store"), testLimits())
	if err != nil {
		f.Fatal(err)
	}
	kept, err := s.Add(strings.NewReader("data"), testName, "text/plain", AddOptions{})
	if err != nil {
		f.Fatal(err)
	}

	f.Fuzz(func(t *testing.T, id string) {
		if id == kept.ID {
			return
		}
		if _, err := s.Get(id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Get(%q) = %v, want ErrNotFound", id, err)
		}
		if _, blob, err := s.OpenBlob(id); !errors.Is(err, ErrNotFound) {
			if blob != nil {
				blob.Close()
			}
			t.Fatalf("OpenBlob(%q) = %v, want ErrNotFound", id, err)
		}
		if _, err := s.BlobPath(id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("BlobPath(%q) = %v, want ErrNotFound", id, err)
		}
		if err := s.Delete(id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Delete(%q) = %v, want ErrNotFound", id, err)
		}
		if _, err := os.Stat(outside); err != nil {
			t.Fatalf("file outside the store is gone after id %q: %v", id, err)
		}
		if _, err := os.Stat(s.manifestPath); err != nil {
			t.Fatalf("manifest is gone after id %q: %v", id, err)
		}
		if _, err := s.Get(kept.ID); err != nil {
			t.Fatalf("stored file lost after id %q: %v", id, err)
		}
	})
}
