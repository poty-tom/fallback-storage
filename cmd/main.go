package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	applicationstorage "fallback-storage/internal/application/storage"
	"fallback-storage/internal/injector"
)

const usageText = `usage:
  put <key> <input-file>
  get <key> <primary|fallback>
  delete <key> <primary|fallback>`

type storageAccess interface {
	Put(ctx context.Context, key string, src io.Reader) (applicationstorage.Location, error)
	Get(ctx context.Context, key string, loc applicationstorage.Location) (io.ReadCloser, error)
	Delete(ctx context.Context, key string, loc applicationstorage.Location) error
}

func main() {
	ctx := context.Background()

	cfg, err := loadConfigFromEnv(os.Getenv)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	usecase, err := injector.BuildStorageAccessUsecase(ctx, cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "build usecase: %v\n", err)
		os.Exit(1)
	}

	if err := run(ctx, os.Args[1:], usecase, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, svc storageAccess, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, usageText)
		return errors.New("command is required")
	}

	switch args[0] {
	case "put":
		if len(args) != 3 {
			_, _ = fmt.Fprintln(stderr, usageText)
			return errors.New("put requires <key> <input-file>")
		}
		return runPut(ctx, svc, stdout, args[1], args[2])
	case "get":
		if len(args) != 3 {
			_, _ = fmt.Fprintln(stderr, usageText)
			return errors.New("get requires <key> <primary|fallback>")
		}
		return runGet(ctx, svc, stdout, args[1], args[2])
	case "delete":
		if len(args) != 3 {
			_, _ = fmt.Fprintln(stderr, usageText)
			return errors.New("delete requires <key> <primary|fallback>")
		}
		return runDelete(ctx, svc, stdout, args[1], args[2])
	default:
		_, _ = fmt.Fprintln(stderr, usageText)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runPut(ctx context.Context, svc storageAccess, stdout io.Writer, key, inputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer file.Close()

	loc, err := svc.Put(ctx, key, file)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "stored in %s\n", formatLocation(loc))
	return nil
}

func runGet(ctx context.Context, svc storageAccess, stdout io.Writer, key, rawLocation string) error {
	loc, err := parseLocation(rawLocation)
	if err != nil {
		return err
	}

	rc, err := svc.Get(ctx, key, loc)
	if err != nil {
		return fmt.Errorf("get object: %w", err)
	}
	defer rc.Close()

	if _, err := io.Copy(stdout, rc); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func runDelete(ctx context.Context, svc storageAccess, stdout io.Writer, key, rawLocation string) error {
	loc, err := parseLocation(rawLocation)
	if err != nil {
		return err
	}

	if err := svc.Delete(ctx, key, loc); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "deleted from %s\n", formatLocation(loc))
	return nil
}

func loadConfigFromEnv(getenv func(string) string) (injector.Config, error) {
	usePathStyle, err := parseBoolWithDefault(getenv("GARAGE_USE_PATH_STYLE"), true)
	if err != nil {
		return injector.Config{}, fmt.Errorf("parse GARAGE_USE_PATH_STYLE: %w", err)
	}

	cfg := injector.Config{
		GarageEndpoint:        strings.TrimSpace(getenv("GARAGE_ENDPOINT")),
		GarageRegion:          strings.TrimSpace(getenv("GARAGE_REGION")),
		GarageBucket:          strings.TrimSpace(getenv("GARAGE_BUCKET")),
		GarageAccessKeyID:     strings.TrimSpace(getenv("GARAGE_ACCESS_KEY_ID")),
		GarageSecretAccessKey: strings.TrimSpace(getenv("GARAGE_SECRET_ACCESS_KEY")),
		GarageUsePathStyle:    usePathStyle,
		LocalRootDir:          strings.TrimSpace(getenv("LOCAL_ROOT_DIR")),
	}

	if cfg.GarageEndpoint == "" {
		return injector.Config{}, errors.New("GARAGE_ENDPOINT is required")
	}
	if cfg.GarageRegion == "" {
		return injector.Config{}, errors.New("GARAGE_REGION is required")
	}
	if cfg.GarageBucket == "" {
		return injector.Config{}, errors.New("GARAGE_BUCKET is required")
	}
	if cfg.GarageAccessKeyID == "" {
		return injector.Config{}, errors.New("GARAGE_ACCESS_KEY_ID is required")
	}
	if cfg.GarageSecretAccessKey == "" {
		return injector.Config{}, errors.New("GARAGE_SECRET_ACCESS_KEY is required")
	}
	if cfg.LocalRootDir == "" {
		return injector.Config{}, errors.New("LOCAL_ROOT_DIR is required")
	}

	return cfg, nil
}

func parseBoolWithDefault(raw string, defaultValue bool) (bool, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(raw)
}

func parseLocation(raw string) (applicationstorage.Location, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "primary":
		return applicationstorage.LocationPrimary, nil
	case "fallback":
		return applicationstorage.LocationFallback, nil
	default:
		return 0, fmt.Errorf("invalid location %q", raw)
	}
}

func formatLocation(loc applicationstorage.Location) string {
	switch loc {
	case applicationstorage.LocationPrimary:
		return "primary"
	case applicationstorage.LocationFallback:
		return "fallback"
	default:
		return "unknown"
	}
}
