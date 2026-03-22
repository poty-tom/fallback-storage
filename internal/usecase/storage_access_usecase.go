package usecase

import (
	"context"
	"io"

	applicationstorage "fallback-storage/internal/application/storage"
)

type StorageAccessUsecase struct {
	storage applicationstorage.Storage
}

func NewStorageAccessUsecase(storage applicationstorage.Storage) *StorageAccessUsecase {
	return &StorageAccessUsecase{storage: storage}
}

func (u *StorageAccessUsecase) Put(ctx context.Context, key string, src io.Reader) (applicationstorage.Location, error) {
	return u.storage.Put(ctx, key, src)
}

func (u *StorageAccessUsecase) Get(ctx context.Context, key string, loc applicationstorage.Location) (io.ReadCloser, error) {
	return u.storage.Get(ctx, key, loc)
}

func (u *StorageAccessUsecase) Delete(ctx context.Context, key string, loc applicationstorage.Location) error {
	return u.storage.Delete(ctx, key, loc)
}
