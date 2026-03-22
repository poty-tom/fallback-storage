package storage

import (
	"bytes"
	"context"
	"errors"
	storageport "fallback-storage/internal/application/storage"
	"io"
	"slices"
	"strings"
	"testing"
)

type stubStorageClient struct {
	putErr    error
	getErr    error
	deleteErr error
	existsErr error
	listErr   error

	getBody    string
	exists     bool
	listResult []string

	putCalls    int
	getCalls    int
	deleteCalls int
	existsCalls int
	listCalls   int

	lastPutBody string
}

func (s *stubStorageClient) Put(_ context.Context, _ string, content io.Reader) error {
	s.putCalls++
	body, err := io.ReadAll(content)
	if err != nil {
		return err
	}
	s.lastPutBody = string(body)
	return s.putErr
}

func (s *stubStorageClient) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	return io.NopCloser(strings.NewReader(s.getBody)), nil
}

func (s *stubStorageClient) Delete(_ context.Context, _ string) error {
	s.deleteCalls++
	return s.deleteErr
}

func (s *stubStorageClient) Exists(_ context.Context, _ string) (bool, error) {
	s.existsCalls++
	if s.existsErr != nil {
		return false, s.existsErr
	}
	return s.exists, nil
}

func (s *stubStorageClient) List(_ context.Context, _ string) ([]string, error) {
	s.listCalls++
	if s.listErr != nil {
		return nil, s.listErr
	}
	return slices.Clone(s.listResult), nil
}

func TestFallbackStoragePutUsesPrimaryOnSuccess(t *testing.T) {
	primary := &stubStorageClient{}
	fallback := &stubStorageClient{}
	fs := NewFallbackStorage(primary, fallback)

	loc, err := fs.Put(context.Background(), "key", strings.NewReader("payload"))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	if loc != storageport.LocationPrimary {
		t.Fatalf("Put location = %v, want %v", loc, storageport.LocationPrimary)
	}
	if primary.putCalls != 1 {
		t.Fatalf("primary put calls = %d, want 1", primary.putCalls)
	}
	if fallback.putCalls != 0 {
		t.Fatalf("fallback put calls = %d, want 0", fallback.putCalls)
	}
	if primary.lastPutBody != "payload" {
		t.Fatalf("primary body = %q, want %q", primary.lastPutBody, "payload")
	}
}

func TestFallbackStoragePutFallsBackWithSameBody(t *testing.T) {
	primary := &stubStorageClient{putErr: errors.New("primary unavailable")}
	fallback := &stubStorageClient{}
	fs := NewFallbackStorage(primary, fallback)

	loc, err := fs.Put(context.Background(), "key", bytes.NewBufferString("payload"))
	if err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	if loc != storageport.LocationFallback {
		t.Fatalf("Put location = %v, want %v", loc, storageport.LocationFallback)
	}
	if primary.putCalls != 1 {
		t.Fatalf("primary put calls = %d, want 1", primary.putCalls)
	}
	if fallback.putCalls != 1 {
		t.Fatalf("fallback put calls = %d, want 1", fallback.putCalls)
	}
	if fallback.lastPutBody != "payload" {
		t.Fatalf("fallback body = %q, want %q", fallback.lastPutBody, "payload")
	}
}

func TestFallbackStoragePutReturnsFallbackError(t *testing.T) {
	primary := &stubStorageClient{putErr: errors.New("primary unavailable")}
	fallbackErr := errors.New("fallback unavailable")
	fallback := &stubStorageClient{putErr: fallbackErr}
	fs := NewFallbackStorage(primary, fallback)

	loc, err := fs.Put(context.Background(), "key", strings.NewReader("payload"))
	if !errors.Is(err, fallbackErr) {
		t.Fatalf("Put error = %v, want %v", err, fallbackErr)
	}
	if loc != 0 {
		t.Fatalf("Put location = %v, want 0", loc)
	}
}

func TestFallbackStorageRoutesByLocation(t *testing.T) {
	primary := &stubStorageClient{
		getBody:    "primary-data",
		exists:     true,
		listResult: []string{"a", "b"},
	}
	fallback := &stubStorageClient{
		getBody:    "fallback-data",
		exists:     true,
		listResult: []string{"c"},
	}
	fs := NewFallbackStorage(primary, fallback)

	rc, err := fs.Get(context.Background(), "key", storageport.LocationFallback)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if string(body) != "fallback-data" {
		t.Fatalf("Get body = %q, want %q", string(body), "fallback-data")
	}

	if err := fs.Delete(context.Background(), "key", storageport.LocationPrimary); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	exists, err := fs.Exists(context.Background(), "key", storageport.LocationPrimary)
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}
	if !exists {
		t.Fatalf("Exists = false, want true")
	}

	keys, err := fs.List(context.Background(), "prefix", storageport.LocationFallback)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if !slices.Equal(keys, []string{"c"}) {
		t.Fatalf("List = %v, want %v", keys, []string{"c"})
	}

	if primary.getCalls != 0 {
		t.Fatalf("primary get calls = %d, want 0", primary.getCalls)
	}
	if fallback.getCalls != 1 {
		t.Fatalf("fallback get calls = %d, want 1", fallback.getCalls)
	}
	if primary.deleteCalls != 1 {
		t.Fatalf("primary delete calls = %d, want 1", primary.deleteCalls)
	}
	if fallback.deleteCalls != 0 {
		t.Fatalf("fallback delete calls = %d, want 0", fallback.deleteCalls)
	}
	if primary.existsCalls != 1 {
		t.Fatalf("primary exists calls = %d, want 1", primary.existsCalls)
	}
	if fallback.existsCalls != 0 {
		t.Fatalf("fallback exists calls = %d, want 0", fallback.existsCalls)
	}
	if primary.listCalls != 0 {
		t.Fatalf("primary list calls = %d, want 0", primary.listCalls)
	}
	if fallback.listCalls != 1 {
		t.Fatalf("fallback list calls = %d, want 1", fallback.listCalls)
	}
}

func TestFallbackStorageRejectsInvalidLocation(t *testing.T) {
	fs := NewFallbackStorage(&stubStorageClient{}, &stubStorageClient{})

	if _, err := fs.Get(context.Background(), "key", storageport.Location(99)); !errors.Is(err, errInvalidLocation) {
		t.Fatalf("Get error = %v, want %v", err, errInvalidLocation)
	}
	if err := fs.Delete(context.Background(), "key", storageport.Location(99)); !errors.Is(err, errInvalidLocation) {
		t.Fatalf("Delete error = %v, want %v", err, errInvalidLocation)
	}
	if _, err := fs.Exists(context.Background(), "key", storageport.Location(99)); !errors.Is(err, errInvalidLocation) {
		t.Fatalf("Exists error = %v, want %v", err, errInvalidLocation)
	}
	if _, err := fs.List(context.Background(), "prefix", storageport.Location(99)); !errors.Is(err, errInvalidLocation) {
		t.Fatalf("List error = %v, want %v", err, errInvalidLocation)
	}
}
