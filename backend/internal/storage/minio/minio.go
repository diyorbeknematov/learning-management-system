package minio

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	clientMinIO "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

// TempPrefix is the folder where the client uploads files first. A file is
// copied out of it when it is attached to a profile, a course or a lesson, and
// what stays is removed by a lifecycle rule of the bucket.
const TempPrefix = "tmp/"

// tempLifetime is how long a file in TempPrefix lives. It is longer than the
// time a presigned upload URL is valid, so an upload is never cut short.
const tempLifetime = 1

// region is fixed so that presigning is computed locally, without asking the
// server for the bucket region first.
const region = "us-east-1"

type MinIO struct {
	Client *clientMinIO.Client
	// presigner makes the links for clients. A link is signed for one host, so
	// it has to be made for the address the browser uses, which can differ from
	// the address of the API (see config.MinIOConfig.PublicEndpoint).
	presigner       *clientMinIO.Client
	Bucket          string
	PresignedExpiry time.Duration
}

func newClient(cfg config.MinIOConfig, endpoint string) (*clientMinIO.Client, error) {
	client, err := clientMinIO.New(
		endpoint,
		&clientMinIO.Options{
			Creds: credentials.NewStaticV4(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			),
			Secure: cfg.UseSSL,
			Region: region,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return client, nil
}

// New connects to MinIO and creates the bucket if it does not exist yet.
func New(cfg config.MinIOConfig) (*MinIO, error) {
	client, err := newClient(cfg, cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	presigner := client

	if cfg.PublicEndpoint != "" && cfg.PublicEndpoint != cfg.Endpoint {
		// no request is sent with this client: presigning is computed locally
		presigner, err = newClient(cfg, cfg.PublicEndpoint)
		if err != nil {
			return nil, err
		}
	}

	m := &MinIO{
		Client:          client,
		presigner:       presigner,
		Bucket:          cfg.Bucket,
		PresignedExpiry: cfg.PresignedExpiry,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := m.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *MinIO) ensureBucket(ctx context.Context) error {
	exists, err := m.Client.BucketExists(ctx, m.Bucket)
	if err != nil {
		return fmt.Errorf("check minio bucket: %w", err)
	}

	if !exists {
		err = m.Client.MakeBucket(ctx, m.Bucket, clientMinIO.MakeBucketOptions{Region: region})
		if err != nil {
			return fmt.Errorf("create minio bucket: %w", err)
		}
	}

	return m.ensureTempLifecycle(ctx)
}

// ensureTempLifecycle makes the bucket remove the files of TempPrefix after
// tempLifetime days: uploads that were never attached to anything.
func (m *MinIO) ensureTempLifecycle(ctx context.Context) error {
	rules := lifecycle.NewConfiguration()
	rules.Rules = []lifecycle.Rule{{
		ID:         "expire-unattached-uploads",
		Status:     "Enabled",
		RuleFilter: lifecycle.Filter{Prefix: TempPrefix},
		Expiration: lifecycle.Expiration{Days: tempLifetime},
	}}

	if err := m.Client.SetBucketLifecycle(ctx, m.Bucket, rules); err != nil {
		return fmt.Errorf("set minio lifecycle: %w", err)
	}

	return nil
}

// PresignUpload returns a URL the client can PUT the file to directly. It is
// valid for PresignedExpiry.
func (m *MinIO) PresignUpload(ctx context.Context, objectKey string) (string, error) {
	u, err := m.presigner.PresignedPutObject(ctx, m.Bucket, objectKey, m.PresignedExpiry)
	if err != nil {
		return "", apperror.Internal(
			"storage",
			"PresignUpload",
			"failed to presign upload url",
			err,
		)
	}

	return u.String(), nil
}

// PresignDownload returns a URL the client can GET the file from. It is valid
// for PresignedExpiry.
func (m *MinIO) PresignDownload(ctx context.Context, objectKey string) (string, error) {
	u, err := m.presigner.PresignedGetObject(ctx, m.Bucket, objectKey, m.PresignedExpiry, url.Values{})
	if err != nil {
		return "", apperror.Internal(
			"storage",
			"PresignDownload",
			"failed to presign download url",
			err,
		)
	}

	return u.String(), nil
}

type ObjectInfo struct {
	Size        int64
	ContentType string
}

// Stat checks that a file was really uploaded and returns its size and type.
func (m *MinIO) Stat(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	info, err := m.Client.StatObject(ctx, m.Bucket, objectKey, clientMinIO.StatObjectOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil, apperror.NotFound(
				"storage",
				"Stat",
				"file not found",
				apperror.ErrNotFound,
			)
		}

		return nil, apperror.Internal(
			"storage",
			"Stat",
			"failed to get file info",
			err,
		)
	}

	return &ObjectInfo{
		Size:        info.Size,
		ContentType: info.ContentType,
	}, nil
}

// Copy copies a file inside the bucket. An existing destination is replaced.
func (m *MinIO) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	_, err := m.Client.CopyObject(
		ctx,
		clientMinIO.CopyDestOptions{Bucket: m.Bucket, Object: destinationKey},
		clientMinIO.CopySrcOptions{Bucket: m.Bucket, Object: sourceKey},
	)
	if err != nil {
		return apperror.Internal(
			"storage",
			"Copy",
			"failed to copy file",
			err,
		)
	}

	return nil
}

// Delete removes a file. Deleting a file that does not exist is not an error.
func (m *MinIO) Delete(ctx context.Context, objectKey string) error {
	err := m.Client.RemoveObject(ctx, m.Bucket, objectKey, clientMinIO.RemoveObjectOptions{})
	if err != nil {
		return apperror.Internal(
			"storage",
			"Delete",
			"failed to delete file",
			err,
		)
	}

	return nil
}

func isNotFound(err error) bool {
	response := clientMinIO.ToErrorResponse(err)

	return response.Code == "NoSuchKey" || response.Code == "NotFound"
}
