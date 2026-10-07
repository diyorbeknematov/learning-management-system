package learning

import (
	"context"
	"strings"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/certpdf"
	"github.com/google/uuid"
)

// Certificate only reads certificates. They are issued by itself, at the
// moment a student completes a course (see syncCompletion).
type Certificate struct {
	repo *repo.Repository
	cfg  core.Config
}

func NewCertificate(deps core.Dependencies) *Certificate {
	return &Certificate{
		repo: deps.Repo,
		cfg:  deps.Config,
	}
}

// GetMyList returns the certificates of the student, the newest first.
func (s *Certificate) GetMyList(ctx context.Context, actor models.Actor) ([]models.CertificateDetail, error) {
	return s.repo.Certificate.GetListByStudentID(ctx, actor.UserID)
}

// GetByID returns a certificate to its student, the owner of the course and
// the SuperAdmin.
func (s *Certificate) GetByID(ctx context.Context, actor models.Actor, id uuid.UUID) (*models.CertificateDetail, error) {
	certificate, err := s.repo.Certificate.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if certificate.StudentID == actor.UserID {
		return certificate, nil
	}

	course, err := s.repo.Course.GetByID(ctx, certificate.CourseID)
	if err != nil {
		return nil, err
	}

	if err := core.CanManage(actor, course.InstructorID, "GetCertificate"); err != nil {
		return nil, err
	}

	return certificate, nil
}

// CertificateFile is a certificate drawn as a PDF file, ready to download.
type CertificateFile struct {
	FileName string
	Content  []byte
}

// Download draws the certificate as a PDF with a QR code that leads to the
// public check. It is drawn when it is asked for and not kept. Who may see the
// certificate may download it.
func (s *Certificate) Download(ctx context.Context, actor models.Actor, id uuid.UUID) (*CertificateFile, error) {
	certificate, err := s.GetByID(ctx, actor, id)
	if err != nil {
		return nil, err
	}

	verifyURL := strings.TrimRight(s.cfg.CertificateVerifyURL, "/") + "/" + certificate.UniqueID
	if certificate.QRCode != nil && *certificate.QRCode != "" {
		verifyURL = *certificate.QRCode
	}

	content, err := certpdf.Render(certpdf.Certificate{
		StudentName:    certificate.StudentName,
		CourseTitle:    certificate.CourseTitle,
		InstructorName: certificate.InstructorName,
		CompletionDate: certificate.CompletionDate,
		UniqueID:       certificate.UniqueID,
		VerifyURL:      verifyURL,
	})
	if err != nil {
		return nil, apperror.Internal("service", "DownloadCertificate", "failed to draw the certificate", err)
	}

	return &CertificateFile{
		FileName: "certificate-" + certificate.UniqueID + ".pdf",
		Content:  content,
	}, nil
}

// Verify is the public check of a certificate. An id that does not exist is
// not an error: the answer is "not valid".
func (s *Certificate) Verify(ctx context.Context, uniqueID string) (*models.CertificateVerification, error) {
	certificate, err := s.repo.Certificate.GetByUniqueID(ctx, uniqueID)
	if err != nil {
		if core.IsNotFound(err) {
			return &models.CertificateVerification{Valid: false, UniqueID: uniqueID}, nil
		}

		return nil, err
	}

	return &models.CertificateVerification{
		Valid:          true,
		UniqueID:       certificate.UniqueID,
		StudentName:    certificate.StudentName,
		CourseTitle:    certificate.CourseTitle,
		InstructorName: certificate.InstructorName,
		CompletionDate: &certificate.CompletionDate,
	}, nil
}
