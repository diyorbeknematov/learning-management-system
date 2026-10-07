package handler

import (
	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/files"
	"github.com/gin-gonic/gin"
)

type Upload struct {
	svc *files.Upload
}

func NewUpload(svc *files.Upload) *Upload {
	return &Upload{
		svc: svc,
	}
}

func (h *Upload) Private(group *gin.RouterGroup) {
	group.POST("/uploads/presign", h.presign)
}

// presign gives the client an address to upload a file to. The client sends
// the file there (with the Content-Type it asked for), and then sends the key
// to the API where the file belongs: a profile, a course or a lesson.
//
// @Summary Get a presigned URL to upload a file
// @Description Access: any logged in user.
// @Tags Uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body models.PresignRequest true "Request body"
// @Success 200 {object} response.Success{data=models.PresignResponse}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /uploads/presign [post]
func (h *Upload) presign(c *gin.Context) {
	var req models.PresignRequest

	if !bindJSON(c, &req) {
		return
	}

	result, err := h.svc.Presign(c.Request.Context(), middleware.ActorFrom(c), req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, result)
}
