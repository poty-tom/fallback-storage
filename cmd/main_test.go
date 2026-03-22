package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	applicationstorage "fallback-storage/internal/application/storage"
	"fallback-storage/internal/injector"
)

type stubStorageAccess struct {
	putFn    func(context.Context, string, io.Reader) (applicationstorage.Location, error)
	getFn    func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error)
	deleteFn func(context.Context, string, applicationstorage.Location) error
}

func (s stubStorageAccess) Put(ctx context.Context, key string, src io.Reader) (applicationstorage.Location, error) {
	return s.putFn(ctx, key, src)
}

func (s stubStorageAccess) Get(ctx context.Context, key string, loc applicationstorage.Location) (io.ReadCloser, error) {
	return s.getFn(ctx, key, loc)
}

func (s stubStorageAccess) Delete(ctx context.Context, key string, loc applicationstorage.Location) error {
	return s.deleteFn(ctx, key, loc)
}

func TestRunPut(t *testing.T) {
	inputFile, err := os.CreateTemp(t.TempDir(), "input-*")
	if err != nil {
		t.Fatalf("CreateTemp error = %v", err)
	}
	if _, err := inputFile.WriteString("payload"); err != nil {
		t.Fatalf("WriteString error = %v", err)
	}
	if err := inputFile.Close(); err != nil {
		t.Fatalf("Close error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = run(context.Background(), []string{"put", "file.txt", inputFile.Name()}, stubStorageAccess{
		putFn: func(_ context.Context, key string, src io.Reader) (applicationstorage.Location, error) {
			body, readErr := io.ReadAll(src)
			if readErr != nil {
				t.Fatalf("ReadAll error = %v", readErr)
			}
			if key != "file.txt" {
				t.Fatalf("key = %q", key)
			}
			if string(body) != "payload" {
				t.Fatalf("body = %q", string(body))
			}
			return applicationstorage.LocationPrimary, nil
		},
		getFn: func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error) {
			panic("unexpected call")
		},
		deleteFn: func(context.Context, string, applicationstorage.Location) error { panic("unexpected call") },
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "stored in primary" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunGet(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(context.Background(), []string{"get", "file.txt", "fallback"}, stubStorageAccess{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(_ context.Context, key string, loc applicationstorage.Location) (io.ReadCloser, error) {
			if key != "file.txt" || loc != applicationstorage.LocationFallback {
				t.Fatalf("unexpected args: %q %v", key, loc)
			}
			return io.NopCloser(strings.NewReader("payload")), nil
		},
		deleteFn: func(context.Context, string, applicationstorage.Location) error { panic("unexpected call") },
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if stdout.String() != "payload" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDelete(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(context.Background(), []string{"delete", "file.txt", "primary"}, stubStorageAccess{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error) {
			panic("unexpected call")
		},
		deleteFn: func(_ context.Context, key string, loc applicationstorage.Location) error {
			if key != "file.txt" || loc != applicationstorage.LocationPrimary {
				t.Fatalf("unexpected args: %q %v", key, loc)
			}
			return nil
		},
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "deleted from primary" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunRejectsInvalidLocation(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(context.Background(), []string{"get", "file.txt", "elsewhere"}, stubStorageAccess{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error) {
			panic("unexpected call")
		},
		deleteFn: func(context.Context, string, applicationstorage.Location) error { panic("unexpected call") },
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(context.Background(), []string{"unknown"}, stubStorageAccess{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error) {
			panic("unexpected call")
		},
		deleteFn: func(context.Context, string, applicationstorage.Location) error { panic("unexpected call") },
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	cfg, err := loadConfigFromEnv(func(key string) string {
		values := map[string]string{
			"GARAGE_ENDPOINT":          "http://127.0.0.1:3900",
			"GARAGE_REGION":            "garage",
			"GARAGE_BUCKET":            "bucket",
			"GARAGE_ACCESS_KEY_ID":     "access",
			"GARAGE_SECRET_ACCESS_KEY": "secret",
			"LOCAL_ROOT_DIR":           "/tmp/storage",
		}
		return values[key]
	})
	if err != nil {
		t.Fatalf("loadConfigFromEnv error = %v", err)
	}
	if !cfg.GarageUsePathStyle {
		t.Fatal("GarageUsePathStyle = false, want true")
	}
}

func TestLoadConfigFromEnvRejectsMissingValues(t *testing.T) {
	_, err := loadConfigFromEnv(func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseLocation(t *testing.T) {
	loc, err := parseLocation("primary")
	if err != nil {
		t.Fatalf("parseLocation error = %v", err)
	}
	if loc != applicationstorage.LocationPrimary {
		t.Fatalf("loc = %v", loc)
	}
}

func TestParseBoolWithDefault(t *testing.T) {
	value, err := parseBoolWithDefault("", true)
	if err != nil {
		t.Fatalf("parseBoolWithDefault error = %v", err)
	}
	if !value {
		t.Fatal("value = false, want true")
	}
}

func TestRunPropagatesUsecaseErrors(t *testing.T) {
	expectedErr := errors.New("boom")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run(context.Background(), []string{"delete", "file.txt", "primary"}, stubStorageAccess{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error) {
			panic("unexpected call")
		},
		deleteFn: func(context.Context, string, applicationstorage.Location) error {
			return expectedErr
		},
	}, &stdout, &stderr)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadConfigFromEnvReturnsInjectorConfig(t *testing.T) {
	cfg, err := loadConfigFromEnv(func(key string) string {
		switch key {
		case "GARAGE_ENDPOINT":
			return "http://127.0.0.1:3900"
		case "GARAGE_REGION":
			return "garage"
		case "GARAGE_BUCKET":
			return "bucket"
		case "GARAGE_ACCESS_KEY_ID":
			return "access"
		case "GARAGE_SECRET_ACCESS_KEY":
			return "secret"
		case "LOCAL_ROOT_DIR":
			return "/tmp/storage"
		case "GARAGE_USE_PATH_STYLE":
			return "false"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("loadConfigFromEnv error = %v", err)
	}
	if cfg != (injector.Config{
		GarageEndpoint:        "http://127.0.0.1:3900",
		GarageRegion:          "garage",
		GarageBucket:          "bucket",
		GarageAccessKeyID:     "access",
		GarageSecretAccessKey: "secret",
		GarageUsePathStyle:    false,
		LocalRootDir:          "/tmp/storage",
	}) {
		t.Fatalf("cfg = %+v", cfg)
	}
}
