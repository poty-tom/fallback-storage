package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fallback-storage/internal/storage"
)

var _ storage.StorageClient = (*LocalClient)(nil)

type LocalConfig struct {
	RootDir string
}

type LocalClient struct {
	rootDir string
}

func NewLocalClient(cfg LocalConfig) (*LocalClient, error) {
	rootDir := strings.TrimSpace(cfg.RootDir)
	if rootDir == "" {
		return nil, errors.New("root dir is required")
	}

	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve root dir: %w", err)
	}

	return &LocalClient{rootDir: absRootDir}, nil
}

func (c *LocalClient) Put(_ context.Context, key string, content io.Reader) error {
	path, err := c.resolveKeyPath(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent dir for %q: %w", key, err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %q: %w", key, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return fmt.Errorf("write file %q: %w", key, err)
	}

	return nil
}

func (c *LocalClient) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := c.resolveKeyPath(key)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", key, err)
	}

	return file, nil
}

func (c *LocalClient) Delete(_ context.Context, key string) error {
	path, err := c.resolveKeyPath(key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove file %q: %w", key, err)
	}

	return nil
}

func (c *LocalClient) Exists(_ context.Context, key string) (bool, error) {
	path, err := c.resolveKeyPath(key)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat file %q: %w", key, err)
}

func (c *LocalClient) List(_ context.Context, prefix string) ([]string, error) {
	prefix, err := normalizePrefix(prefix)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0)
	if err := filepath.WalkDir(c.rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(c.rootDir, path)
		if err != nil {
			return err
		}

		key := filepath.ToSlash(rel)
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
		return nil
	}); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("list files with prefix %q: %w", prefix, err)
	}

	sort.Strings(keys)
	return keys, nil
}

func (c *LocalClient) resolveKeyPath(key string) (string, error) {
	cleanKey, err := normalizeKey(key)
	if err != nil {
		return "", err
	}

	path := filepath.Join(c.rootDir, filepath.FromSlash(cleanKey))
	if !isWithinBase(c.rootDir, path) {
		return "", fmt.Errorf("key %q escapes root dir", key)
	}

	return path, nil
}

func normalizeKey(key string) (string, error) {
	key = strings.TrimSpace(strings.TrimPrefix(filepath.ToSlash(key), "/"))
	if key == "" {
		return "", errors.New("key is required")
	}

	cleaned := filepath.ToSlash(filepath.Clean(key))
	if cleaned == "." || strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("invalid key %q", key)
	}

	return cleaned, nil
}

func normalizePrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(strings.TrimPrefix(filepath.ToSlash(prefix), "/"))
	if prefix == "" {
		return "", nil
	}

	cleaned := filepath.ToSlash(filepath.Clean(prefix))
	if cleaned == "." {
		return "", nil
	}
	if strings.HasPrefix(cleaned, "../") || cleaned == ".." {
		return "", fmt.Errorf("invalid prefix %q", prefix)
	}

	return cleaned, nil
}

func isWithinBase(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}

	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}
