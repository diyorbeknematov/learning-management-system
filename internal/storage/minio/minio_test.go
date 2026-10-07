package minio_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/storage/minio"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	clientMinIO "github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

// setupMinIO connects to the MinIO from the environment with a temporary
// bucket. The test is skipped when MINIO_ACCESS_KEY is not set.
func setupMinIO(t *testing.T) *minio.MinIO {
	t.Helper()

	cfg := config.Load().MinIO

	if cfg.AccessKey == "" {
		t.Skip("MINIO_ACCESS_KEY is not set")
	}

	cfg.Bucket = "test-" + uuid.NewString()
	cfg.PresignedExpiry = time.Minute

	m, err := minio.New(cfg)
	require.NoError(t, err)

	t.Cleanup(func() {
		ctx := context.Background()

		for object := range m.Client.ListObjects(ctx, m.Bucket, clientMinIO.ListObjectsOptions{Recursive: true}) {
			_ = m.Client.RemoveObject(ctx, m.Bucket, object.Key, clientMinIO.RemoveObjectOptions{})
		}

		_ = m.Client.RemoveBucket(ctx, m.Bucket)
	})

	return m
}

func TestNew_CreatesBucket(t *testing.T) {
	m := setupMinIO(t)

	exists, err := m.Client.BucketExists(context.Background(), m.Bucket)

	require.NoError(t, err)
	require.True(t, exists)
}

func TestNew_ExistingBucket(t *testing.T) {
	m := setupMinIO(t)

	cfg := config.Load().MinIO
	cfg.Bucket = m.Bucket

	_, err := minio.New(cfg)
	require.NoError(t, err)
}

func TestUploadDownloadDelete(t *testing.T) {
	m := setupMinIO(t)

	ctx := context.Background()
	key := minio.NewObjectKey(models.UploadPurposeMaterial, "notes.txt")
	content := []byte("hello minio")

	uploadURL, err := m.PresignUpload(ctx, key)
	require.NoError(t, err)

	request, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(content))
	require.NoError(t, err)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)

	info, err := m.Stat(ctx, key)
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), info.Size)

	downloadURL, err := m.PresignDownload(ctx, key)
	require.NoError(t, err)

	response, err = http.Get(downloadURL)
	require.NoError(t, err)

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, content, body)

	require.NoError(t, m.Delete(ctx, key))

	_, err = m.Stat(ctx, key)
	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestStat_NotFound(t *testing.T) {
	m := setupMinIO(t)

	_, err := m.Stat(context.Background(), "avatars/missing.png")

	appErr, ok := apperror.As(err)
	require.True(t, ok)
	require.Equal(t, apperror.CodeNotFound, appErr.Code)
}

func TestDelete_Missing(t *testing.T) {
	m := setupMinIO(t)

	require.NoError(t, m.Delete(context.Background(), "avatars/missing.png"))
}
