package files

import (
	"context"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/internal/storage/minio"
	"github.com/diyorbeknematov/lms/pkg/apperror"
)

type Upload struct {
	storage core.Storage
}

func NewUpload(deps core.Dependencies) *Upload {
	return &Upload{
		storage: deps.Storage,
	}
}

// Presign gives the client a URL to upload a file to directly, and the key to
// send back when the file is attached to a profile, a course or a lesson. The
// file goes to a temporary folder; it is moved to its place when it is attached
// and the bucket removes what is never attached.
// Everybody may upload an avatar; covers and lesson files are for instructors
// and the SuperAdmin.
func (s *Upload) Presign(ctx context.Context, actor models.Actor, req models.PresignRequest) (*models.PresignResponse, error) {
	invalid := func(message string) error {
		return apperror.InvalidInput("service", "Presign", message, apperror.ErrInvalidInput)
	}

	rule, ok := core.RuleFor(req.Purpose)
	if !ok {
		return nil, invalid("unknown upload purpose")
	}

	if req.Purpose != models.UploadPurposeAvatar && actor.RoleName == models.RoleStudent {
		return nil, apperror.Forbidden("service", "Presign", "students can upload only an avatar", apperror.ErrForbidden)
	}

	if _, allowed := rule.ContentTypes[core.BaseContentType(req.ContentType)]; !allowed {
		return nil, invalid("this file type is not allowed")
	}

	key := minio.TempPrefix + minio.NewObjectKey(req.Purpose, req.FileName)

	url, err := s.storage.PresignUpload(ctx, key)
	if err != nil {
		return nil, err
	}

	return &models.PresignResponse{
		UploadURL:   url,
		ObjectKey:   key,
		MaxFileSize: rule.ContentTypes[core.BaseContentType(req.ContentType)],
	}, nil
}
