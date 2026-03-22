package garage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type stubS3Client struct {
	putObjectFn     func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFn     func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	deleteObjectFn  func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	headObjectFn    func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	listObjectsV2Fn func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

func (s stubS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return s.putObjectFn(ctx, params, optFns...)
}

func (s stubS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return s.getObjectFn(ctx, params, optFns...)
}

func (s stubS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return s.deleteObjectFn(ctx, params, optFns...)
}

func (s stubS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return s.headObjectFn(ctx, params, optFns...)
}

func (s stubS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return s.listObjectsV2Fn(ctx, params, optFns...)
}

func TestNewGarageClientWithS3RejectsInvalidConfig(t *testing.T) {
	_, err := NewGarageClientWithS3(GarageConfig{}, stubS3Client{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGarageClientPut(t *testing.T) {
	client, err := NewGarageClientWithS3(validConfig(), stubS3Client{
		putObjectFn: func(_ context.Context, in *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			body, readErr := io.ReadAll(in.Body)
			if readErr != nil {
				t.Fatalf("ReadAll error = %v", readErr)
			}
			if aws.ToString(in.Bucket) != "bucket" {
				t.Fatalf("bucket = %q", aws.ToString(in.Bucket))
			}
			if aws.ToString(in.Key) != "path/file.txt" {
				t.Fatalf("key = %q", aws.ToString(in.Key))
			}
			if string(body) != "payload" {
				t.Fatalf("body = %q", string(body))
			}
			return &s3.PutObjectOutput{}, nil
		},
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			panic("unexpected call")
		},
		deleteObjectFn: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			panic("unexpected call")
		},
		headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			panic("unexpected call")
		},
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			panic("unexpected call")
		},
	})
	if err != nil {
		t.Fatalf("NewGarageClientWithS3 error = %v", err)
	}

	if err := client.Put(context.Background(), "path/file.txt", strings.NewReader("payload")); err != nil {
		t.Fatalf("Put error = %v", err)
	}
}

func TestGarageClientGet(t *testing.T) {
	client, err := NewGarageClientWithS3(validConfig(), stubS3Client{
		putObjectFn: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			panic("unexpected call")
		},
		deleteObjectFn: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			panic("unexpected call")
		},
		headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			panic("unexpected call")
		},
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			panic("unexpected call")
		},
		getObjectFn: func(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			if aws.ToString(in.Key) != "path/file.txt" {
				t.Fatalf("key = %q", aws.ToString(in.Key))
			}
			return &s3.GetObjectOutput{
				Body: io.NopCloser(bytes.NewReader([]byte("payload"))),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewGarageClientWithS3 error = %v", err)
	}

	rc, err := client.Get(context.Background(), "path/file.txt")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll error = %v", err)
	}
	if string(body) != "payload" {
		t.Fatalf("body = %q", string(body))
	}
}

func TestGarageClientDelete(t *testing.T) {
	client, err := NewGarageClientWithS3(validConfig(), stubS3Client{
		putObjectFn: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			panic("unexpected call")
		},
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			panic("unexpected call")
		},
		headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			panic("unexpected call")
		},
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			panic("unexpected call")
		},
		deleteObjectFn: func(_ context.Context, in *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			if aws.ToString(in.Key) != "path/file.txt" {
				t.Fatalf("key = %q", aws.ToString(in.Key))
			}
			return &s3.DeleteObjectOutput{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewGarageClientWithS3 error = %v", err)
	}

	if err := client.Delete(context.Background(), "path/file.txt"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
}

func TestGarageClientExistsNormalizesNotFound(t *testing.T) {
	notFoundErr := &types.NoSuchKey{}
	client, err := NewGarageClientWithS3(validConfig(), stubS3Client{
		putObjectFn: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			panic("unexpected call")
		},
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			panic("unexpected call")
		},
		deleteObjectFn: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			panic("unexpected call")
		},
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			panic("unexpected call")
		},
		headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			return nil, notFoundErr
		},
	})
	if err != nil {
		t.Fatalf("NewGarageClientWithS3 error = %v", err)
	}

	exists, err := client.Exists(context.Background(), "path/file.txt")
	if err != nil {
		t.Fatalf("Exists error = %v", err)
	}
	if exists {
		t.Fatal("Exists = true, want false")
	}
}

func TestGarageClientListSortsAcrossPages(t *testing.T) {
	callCount := 0
	client, err := NewGarageClientWithS3(validConfig(), stubS3Client{
		putObjectFn: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			panic("unexpected call")
		},
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			panic("unexpected call")
		},
		deleteObjectFn: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			panic("unexpected call")
		},
		headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			panic("unexpected call")
		},
		listObjectsV2Fn: func(_ context.Context, in *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			callCount++
			if aws.ToString(in.Prefix) != "prefix/" {
				t.Fatalf("prefix = %q", aws.ToString(in.Prefix))
			}
			if callCount == 1 {
				return &s3.ListObjectsV2Output{
					Contents:              []types.Object{{Key: aws.String("prefix/b.txt")}},
					IsTruncated:           aws.Bool(true),
					NextContinuationToken: aws.String("page-2"),
				}, nil
			}
			if aws.ToString(in.ContinuationToken) != "page-2" {
				t.Fatalf("continuation token = %q", aws.ToString(in.ContinuationToken))
			}
			return &s3.ListObjectsV2Output{
				Contents: []types.Object{{Key: aws.String("prefix/a.txt")}},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewGarageClientWithS3 error = %v", err)
	}

	keys, err := client.List(context.Background(), "prefix/")
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	if len(keys) != 2 || keys[0] != "prefix/a.txt" || keys[1] != "prefix/b.txt" {
		t.Fatalf("keys = %v", keys)
	}
}

func TestGarageClientExistsReturnsUnexpectedErrors(t *testing.T) {
	headErr := errors.New("boom")
	client, err := NewGarageClientWithS3(validConfig(), stubS3Client{
		putObjectFn: func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			panic("unexpected call")
		},
		getObjectFn: func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			panic("unexpected call")
		},
		deleteObjectFn: func(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			panic("unexpected call")
		},
		listObjectsV2Fn: func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			panic("unexpected call")
		},
		headObjectFn: func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
			return nil, headErr
		},
	})
	if err != nil {
		t.Fatalf("NewGarageClientWithS3 error = %v", err)
	}

	_, err = client.Exists(context.Background(), "path/file.txt")
	if !errors.Is(err, headErr) {
		t.Fatalf("Exists error = %v, want %v", err, headErr)
	}
}

func validConfig() GarageConfig {
	return GarageConfig{
		Endpoint:        "http://127.0.0.1:3900",
		Region:          "garage",
		Bucket:          "bucket",
		AccessKeyID:     "access",
		SecretAccessKey: "secret",
		UsePathStyle:    true,
	}
}
