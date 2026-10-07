package handler

import (
	"net/http"

	"github.com/diyorbeknematov/lms/internal/api/middleware"
	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/learning"
	"github.com/gin-gonic/gin"
)

// Review has the routes of certificates and of course reviews.
type Review struct {
	certificates *learning.Certificate
	reviews      *learning.Review
}

func NewReview(certificates *learning.Certificate, reviews *learning.Review) *Review {
	return &Review{
		certificates: certificates,
		reviews:      reviews,
	}
}

// Public registers the check of a certificate and the reviews of a course.
// Anybody can check a certificate: that is what the QR code is for.
func (h *Review) Public(group *gin.RouterGroup) {
	group.GET("/certificates/verify/:uniqueId", h.verify)
	group.GET("/courses/:courseId/reviews", h.listReviews)
}

func (h *Review) Private(group *gin.RouterGroup) {
	group.GET("/certificates/me", h.myCertificates)
	group.GET("/certificates/:certificateId", h.getCertificate)
	group.GET("/certificates/:certificateId/download", h.downloadCertificate)

	group.POST("/courses/:courseId/reviews", h.createReview)
	group.PUT("/reviews/:reviewId", h.updateReview)
	group.DELETE("/reviews/:reviewId", h.deleteReview)
}

// @Summary Verify a certificate by its number (LMS-XXXX-XXXX-XXXX)
// @Description Access: anyone.
// @Tags Certificates
// @Produce json
// @Param uniqueId path string true "Certificate number"
// @Success 200 {object} response.Success{data=models.CertificateVerification}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /certificates/verify/{uniqueId} [get]
func (h *Review) verify(c *gin.Context) {
	result, err := h.certificates.Verify(c.Request.Context(), c.Param("uniqueId"))
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, result)
}

// @Summary List my certificates
// @Description Access: Student.
// @Tags Certificates
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Success{data=[]models.CertificateDetail}
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Router /certificates/me [get]
func (h *Review) myCertificates(c *gin.Context) {
	certificates, err := h.certificates.GetMyList(c.Request.Context(), middleware.ActorFrom(c))
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, certificates)
}

// @Summary Get a certificate
// @Description Access: owner or SuperAdmin.
// @Tags Certificates
// @Produce json
// @Security BearerAuth
// @Param certificateId path string true "Certificate id" format(uuid)
// @Success 200 {object} response.Success{data=models.CertificateDetail}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /certificates/{certificateId} [get]
func (h *Review) getCertificate(c *gin.Context) {
	id, ok := pathID(c, "certificateId")
	if !ok {
		return
	}

	certificate, err := h.certificates.GetByID(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, certificate)
}

// downloadCertificate sends the certificate as a PDF file.
//
// @Summary Download the certificate as a PDF
// @Description Access: owner or SuperAdmin.
// @Tags Certificates
// @Produce application/pdf
// @Security BearerAuth
// @Param certificateId path string true "Certificate id" format(uuid)
// @Success 200 {file} file "The PDF"
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /certificates/{certificateId}/download [get]
func (h *Review) downloadCertificate(c *gin.Context) {
	id, ok := pathID(c, "certificateId")
	if !ok {
		return
	}

	file, err := h.certificates.Download(c.Request.Context(), middleware.ActorFrom(c), id)
	if err != nil {
		response.Fail(c, err)

		return
	}

	c.Header("Content-Disposition", `attachment; filename="`+file.FileName+`"`)
	c.Data(http.StatusOK, "application/pdf", file.Content)
}

// @Summary List the reviews of a course
// @Description Access: anyone.
// @Tags Reviews
// @Produce json
// @Param courseId path string true "Course id" format(uuid)
// @Param query query models.ReviewFilter false "Query parameters"
// @Success 200 {object} response.Success{data=models.ReviewList}
// @Failure 400 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /courses/{courseId}/reviews [get]
func (h *Review) listReviews(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var filter models.ReviewFilter

	if !bindQuery(c, &filter) {
		return
	}

	reviews, err := h.reviews.GetListByCourse(c.Request.Context(), middleware.ActorFrom(c), courseID, filter)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, reviews)
}

// @Summary Review a course (needs an enrollment; one review per student)
// @Description Access: Student.
// @Tags Reviews
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param courseId path string true "Course id" format(uuid)
// @Param body body models.CreateReview true "Request body"
// @Success 201 {object} response.Success{data=models.Review}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Failure 409 {object} response.Failure
// @Router /courses/{courseId}/reviews [post]
func (h *Review) createReview(c *gin.Context) {
	courseID, ok := pathID(c, "courseId")
	if !ok {
		return
	}

	var req models.CreateReview

	if !bindJSON(c, &req) {
		return
	}

	review, err := h.reviews.Create(c.Request.Context(), middleware.ActorFrom(c), courseID, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.Created(c, review)
}

// @Summary Update my review
// @Description Access: Student (owner).
// @Tags Reviews
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param reviewId path string true "Review id" format(uuid)
// @Param body body models.UpdateReview true "Request body"
// @Success 200 {object} response.Success{data=models.Review}
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /reviews/{reviewId} [put]
func (h *Review) updateReview(c *gin.Context) {
	id, ok := pathID(c, "reviewId")
	if !ok {
		return
	}

	var req models.UpdateReview

	if !bindJSON(c, &req) {
		return
	}

	review, err := h.reviews.Update(c.Request.Context(), middleware.ActorFrom(c), id, req)
	if err != nil {
		response.Fail(c, err)

		return
	}

	response.OK(c, review)
}

// @Summary Delete a review
// @Description Access: owner or SuperAdmin.
// @Tags Reviews
// @Produce json
// @Security BearerAuth
// @Param reviewId path string true "Review id" format(uuid)
// @Success 204
// @Failure 400 {object} response.Failure
// @Failure 401 {object} response.Failure
// @Failure 403 {object} response.Failure
// @Failure 404 {object} response.Failure
// @Router /reviews/{reviewId} [delete]
func (h *Review) deleteReview(c *gin.Context) {
	id, ok := pathID(c, "reviewId")
	if !ok {
		return
	}

	if err := h.reviews.Delete(c.Request.Context(), middleware.ActorFrom(c), id); err != nil {
		response.Fail(c, err)

		return
	}

	response.NoContent(c)
}
