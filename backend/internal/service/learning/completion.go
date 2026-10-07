package learning

import (
	"context"
	"crypto/rand"
	"strings"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/service/core"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
)

// uniqueIDAlphabet has no 0, O, 1, I or L, so a certificate id is easy to read
// out and to type.
const uniqueIDAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// newUniqueID returns an id like LMS-7K2M-QX9P-4DHB: 12 random characters, about
// 59 bits, so nobody can guess another certificate.
func newUniqueID() (string, error) {
	random := make([]byte, 12)

	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	var id strings.Builder

	id.WriteString("LMS")

	for i, b := range random {
		if i%4 == 0 {
			id.WriteByte('-')
		}

		id.WriteByte(uniqueIDAlphabet[int(b)%len(uniqueIDAlphabet)])
	}

	return id.String(), nil
}

// syncCompletion decides whether the student has finished the course and
// updates the enrollment: the course is completed when all lessons are done
// and, if the course has a final quiz, the student passed it. The certificate
// is issued the moment the course becomes completed. A certificate that was
// issued stays valid when a lesson is taken back later.
//
// Call it inside a transaction after the lessons or the quiz result changed.
func syncCompletion(ctx context.Context, tx *repo.Repository, cfg core.Config, studentID uuid.UUID, course *models.Course) error {
	enrollment, err := tx.Enrollment.GetByStudentAndCourse(ctx, studentID, course.ID)
	if err != nil {
		if core.IsNotFound(err) {
			return nil
		}

		return err
	}

	if enrollment.Status == models.EnrollmentStatusDropped {
		return nil
	}

	finished, err := finishedCourse(ctx, tx, studentID, course.ID)
	if err != nil {
		return err
	}

	switch {
	case finished && enrollment.Status == models.EnrollmentStatusActive:
		err := tx.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusCompleted})
		if err != nil {
			return err
		}

		return issueCertificate(ctx, tx, cfg, studentID, course.ID)
	case finished:
		// completed already; make sure the certificate exists
		return issueCertificate(ctx, tx, cfg, studentID, course.ID)
	case enrollment.Status == models.EnrollmentStatusCompleted:
		return tx.Enrollment.UpdateStatus(ctx, models.UpdateEnrollmentStatus{ID: enrollment.ID, Status: models.EnrollmentStatusActive})
	}

	return nil
}

// finishedCourse is true when every lesson is done and the final quiz, if
// there is one, is passed.
func finishedCourse(ctx context.Context, tx *repo.Repository, studentID, courseID uuid.UUID) (bool, error) {
	progress, err := tx.Progress.GetCourseProgress(ctx, studentID, courseID)
	if err != nil {
		return false, err
	}

	if progress.TotalLessons == 0 || progress.CompletedLessons < progress.TotalLessons {
		return false, nil
	}

	quizzes, err := tx.Quiz.GetListByCourseID(ctx, courseID)
	if err != nil {
		return false, err
	}

	for _, quiz := range quizzes {
		attempts, err := tx.Attempt.GetList(ctx, quiz.ID, &studentID)
		if err != nil {
			return false, err
		}

		passed := false

		for _, attempt := range attempts {
			if attempt.Score != nil && *attempt.Score >= quiz.PassThreshold {
				passed = true
			}
		}

		if !passed {
			return false, nil
		}
	}

	return true, nil
}

// issueCertificate gives the student a certificate for the course, once.
func issueCertificate(ctx context.Context, tx *repo.Repository, cfg core.Config, studentID, courseID uuid.UUID) error {
	_, err := tx.Certificate.GetByStudentAndCourse(ctx, studentID, courseID)
	if err == nil {
		return nil
	}

	if !core.IsNotFound(err) {
		return err
	}

	uniqueID, err := newUniqueID()
	if err != nil {
		return apperror.Internal("service", "IssueCertificate", "failed to create a certificate id", err)
	}

	// what the QR code of the certificate says: where to check it
	verifyURL := strings.TrimRight(cfg.CertificateVerifyURL, "/") + "/" + uniqueID

	_, err = tx.Certificate.Create(ctx, models.CreateCertificate{
		StudentID:      studentID,
		CourseID:       courseID,
		CompletionDate: time.Now(),
		UniqueID:       uniqueID,
		QRCode:         &verifyURL,
	})

	return err
}
