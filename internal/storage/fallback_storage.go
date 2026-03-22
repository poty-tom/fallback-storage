package storage

import (
	"bytes"
	"context"
	"errors"
	storageport "fallback-storage/internal/application/storage"
	"io"
)

// StorageClient は内部依存でのストレージクライアントのインターフェース
type StorageClient interface {
	Put(ctx context.Context, key string, content io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	List(ctx context.Context, prefix string) ([]string, error)
}

// 型アサーションで StorageClient が storageport.Storage を満たすことを確認する
var _ storageport.Storage = (*FallbackStorage)(nil)

// FallbackStorage は、フォールバックを前提に動作するストレージ実装
type FallbackStorage struct {
	primary  StorageClient
	fallback StorageClient
}

func NewFallbackStorage(primary, fallback StorageClient) *FallbackStorage {
	return &FallbackStorage{
		primary:  primary,
		fallback: fallback,
	}
}

var errInvalidLocation = errors.New("invalid storage location")

// Put は、primary ストレージに書き込みを行い、失敗した場合は fallback ストレージに書き込む
func (fs *FallbackStorage) Put(ctx context.Context, key string, content io.Reader) (storageport.Location, error) {
	body, err := io.ReadAll(content)
	if err != nil {
		return 0, err
	}

	if err := fs.primary.Put(ctx, key, bytes.NewReader(body)); err != nil {
		if fallbackErr := fs.fallback.Put(ctx, key, bytes.NewReader(body)); fallbackErr != nil {
			return 0, fallbackErr
		}
		return storageport.LocationFallback, nil
	}

	return storageport.LocationPrimary, nil
}

// Get は、loc で指定されたストレージから読み込む
func (fs *FallbackStorage) Get(ctx context.Context, key string, loc storageport.Location) (io.ReadCloser, error) {
	client, err := fs.clientFor(loc)
	if err != nil {
		return nil, err
	}

	return client.Get(ctx, key)
}

// Delete は、指定されたストレージから削除を行う
func (fs *FallbackStorage) Delete(ctx context.Context, key string, loc storageport.Location) error {
	client, err := fs.clientFor(loc)
	if err != nil {
		return err
	}

	return client.Delete(ctx, key)
}

// Exists は、指定されたストレージにキーが存在するかを確認する
func (fs *FallbackStorage) Exists(ctx context.Context, key string, loc storageport.Location) (bool, error) {
	client, err := fs.clientFor(loc)
	if err != nil {
		return false, err
	}

	return client.Exists(ctx, key)
}

// List は、指定されたストレージからプレフィックスにマッチするキーのリストを取得する
func (fs *FallbackStorage) List(ctx context.Context, prefix string, loc storageport.Location) ([]string, error) {
	client, err := fs.clientFor(loc)
	if err != nil {
		return nil, err
	}

	return client.List(ctx, prefix)
}

// clientFor は、Location に応じた StorageClient を返すヘルパーメソッド
func (fs *FallbackStorage) clientFor(loc storageport.Location) (StorageClient, error) {
	switch loc {
	case storageport.LocationPrimary:
		return fs.primary, nil
	case storageport.LocationFallback:
		return fs.fallback, nil
	default:
		return nil, errInvalidLocation
	}
}
