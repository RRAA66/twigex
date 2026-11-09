// Copyright (c) 2022 Twigex, SIA.
// SPDX-License-Identifier: AGPL-3.0-only

package filestore

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLocalListFilesWalksNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", filepath.Join("sub", "b.txt")} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
	}

	b := &LocalFileBackend{}

	files, err := b.ListFiles(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}

	var paths []string
	for _, f := range files {
		if f.ModTime.IsZero() {
			t.Errorf("%s has no modification time", f.Path)
		}
		paths = append(paths, f.Path)
	}

	if want := []string{filepath.Join(dir, "a.txt"), filepath.Join(dir, "sub", "b.txt")}; !slices.Equal(paths, want) {
		t.Errorf("listed %v, want %v", paths, want)
	}

	missing, err := b.ListFiles(context.Background(), filepath.Join(dir, "missing"))
	if err != nil || len(missing) != 0 {
		t.Errorf("a missing directory listed %v, %v; want nothing", missing, err)
	}
}
