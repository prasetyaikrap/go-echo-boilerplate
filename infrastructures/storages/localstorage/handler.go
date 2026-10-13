package localstorage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"go-serviceboilerplate/commons/models"
	"go-serviceboilerplate/infrastructures/configurations"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

// LocalStorageInstance stores files in a directory on the local filesystem. Create one per directory the service needs, e.g.
//
//	mainStorage := localstorage.NewLocalStorageInstance(configs, configs.Envs.MainLocalStorage)
//	cacheStorage := localstorage.NewLocalStorageInstance(configs, configs.Envs.CacheLocalStorage)
//
// Keys are slash-separated paths relative to BaseDir (e.g. "reports/2026/jan.pdf"). Every operation is confined to BaseDir:
// keys that try to escape it (e.g. "../etc/passwd") or follow a symlink out of it are rejected.
// Missing files return an error matching fs.ErrNotExist, checked with errors.Is(err, fs.ErrNotExist).
type LocalStorageInstance struct {
	Name     string
	BaseDir  string
	DirMode  os.FileMode
	FileMode os.FileMode

	root *os.Root
}

func NewLocalStorageInstance(configs *configurations.Configs, storageConfig models.LocalStorageConfig) *LocalStorageInstance {
	if storageConfig.BaseDir == "" {
		configs.Logger.Fatal("Local storage base directory is not configured", "name", storageConfig.Name)
	}

	if err := os.MkdirAll(storageConfig.BaseDir, storageConfig.DirMode); err != nil {
		configs.Logger.Fatal("Failed to create local storage directory", "name", storageConfig.Name, "base_dir", storageConfig.BaseDir, "error", err)
	}

	root, err := os.OpenRoot(storageConfig.BaseDir)
	if err != nil {
		configs.Logger.Fatal("Failed to open local storage directory", "name", storageConfig.Name, "base_dir", storageConfig.BaseDir, "error", err)
	}

	configs.Logger.Info("Local Storage Successfully Initialized", "name", storageConfig.Name, "base_dir", storageConfig.BaseDir)

	return &LocalStorageInstance{
		Name:     storageConfig.Name,
		BaseDir:  storageConfig.BaseDir,
		DirMode:  storageConfig.DirMode,
		FileMode: storageConfig.FileMode,
		root:     root,
	}
}

// cleanKey normalizes a key into a path relative to BaseDir. Escaping BaseDir is also blocked by os.Root itself
func (l *LocalStorageInstance) cleanKey(key string) (string, error) {
	cleaned := path.Clean(strings.TrimPrefix(key, "/"))

	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("local storage %s: invalid key %q", l.Name, key)
	}

	return cleaned, nil
}

// SaveFile writes reader to key, creating parent directories as needed and replacing any existing file.
// The write is atomic: readers see either the old file or the complete new one, never a partial file.
func (l *LocalStorageInstance) SaveFile(key string, reader io.Reader) error {
	assignedKey, err := l.cleanKey(key)
	if err != nil {
		return err
	}

	if dir := path.Dir(assignedKey); dir != "." {
		if err := l.root.MkdirAll(dir, l.DirMode); err != nil {
			return fmt.Errorf("local storage %s save file %q: create directory: %w", l.Name, assignedKey, err)
		}
	}

	tempKey, err := tempKeyFor(assignedKey)
	if err != nil {
		return fmt.Errorf("local storage %s save file %q: %w", l.Name, assignedKey, err)
	}

	file, err := l.root.OpenFile(tempKey, os.O_WRONLY|os.O_CREATE|os.O_EXCL, l.FileMode)
	if err != nil {
		return fmt.Errorf("local storage %s save file %q: %w", l.Name, assignedKey, err)
	}

	_, err = io.Copy(file, reader)
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = l.root.Rename(tempKey, assignedKey)
	}

	if err != nil {
		_ = l.root.Remove(tempKey)
		return fmt.Errorf("local storage %s save file %q: %w", l.Name, assignedKey, err)
	}

	return nil
}

// tempKeyFor returns a hidden, unique sibling path used while a file is being written
func tempKeyFor(key string) (string, error) {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return "", err
	}

	return path.Join(path.Dir(key), "."+path.Base(key)+".tmp-"+hex.EncodeToString(suffix)), nil
}

// WriteFile is SaveFile for content already in memory (e.g. JSON config or a cache entry)
func (l *LocalStorageInstance) WriteFile(key string, data []byte) error {
	return l.SaveFile(key, bytes.NewReader(data))
}

// GetFile opens the file at key for streaming. The caller must close the returned ReadCloser
func (l *LocalStorageInstance) GetFile(key string) (io.ReadCloser, error) {
	assignedKey, err := l.cleanKey(key)
	if err != nil {
		return nil, err
	}

	file, err := l.root.Open(assignedKey)
	if err != nil {
		return nil, fmt.Errorf("local storage %s get file %q: %w", l.Name, assignedKey, err)
	}

	return file, nil
}

// ReadFile returns the whole content of the file at key. Use GetFile for large files
func (l *LocalStorageInstance) ReadFile(key string) ([]byte, error) {
	assignedKey, err := l.cleanKey(key)
	if err != nil {
		return nil, err
	}

	data, err := l.root.ReadFile(assignedKey)
	if err != nil {
		return nil, fmt.Errorf("local storage %s read file %q: %w", l.Name, assignedKey, err)
	}

	return data, nil
}

// FileExists reports whether a file exists at key
func (l *LocalStorageInstance) FileExists(key string) (bool, error) {
	assignedKey, err := l.cleanKey(key)
	if err != nil {
		return false, err
	}

	info, err := l.root.Stat(assignedKey)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("local storage %s stat file %q: %w", l.Name, assignedKey, err)
	}

	return !info.IsDir(), nil
}

// DeleteFile removes the file at key. A missing file is not an error
func (l *LocalStorageInstance) DeleteFile(key string) error {
	assignedKey, err := l.cleanKey(key)
	if err != nil {
		return err
	}

	if err := l.root.Remove(assignedKey); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("local storage %s delete file %q: %w", l.Name, assignedKey, err)
	}

	return nil
}

// DeleteDir removes the directory at key and everything inside it (e.g. clearing a cache folder). A missing directory is not an error
func (l *LocalStorageInstance) DeleteDir(key string) error {
	assignedKey, err := l.cleanKey(key)
	if err != nil {
		return err
	}

	if err := l.root.RemoveAll(assignedKey); err != nil {
		return fmt.Errorf("local storage %s delete dir %q: %w", l.Name, assignedKey, err)
	}

	return nil
}

// ListFiles returns the keys of all files under dir, recursively. Use "" to list the whole storage.
// Temporary files from in-progress writes are skipped. A missing directory returns an empty list
func (l *LocalStorageInstance) ListFiles(dir string) ([]string, error) {
	assignedDir := "."
	if strings.Trim(dir, "/") != "" {
		cleaned, err := l.cleanKey(dir)
		if err != nil {
			return nil, err
		}
		assignedDir = cleaned
	}

	keys := []string{}
	err := fs.WalkDir(l.root.FS(), assignedDir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && !strings.Contains(entry.Name(), ".tmp-") {
			keys = append(keys, filePath)
		}
		return nil
	})

	if errors.Is(err, fs.ErrNotExist) {
		return keys, nil
	}
	if err != nil {
		return nil, fmt.Errorf("local storage %s list files %q: %w", l.Name, assignedDir, err)
	}

	return keys, nil
}

func (l *LocalStorageInstance) Close() error {
	return l.root.Close()
}
