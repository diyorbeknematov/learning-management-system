package files_test

import (
	"context"
	"strings"
	"testing"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/stretchr/testify/require"
)

func presign(e *testutil.Env, actor models.Actor, purpose models.UploadPurpose, fileName, contentType string) (*models.PresignResponse, error) {
	return e.Svc.Upload.Presign(context.Background(), actor, models.PresignRequest{
		Purpose:     purpose,
		FileName:    fileName,
		ContentType: contentType,
	})
}

func TestUpload_Presign_Avatar(t *testing.T) {
	e := testutil.Setup(t)

	student := testutil.ActorOf(e.NewUser(t, models.RoleStudent))

	response, err := presign(e, student, models.UploadPurposeAvatar, "me.PNG", "image/png")
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(response.ObjectKey, "tmp/avatars/"))
	require.True(t, strings.HasSuffix(response.ObjectKey, ".png"))
	require.Equal(t, "http://files.test/upload/"+response.ObjectKey, response.UploadURL)
	require.Equal(t, int64(5<<20), response.MaxFileSize)
}

func TestUpload_Presign_KeysAreUnique(t *testing.T) {
	e := testutil.Setup(t)

	student := testutil.ActorOf(e.NewUser(t, models.RoleStudent))

	first, err := presign(e, student, models.UploadPurposeAvatar, "me.png", "image/png")
	require.NoError(t, err)

	second, err := presign(e, student, models.UploadPurposeAvatar, "me.png", "image/png")
	require.NoError(t, err)

	require.NotEqual(t, first.ObjectKey, second.ObjectKey)
}

func TestUpload_Presign_OnlyInstructorsUploadCoversAndMaterials(t *testing.T) {
	e := testutil.Setup(t)

	student := testutil.ActorOf(e.NewUser(t, models.RoleStudent))
	instructor := testutil.ActorOf(e.NewUser(t, models.RoleInstructor))

	_, err := presign(e, student, models.UploadPurposeCourseCover, "c.png", "image/png")
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	_, err = presign(e, student, models.UploadPurposeMaterial, "s.pdf", "application/pdf")
	testutil.RequireCode(t, err, apperror.CodeForbidden)

	cover, err := presign(e, instructor, models.UploadPurposeCourseCover, "c.png", "image/png")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(cover.ObjectKey, "tmp/covers/"))
	require.Equal(t, int64(10<<20), cover.MaxFileSize)

	material, err := presign(e, testutil.AdminActor(), models.UploadPurposeMaterial, "s.pdf", "application/pdf")
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(material.ObjectKey, "tmp/materials/"))
}

func TestUpload_Presign_AllowedTypes(t *testing.T) {
	e := testutil.Setup(t)

	instructor := testutil.ActorOf(e.NewUser(t, models.RoleInstructor))

	cases := []struct {
		purpose     models.UploadPurpose
		contentType string
		maxSize     int64
	}{
		{models.UploadPurposeAvatar, "image/jpeg", 5 << 20},
		{models.UploadPurposeAvatar, "image/webp", 5 << 20},
		{models.UploadPurposeCourseCover, "image/png", 10 << 20},
		{models.UploadPurposeMaterial, "application/pdf", 100 << 20},
		{models.UploadPurposeMaterial, "application/vnd.ms-powerpoint", 100 << 20},
		{models.UploadPurposeMaterial, "application/vnd.openxmlformats-officedocument.presentationml.presentation", 100 << 20},
		{models.UploadPurposeMaterial, "video/mp4", 500 << 20},
		{models.UploadPurposeMaterial, "video/webm", 500 << 20},
		{models.UploadPurposeMaterial, "Application/PDF; charset=binary", 100 << 20},
	}

	for _, tt := range cases {
		response, err := presign(e, instructor, tt.purpose, "file", tt.contentType)
		require.NoError(t, err, tt.contentType)
		require.Equal(t, tt.maxSize, response.MaxFileSize, tt.contentType)
	}
}

func TestUpload_Presign_RejectedTypes(t *testing.T) {
	e := testutil.Setup(t)

	instructor := testutil.ActorOf(e.NewUser(t, models.RoleInstructor))

	cases := []struct {
		purpose     models.UploadPurpose
		contentType string
	}{
		{models.UploadPurposeAvatar, "application/pdf"},
		{models.UploadPurposeAvatar, "image/svg+xml"},
		{models.UploadPurposeAvatar, "video/mp4"},
		{models.UploadPurposeCourseCover, "image/gif"},
		{models.UploadPurposeMaterial, "image/png"},
		{models.UploadPurposeMaterial, "text/html"},
		{models.UploadPurposeMaterial, "application/x-msdownload"},
		{models.UploadPurposeMaterial, ""},
	}

	for _, tt := range cases {
		_, err := presign(e, instructor, tt.purpose, "file", tt.contentType)
		testutil.RequireCode(t, err, apperror.CodeInvalidInput)
	}
}

func TestUpload_Presign_UnknownPurpose(t *testing.T) {
	e := testutil.Setup(t)

	_, err := presign(e, testutil.AdminActor(), models.UploadPurpose("backup"), "a.png", "image/png")
	testutil.RequireCode(t, err, apperror.CodeInvalidInput)
}

func TestUpload_Presign_TheClientNameIsNeverInTheKey(t *testing.T) {
	e := testutil.Setup(t)

	response, err := presign(e, testutil.ActorOf(e.NewUser(t, models.RoleStudent)), models.UploadPurposeAvatar, "../../etc/passwd.png", "image/png")
	require.NoError(t, err)

	require.NotContains(t, response.ObjectKey, "..")
	require.NotContains(t, response.ObjectKey, "passwd")
	require.True(t, strings.HasSuffix(response.ObjectKey, ".png"))
}
