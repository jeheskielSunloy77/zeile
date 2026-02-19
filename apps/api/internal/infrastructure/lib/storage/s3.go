package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jeheskielSunloy77/zeile/internal/infrastructure/config"
)

type S3Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
}

func NewS3Storage(cfg config.FileStorageS3Config) (*S3Storage, error) {
	ctx := context.Background()
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		creds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")
		loadOptions = append(loadOptions, awsconfig.WithCredentialsProvider(creds))
	}

	if cfg.Endpoint != "" {
		endpoint := cfg.Endpoint
		loadOptions = append(loadOptions, awsconfig.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
				if service == s3.ServiceID {
					return aws.Endpoint{URL: endpoint, SigningRegion: cfg.Region}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
	})

	publicURL := strings.TrimRight(cfg.PublicURL, "/")
	if publicURL == "" && cfg.Bucket != "" && cfg.Region != "" {
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
	}

	return &S3Storage{client: client, bucket: cfg.Bucket, publicURL: publicURL}, nil
}

func (s *S3Storage) Save(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*Object, error) {
	cleanKey := strings.TrimLeft(path.Clean("/"+key), "/")
	input := &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &cleanKey,
		Body:   reader,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	if size > 0 {
		input.ContentLength = size
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return nil, err
	}

	url := cleanKey
	if s.publicURL != "" {
		url = strings.TrimRight(s.publicURL, "/") + "/" + cleanKey
	}

	return &Object{Path: cleanKey, URL: url, Size: size}, nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	cleanKey := strings.TrimLeft(path.Clean("/"+key), "/")
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &cleanKey,
	})
	return err
}
