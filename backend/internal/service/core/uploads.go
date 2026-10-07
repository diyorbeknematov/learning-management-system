package core

import (
	"context"
	"mime"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/storage/minio"
	"github.com/diyorbeknematov/lms/pkg/apperror"
)

const megabyte = 1 << 20

// UploadRule says where files of one purpose live and which types are allowed,
// each with its largest size.
type UploadRule struct {
	Prefix       string
	ContentTypes map[string]int64
}

var imageTypes = func(max int64) map[string]int64 {
	return map[string]int64{
		"image/jpeg": max,
		"image/png":  max,
		"image/webp": max,
	}
}

var uploadRules = map[models.UploadPurpose]UploadRule{
	models.UploadPurposeAvatar: {
		Prefix:       "avatars/",
		ContentTypes: imageTypes(5 * megabyte),
	},
	models.UploadPurposeCourseCover: {
		Prefix:       "covers/",
		ContentTypes: imageTypes(10 * megabyte),
	},
	// documents and presentations of a lesson, and videos that are uploaded
	// instead of linked
	models.UploadPurposeMaterial: {
		Prefix: "materials/",
		ContentTypes: map[string]int64{
			"application/pdf":               100 * megabyte,
			"application/vnd.ms-powerpoint": 100 * megabyte,
			"application/vnd.openxmlformats-officedocument.presentationml.presentation": 100 * megabyte,
			"video/mp4":  500 * megabyte,
			"video/webm": 500 * megabyte,
		},
	},
}

// RuleFor returns the upload rule of a purpose.
func RuleFor(purpose models.UploadPurpose) (UploadRule, bool) {
	rule, ok := uploadRules[purpose]

	return rule, ok
}

// MaxUploadSize is the largest file the purpose accepts, of any allowed type.
func MaxUploadSize(purpose models.UploadPurpose) int64 {
	var largest int64

	for _, size := range uploadRules[purpose].ContentTypes {
		if size > largest {
			largest = size
		}
	}

	return largest
}

// BaseContentType lowercases a Content-Type and drops its parameters:
// "Image/PNG; charset=x" becomes "image/png".
func BaseContentType(contentType string) string {
	base, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(contentType))
	}

	return base
}

// CheckUpload checks a file the client says it uploaded: its key is in the
// folder of the purpose, the file exists, and its type and size are allowed. A
// file that breaks the rules is removed from the storage. The presigned URL
// cannot limit the size, so this is where the limit holds.
//
// A client uploads to the temporary folder ("tmp/avatars/<id>.png"). When such
// a file passes the checks it is copied to its permanent place
// ("avatars/<id>.png") and *objectKey is changed to the new key, which is the
// one to save. The temporary file is left to the lifecycle rule of the bucket,
// so a retry after a failed save still finds it. A key that is already
// permanent (the current avatar sent again) is checked as it is.
func CheckUpload(ctx context.Context, storage Storage, purpose models.UploadPurpose, objectKey *string, op string) (*minio.ObjectInfo, error) {
	invalid := func(message string) error {
		return apperror.InvalidInput("service", op, message, apperror.ErrInvalidInput)
	}

	rule, ok := uploadRules[purpose]
	if !ok {
		return nil, invalid("unknown upload purpose")
	}

	if objectKey == nil || strings.Contains(*objectKey, "..") {
		return nil, invalid("invalid file key")
	}

	temporary := strings.HasPrefix(*objectKey, minio.TempPrefix+rule.Prefix)

	if !temporary && !strings.HasPrefix(*objectKey, rule.Prefix) {
		return nil, invalid("invalid file key")
	}

	info, err := storage.Stat(ctx, *objectKey)
	if err != nil {
		if IsNotFound(err) {
			return nil, invalid("the file was not uploaded")
		}

		return nil, err
	}

	maxSize, allowed := rule.ContentTypes[BaseContentType(info.ContentType)]

	switch {
	case !allowed:
		_ = storage.Delete(ctx, *objectKey)

		return nil, invalid("this file type is not allowed")
	case info.Size > maxSize:
		_ = storage.Delete(ctx, *objectKey)

		return nil, invalid("the file is too large")
	}

	if temporary {
		permanent := strings.TrimPrefix(*objectKey, minio.TempPrefix)

		if err := storage.Copy(ctx, *objectKey, permanent); err != nil {
			return nil, err
		}

		*objectKey = permanent
	}

	return info, nil
}

// DownloadURL returns a temporary link to a stored file, or nil when there is no
// file.
func DownloadURL(ctx context.Context, storage Storage, objectKey *string) (*string, error) {
	if objectKey == nil || *objectKey == "" {
		return nil, nil
	}

	link, err := storage.PresignDownload(ctx, *objectKey)
	if err != nil {
		return nil, err
	}

	return &link, nil
}
