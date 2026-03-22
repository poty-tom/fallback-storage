package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	applicationstorage "fallback-storage/internal/application/storage"
)

type stubStorage struct {
	putFn    func(context.Context, string, io.Reader) (applicationstorage.Location, error)
	getFn    func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error)
	deleteFn func(context.Context, string, applicationstorage.Location) error
}

func (s stubStorage) Put(ctx context.Context, key string, src io.Reader) (applicationstorage.Location, error) {
	return s.putFn(ctx, key, src)
}

func (s stubStorage) Get(ctx context.Context, key string, loc applicationstorage.Location) (io.ReadCloser, error) {
	return s.getFn(ctx, key, loc)
}

func (s stubStorage) Delete(ctx context.Context, key string, loc applicationstorage.Location) error {
	return s.deleteFn(ctx, key, loc)
}

func (s stubStorage) Exists(context.Context, string, applicationstorage.Location) (bool, error) {
	panic("unexpected call")
}

func (s stubStorage) List(context.Context, string, applicationstorage.Location) ([]string, error) {
	panic("unexpected call")
}

func TestStorageAccessUsecasePutDelegates(t *testing.T) {
	u := NewStorageAccessUsecase(stubStorage{
		putFn: func(_ context.Context, key string, src io.Reader) (applicationstorage.Location, error) {
			body, err := io.ReadAll(src)
			if err != nil {
				t.Fatalf("ReadAll error = %v", err)
			}
			if key != "file.txt" {
				t.Fatalf("key = %q", key)
			}
			if string(body) != "payload" {
				t.Fatalf("body = %q", string(body))
			}
			return applicationstorage.LocationFallback, nil
		},
	})

	loc, err := u.Put(context.Background(), "file.txt", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("Put error = %v", err)
	}
	if loc != applicationstorage.LocationFallback {
		t.Fatalf("loc = %v", loc)
	}
}

func TestStorageAccessUsecaseGetDelegates(t *testing.T) {
	u := NewStorageAccessUsecase(stubStorage{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(_ context.Context, key string, loc applicationstorage.Location) (io.ReadCloser, error) {
			if key != "file.txt" || loc != applicationstorage.LocationPrimary {
				t.Fatalf("unexpected args: %q %v", key, loc)
			}
			return io.NopCloser(strings.NewReader("payload")), nil
		},
		deleteFn: func(context.Context, string, applicationstorage.Location) error { panic("unexpected call") },
	})

	rc, err := u.Get(context.Background(), "file.txt", applicationstorage.LocationPrimary)
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	defer rc.Close()

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}
	if string(body) != "payload" {
		t.Fatalf("body = %q", string(body))
	}
}

func TestStorageAccessUsecaseDeleteDelegates(t *testing.T) {
	deleteErr := errors.New("boom")
	u := NewStorageAccessUsecase(stubStorage{
		putFn: func(context.Context, string, io.Reader) (applicationstorage.Location, error) {
			panic("unexpected call")
		},
		getFn: func(context.Context, string, applicationstorage.Location) (io.ReadCloser, error) {
			panic("unexpected call")
		},
		deleteFn: func(_ context.Context, key string, loc applicationstorage.Location) error {
			if key != "file.txt" || loc != applicationstorage.LocationFallback {
				t.Fatalf("unexpected args: %q %v", key, loc)
			}
			return deleteErr
		},
	})

	err := u.Delete(context.Background(), "file.txt", applicationstorage.LocationFallback)
	if !errors.Is(err, deleteErr) {
		t.Fatalf("Delete error = %v", err)
	}
}
