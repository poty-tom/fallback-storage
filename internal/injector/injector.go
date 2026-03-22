package injector

import (
	"context"

	"fallback-storage/internal/storage"
	"fallback-storage/internal/storage/garage"
	"fallback-storage/internal/storage/local"
	"fallback-storage/internal/usecase"
)

type Config struct {
	GarageEndpoint        string
	GarageRegion          string
	GarageBucket          string
	GarageAccessKeyID     string
	GarageSecretAccessKey string
	GarageUsePathStyle    bool
	LocalRootDir          string
}

func BuildStorageAccessUsecase(ctx context.Context, cfg Config) (*usecase.StorageAccessUsecase, error) {
	primary, err := garage.NewGarageClient(ctx, garage.GarageConfig{
		Endpoint:        cfg.GarageEndpoint,
		Region:          cfg.GarageRegion,
		Bucket:          cfg.GarageBucket,
		AccessKeyID:     cfg.GarageAccessKeyID,
		SecretAccessKey: cfg.GarageSecretAccessKey,
		UsePathStyle:    cfg.GarageUsePathStyle,
	})
	if err != nil {
		return nil, err
	}

	fallback, err := local.NewLocalClient(local.LocalConfig{
		RootDir: cfg.LocalRootDir,
	})
	if err != nil {
		return nil, err
	}

	fallbackStorage := storage.NewFallbackStorage(primary, fallback)
	return usecase.NewStorageAccessUsecase(fallbackStorage), nil
}
