package minio

import (
	"path"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/google/uuid"
)

var folders = map[models.UploadPurpose]string{
	models.UploadPurposeAvatar:      "avatars",
	models.UploadPurposeCourseCover: "covers",
	models.UploadPurposeMaterial:    "materials",
}

// NewObjectKey returns a unique key such as "avatars/<uuid>.png". The client's
// file name is never used in the key, only its extension, so two uploads can
// not overwrite each other and odd names cannot escape the folder.
func NewObjectKey(purpose models.UploadPurpose, fileName string) string {
	folder, ok := folders[purpose]
	if !ok {
		folder = "other"
	}

	return folder + "/" + uuid.NewString() + safeExtension(fileName)
}

// safeExtension keeps a short, lowercase, alphanumeric extension and drops
// anything else.
func safeExtension(fileName string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(fileName), "."))

	if ext == "" || len(ext) > 10 {
		return ""
	}

	for _, r := range ext {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ""
		}
	}

	return "." + ext
}
