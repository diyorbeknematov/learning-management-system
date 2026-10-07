package models

type UploadPurpose string

const (
	UploadPurposeAvatar      UploadPurpose = "avatar"
	UploadPurposeCourseCover UploadPurpose = "course_cover"
	UploadPurposeMaterial    UploadPurpose = "material"
)

type PresignRequest struct {
	Purpose     UploadPurpose `json:"purpose" validate:"required,oneof=avatar course_cover material"`
	FileName    string        `json:"file_name" validate:"required"`
	ContentType string        `json:"content_type" validate:"required"`
}

// PresignResponse tells the client where to PUT the file. The client must send
// the same Content-Type it asked for, and the file may not be larger than
// MaxFileSize.
type PresignResponse struct {
	UploadURL   string `json:"upload_url"`
	ObjectKey   string `json:"object_key"`
	MaxFileSize int64  `json:"max_file_size"`
}
