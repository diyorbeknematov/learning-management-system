package minio

import (
	"fmt"

	"github.com/diyorbeknematov/lms/internal/config"
	clientMinIO "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIO struct {
	Client *clientMinIO.Client
	Bucket string
}

func New(cfg config.MinIOConfig) (*MinIO, error) {
	client, err := clientMinIO.New(
		cfg.Endpoint,
		&clientMinIO.Options{
			Creds: credentials.NewStaticV4(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			),
			Secure: cfg.UseSSL,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &MinIO{
		Client: client,
		Bucket: cfg.Bucket,
	}, nil
}
