package localstorage

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
)

func newTestStorage(t *testing.T) (*LocalStorageInstance, string) {
	t.Helper()

	parent := t.TempDir()
	baseDir := filepath.Join(parent, "data")
	configs := &configurations.Configs{Envs: &models.ENVConfig{}, Logger: configurations.NewSlogLogger()}

	storage := NewLocalStorageInstance(configs, models.LocalStorageConfig{Name: "test", BaseDir: baseDir, DirMode: 0o755, FileMode: 0o644})
	t.Cleanup(func() { storage.Close() })

	return storage, parent
}

func TestSaveAndReadFile(t *testing.T) {
	storage, _ := newTestStorage(t)

	if err := storage.WriteFile("cache/users/1.json", []byte(`{"id":1}`)); err != nil {
		t.Fatalf("write nested file: %v", err)
	}
	if err := storage.SaveFile("/cache/users/1.json", strings.NewReader(`{"id":2}`)); err != nil {
		t.Fatalf("overwrite with leading slash: %v", err)
	}

	data, err := storage.ReadFile("cache/users/1.json")
	if err != nil || string(data) != `{"id":2}` {
		t.Errorf("read: got (%q, %v)", data, err)
	}

	file, err := storage.GetFile("cache/users/1.json")
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	defer file.Close()
	if streamed, _ := io.ReadAll(file); string(streamed) != `{"id":2}` {
		t.Errorf("stream: got %q", streamed)
	}
}

func TestMissingFile(t *testing.T) {
	storage, _ := newTestStorage(t)

	if _, err := storage.ReadFile("missing.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("read missing: error = %v, want fs.ErrNotExist", err)
	}
	if exists, err := storage.FileExists("missing.txt"); exists || err != nil {
		t.Errorf("exists missing: got (%v, %v)", exists, err)
	}
	if err := storage.DeleteFile("missing.txt"); err != nil {
		t.Errorf("delete missing should not fail: %v", err)
	}
}

func TestKeysCannotEscapeBaseDir(t *testing.T) {
	storage, parent := newTestStorage(t)

	secret := filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secret, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"../secret.txt", "a/../../secret.txt", "/../../secret.txt", "..", ""} {
		if _, err := storage.ReadFile(key); err == nil {
			t.Errorf("read %q should be rejected", key)
		}
		if err := storage.WriteFile(key, []byte("pwned")); err == nil {
			t.Errorf("write %q should be rejected", key)
		}
	}

	if err := os.Symlink(secret, filepath.Join(storage.BaseDir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.ReadFile("link.txt"); err == nil {
		t.Error("following a symlink out of the base dir should be rejected")
	}

	if content, _ := os.ReadFile(secret); string(content) != "outside" {
		t.Errorf("file outside the base dir was modified: %q", content)
	}
}

func TestListAndDelete(t *testing.T) {
	storage, _ := newTestStorage(t)

	storage.WriteFile("cache/a.json", []byte("a"))
	storage.WriteFile("cache/b/c.json", []byte("c"))
	storage.WriteFile("config/app.yaml", []byte("x"))

	all, err := storage.ListFiles("")
	if err != nil || !slices.Equal(all, []string{"cache/a.json", "cache/b/c.json", "config/app.yaml"}) {
		t.Errorf("list all: got (%v, %v)", all, err)
	}

	if missing, err := storage.ListFiles("missing"); err != nil || len(missing) != 0 {
		t.Errorf("list missing dir: got (%v, %v)", missing, err)
	}

	if err := storage.DeleteDir("cache"); err != nil {
		t.Fatalf("delete dir: %v", err)
	}
	if remaining, _ := storage.ListFiles(""); !slices.Equal(remaining, []string{"config/app.yaml"}) {
		t.Errorf("after delete dir: got %v", remaining)
	}
}
