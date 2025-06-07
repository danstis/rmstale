package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during function f execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	f()
	if err := w.Close(); err != nil {
		panic(err)
	}
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		panic(err)
	}
	return buf.String()
}

func TestMain_VersionFlag(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale", "-v"}
	output := captureOutput(func() { main() })
	if !strings.Contains(output, "rmstale v") {
		t.Fatalf("expected version info, got %q", output)
	}
}

func TestMain_NoFlagsShowsUsage(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"rmstale"}
	output := captureOutput(func() { main() })
	if !strings.Contains(output, "Usage of rmstale") {
		t.Fatalf("expected usage output, got %q", output)
	}
}

func TestGetExt(t *testing.T) {
	for _, tt := range []struct{ path, want string }{
		{"file.txt", "txt"},
		{"dir/file.tar.gz", "gz"},
		{"dir/file", ""},
		{"dir.name/file", ""},
	} {
		if got := getExt(tt.path); got != tt.want {
			t.Errorf("getExt(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestMatchExt(t *testing.T) {
	for _, tt := range []struct {
		name string
		ext  string
		want bool
	}{
		{"empty ext always matches", "", true},
		{"match", "txt", true},
		{"no match", "gz", false},
	} {
		file := "test.txt"
		if tt.name == "no match" {
			file = "test.doc"
		}
		if got := matchExt(file, tt.ext); got != tt.want {
			t.Errorf("%s: matchExt(%q,%q) = %v, want %v", tt.name, file, tt.ext, got, tt.want)
		}
	}
}

func TestGetDirectoryContents(t *testing.T) {
	dir := os.TempDir()
	tmp, err := os.CreateTemp(dir, "file")
	if err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(tmp.Name()); err != nil {
			t.Fatal(err)
		}
	})

	infos, err := getDirectoryContents(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) == 0 {
		t.Fatalf("expected some entries, got 0")
	}

	_, err = getDirectoryContents("non-existent")
	if err == nil {
		t.Fatalf("expected error for bad directory")
	}
}
