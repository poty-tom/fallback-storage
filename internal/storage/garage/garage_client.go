package garage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"fallback-storage/internal/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

var _ storage.StorageClient = (*GarageClient)(nil)

type GarageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type GarageClient struct {
	client s3Client
	bucket string
}

func NewGarageClient(ctx context.Context, cfg GarageConfig) (*GarageClient, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	return NewGarageClientWithS3(cfg, client)
}

func NewGarageClientWithS3(cfg GarageConfig, client s3Client) (*GarageClient, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("s3 client is required")
	}

	return &GarageClient{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (c *GarageClient) Put(ctx context.Context, key string, content io.Reader) error {
	key, err := normalizeKey(key)
	if err != nil {
		return err
	}

	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   content,
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}

	return nil
}

func (c *GarageClient) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	key, err := normalizeKey(key)
	if err != nil {
		return nil, err
	}

	out, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}

	return out.Body, nil
}

func (c *GarageClient) Delete(ctx context.Context, key string) error {
	key, err := normalizeKey(key)
	if err != nil {
		return err
	}

	_, err = c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object %q: %w", key, err)
	}

	return nil
}

func (c *GarageClient) Exists(ctx context.Context, key string) (bool, error) {
	key, err := normalizeKey(key)
	if err != nil {
		return false, err
	}

	_, err = c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err == nil {
		return true, nil
	}
	if isNotFoundError(err) {
		return false, nil
	}

	return false, fmt.Errorf("head object %q: %w", key, err)
}

func (c *GarageClient) List(ctx context.Context, prefix string) ([]string, error) {
	prefix, err := normalizePrefix(prefix)
	if err != nil {
		return nil, err
	}

	var keys []string
	var token *string
	for {
		out, listErr := c.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(c.bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if listErr != nil {
			return nil, fmt.Errorf("list objects %q: %w", prefix, listErr)
		}

		for _, obj := range out.Contents {
			if obj.Key == nil {
				continue
			}
			keys = append(keys, *obj.Key)
		}

		if !aws.ToBool(out.IsTruncated) || out.NextContinuationToken == nil {
			break
		}
		token = out.NextContinuationToken
	}

	sort.Strings(keys)
	return keys, nil
}

func (cfg GarageConfig) validate() error {
	switch {
	case strings.TrimSpace(cfg.Endpoint) == "":
		return errors.New("endpoint is required")
	case strings.TrimSpace(cfg.Region) == "":
		return errors.New("region is required")
	case strings.TrimSpace(cfg.Bucket) == "":
		return errors.New("bucket is required")
	case strings.TrimSpace(cfg.AccessKeyID) == "":
		return errors.New("access key id is required")
	case strings.TrimSpace(cfg.SecretAccessKey) == "":
		return errors.New("secret access key is required")
	default:
		return nil
	}
}

func normalizeKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("key is required")
	}
	return strings.TrimPrefix(key, "/"), nil
}

func normalizePrefix(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", nil
	}
	return strings.TrimPrefix(prefix, "/"), nil
}

func isNotFoundError(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}

	var apiErr interface{ ErrorCode() string }
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NotFound" || code == "NoSuchKey"
	}

	return false
}
