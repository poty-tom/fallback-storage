package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNewLocalClientRejectsEmptyRootDir(t *testing.T) {
	_, err := NewLocalClient(LocalConfig{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalClientCRUD(t *testing.T) {
	client, err := NewLocalClient(LocalConfig{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewLocalClient error = %v", err)
	}

	if err := client.Put(context.Background(), "nested/file.txt", strings.NewReader("payload")); err != nil {
		t.Fatalf("Put error = %v", err)
	}

	exists, err := client.Exists(context.Background(), "nested/file.txt")
	if err != nil {
		t.Fatalf("Exists error = %v", err)
	}
	if !exists {
		t.Fatal("Exists = false, want true")
	}

	rc, err := client.Get(context.Background(), "nested/file.txt")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close error = %v", err)
	}
	if string(body) != "payload" {
		t.Fatalf("body = %q, want %q", string(body), "payload")
	}

	if err := client.Delete(context.Background(), "nested/file.txt"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}

	exists, err = client.Exists(context.Background(), "nested/file.txt")
	if err != nil {
		t.Fatalf("Exists after delete error = %v", err)
	}
	if exists {
		t.Fatal("Exists = true, want false")
	}
}

func TestLocalClientDeleteMissingFileSucceeds(t *testing.T) {
	client, err := NewLocalClient(LocalConfig{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewLocalClient error = %v", err)
	}

	if err := client.Delete(context.Background(), "missing.txt"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
}

func TestLocalClientListUsesPrefixAndSortedKeys(t *testing.T) {
	rootDir := t.TempDir()
	client, err := NewLocalClient(LocalConfig{RootDir: rootDir})
	if err != nil {
		t.Fatalf("NewLocalClient error = %v", err)
	}

	files := map[string]string{
		"alpha/1.txt": "a",
		"alpha/2.txt": "b",
		"beta/1.txt":  "c",
	}
	for key, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(rootDir, key)), 0o755); err != nil {
			t.Fatalf("MkdirAll error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(rootDir, key), []byte(body), 0o644); err != nil {
			t.Fatalf("WriteFile error = %v", err)
		}
	}

	keys, err := client.List(context.Background(), "alpha/")
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	if !slices.Equal(keys, []string{"alpha/1.txt", "alpha/2.txt"}) {
		t.Fatalf("keys = %v", keys)
	}
}

func TestLocalClientRejectsEscapingPaths(t *testing.T) {
	client, err := NewLocalClient(LocalConfig{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewLocalClient error = %v", err)
	}

	if err := client.Put(context.Background(), "../escape.txt", strings.NewReader("payload")); err == nil {
		t.Fatal("expected error")
	}
	if _, err := client.Get(context.Background(), "../escape.txt"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := client.List(context.Background(), "../escape"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalClientGetMissingFileReturnsError(t *testing.T) {
	client, err := NewLocalClient(LocalConfig{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("NewLocalClient error = %v", err)
	}

	_, err = client.Get(context.Background(), "missing.txt")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get error = %v, want os.ErrNotExist", err)
	}
}
