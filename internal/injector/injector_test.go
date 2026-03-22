package injector

import (
	"context"
	"testing"
)

func TestBuildStorageAccessUsecase(t *testing.T) {
	uc, err := BuildStorageAccessUsecase(context.Background(), Config{
		GarageEndpoint:        "http://127.0.0.1:3900",
		GarageRegion:          "garage",
		GarageBucket:          "bucket",
		GarageAccessKeyID:     "access",
		GarageSecretAccessKey: "secret",
		GarageUsePathStyle:    true,
		LocalRootDir:          t.TempDir(),
	})
	if err != nil {
		t.Fatalf("BuildStorageAccessUsecase error = %v", err)
	}
	if uc == nil {
		t.Fatal("usecase is nil")
	}
}

func TestBuildStorageAccessUsecaseRejectsInvalidConfig(t *testing.T) {
	_, err := BuildStorageAccessUsecase(context.Background(), Config{
		GarageRegion:          "garage",
		GarageBucket:          "bucket",
		GarageAccessKeyID:     "access",
		GarageSecretAccessKey: "secret",
		GarageUsePathStyle:    true,
		LocalRootDir:          t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
