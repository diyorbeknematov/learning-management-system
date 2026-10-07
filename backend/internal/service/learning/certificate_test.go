package learning_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/service/testutil"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const verifyURL = "http://localhost:8080/api/v1/certificates/verify"

var certificateID = regexp.MustCompile(`^LMS(-[A-Z2-9]{4}){3}$`)

func (s shop) certificates(t *testing.T, student *models.User) []models.CertificateDetail {
	t.Helper()

	certificates, err := s.e.Svc.Certificate.GetMyList(context.Background(), testutil.ActorOf(student))
	require.NoError(t, err)

	return certificates
}

func (s shop) status(t *testing.T, student *models.User) models.EnrollmentStatus {
	t.Helper()

	enrollment, err := s.e.Repo.Enrollment.GetByStudentAndCourse(context.Background(), student.ID, s.course.ID)
	require.NoError(t, err)

	return enrollment.Status
}

// finish does all the lessons of the course.
func (s shop) finish(t *testing.T, student *models.User) {
	t.Helper()

	for _, lessonID := range s.lessons {
		s.mark(t, student, lessonID, true)
	}
}

// pass takes the final quiz with the right answers.
func (x exam) pass(t *testing.T, student *models.User) {
	t.Helper()

	view := x.start(t, student)

	result, err := x.submit(t, student, view.ID, x.rightAnswers())
	require.NoError(t, err)
	require.True(t, *result.Passed)
}

// fail takes the final quiz and answers nothing.
func (x exam) fail(t *testing.T, student *models.User) {
	t.Helper()

	view := x.start(t, student)

	result, err := x.submit(t, student, view.ID, nil)
	require.NoError(t, err)
	require.False(t, *result.Passed)
}

func TestCertificate_IssuedWhenTheCourseIsDone(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	student := s.enrolledStudent(t)

	s.mark(t, student, s.lessons[0], true)
	require.Empty(t, s.certificates(t, student), "half of the course is not enough")
	require.Equal(t, models.EnrollmentStatusActive, s.status(t, student))

	before := time.Now().Add(-time.Minute)
	s.mark(t, student, s.lessons[1], true)

	certificates := s.certificates(t, student)
	require.Len(t, certificates, 1)

	certificate := certificates[0]
	require.Equal(t, student.ID, certificate.StudentID)
	require.Equal(t, s.course.ID, certificate.CourseID)
	require.Regexp(t, certificateID, certificate.UniqueID)
	require.Equal(t, s.course.Title, certificate.CourseTitle)
	require.Equal(t, s.owner.FirstName+" "+s.owner.LastName, certificate.InstructorName)
	require.Equal(t, student.FirstName+" "+student.LastName, certificate.StudentName)
	require.True(t, certificate.CompletionDate.After(before))
	require.Equal(t, verifyURL+"/"+certificate.UniqueID, *certificate.QRCode)
	require.Equal(t, models.EnrollmentStatusCompleted, s.status(t, student))
}

func TestCertificate_IssuedOnce(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	student := s.enrolledStudent(t)
	s.finish(t, student)

	first := s.certificates(t, student)
	require.Len(t, first, 1)

	// marking a done lesson again changes nothing
	s.mark(t, student, s.lessons[0], true)
	require.Len(t, s.certificates(t, student), 1)

	// taking a lesson back and doing it again does not issue a second one
	s.mark(t, student, s.lessons[1], false)
	s.mark(t, student, s.lessons[1], true)

	again := s.certificates(t, student)
	require.Len(t, again, 1)
	require.Equal(t, first[0].UniqueID, again[0].UniqueID)
	require.Equal(t, first[0].ID, again[0].ID)
}

func TestCertificate_StaysWhenALessonIsTakenBack(t *testing.T) {
	s := newShop(t, 0, 2, nil, 0)

	student := s.enrolledStudent(t)
	s.finish(t, student)

	s.mark(t, student, s.lessons[0], false)

	require.Equal(t, models.EnrollmentStatusActive, s.status(t, student))
	require.Len(t, s.certificates(t, student), 1, "an issued certificate stays valid")
}

func TestCertificate_EveryStudentGetsTheirOwnId(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	first, second := s.enrolledStudent(t), s.enrolledStudent(t)
	s.finish(t, first)
	s.finish(t, second)

	one, two := s.certificates(t, first), s.certificates(t, second)
	require.Len(t, one, 1)
	require.Len(t, two, 1)
	require.NotEqual(t, one[0].UniqueID, two[0].UniqueID)
}

func TestCertificate_FinalQuizIsRequired(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	x.finish(t, student)

	require.Equal(t, models.EnrollmentStatusActive, x.status(t, student), "the lessons are done, the quiz is not")
	require.Empty(t, x.certificates(t, student))

	x.fail(t, student)

	require.Equal(t, models.EnrollmentStatusActive, x.status(t, student), "a failed quiz does not finish the course")
	require.Empty(t, x.certificates(t, student))

	x.pass(t, student)

	require.Equal(t, models.EnrollmentStatusCompleted, x.status(t, student))
	require.Len(t, x.certificates(t, student), 1)
}

func TestCertificate_QuizPassedBeforeTheLastLesson(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)
	x.pass(t, student)

	require.Equal(t, models.EnrollmentStatusActive, x.status(t, student), "the quiz alone does not finish the course")
	require.Empty(t, x.certificates(t, student))

	x.finish(t, student)

	require.Equal(t, models.EnrollmentStatusCompleted, x.status(t, student))
	require.Len(t, x.certificates(t, student), 1)
}

func TestCertificate_ALaterPassIsEnoughAfterAFailure(t *testing.T) {
	x := newExam(t, 70, 3)

	student := x.enrolledStudent(t)

	x.fail(t, student)
	x.finish(t, student)
	require.Empty(t, x.certificates(t, student))

	x.pass(t, student)
	require.Len(t, x.certificates(t, student), 1)
}

func TestCertificate_GetMyList_OnlyOwn(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	student := s.enrolledStudent(t)
	s.finish(t, student)

	require.Len(t, s.certificates(t, student), 1)
	require.Empty(t, s.certificates(t, s.student(t)))
}

func TestCertificate_GetByID(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.enrolledStudent(t)
	s.finish(t, student)

	certificate := s.certificates(t, student)[0]

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.ActorOf(s.owner), testutil.AdminActor()} {
		found, err := s.e.Svc.Certificate.GetByID(ctx, actor, certificate.ID)
		require.NoError(t, err)
		require.Equal(t, certificate.UniqueID, found.UniqueID)
	}

	for _, other := range []*models.User{s.student(t), s.e.NewUser(t, models.RoleInstructor)} {
		_, err := s.e.Svc.Certificate.GetByID(ctx, testutil.ActorOf(other), certificate.ID)
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}

	_, err := s.e.Svc.Certificate.GetByID(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCertificate_Verify(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.enrolledStudent(t)
	s.finish(t, student)

	certificate := s.certificates(t, student)[0]

	verification, err := s.e.Svc.Certificate.Verify(ctx, certificate.UniqueID)
	require.NoError(t, err)
	require.True(t, verification.Valid)
	require.Equal(t, certificate.UniqueID, verification.UniqueID)
	require.Equal(t, certificate.StudentName, verification.StudentName)
	require.Equal(t, s.course.Title, verification.CourseTitle)
	require.Equal(t, certificate.InstructorName, verification.InstructorName)
	require.NotNil(t, verification.CompletionDate)
}

func TestCertificate_Verify_UnknownIdIsNotValid(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	verification, err := s.e.Svc.Certificate.Verify(context.Background(), "LMS-AAAA-BBBB-CCCC")
	require.NoError(t, err)
	require.False(t, verification.Valid)
	require.Equal(t, "LMS-AAAA-BBBB-CCCC", verification.UniqueID)
	require.Empty(t, verification.StudentName)
	require.Nil(t, verification.CompletionDate)
}

func TestCertificate_Download(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.enrolledStudent(t)
	s.finish(t, student)

	certificate := s.certificates(t, student)[0]

	for _, actor := range []models.Actor{testutil.ActorOf(student), testutil.ActorOf(s.owner), testutil.AdminActor()} {
		file, err := s.e.Svc.Certificate.Download(ctx, actor, certificate.ID)
		require.NoError(t, err)
		require.Equal(t, "certificate-"+certificate.UniqueID+".pdf", file.FileName)
		require.True(t, strings.HasPrefix(string(file.Content), "%PDF-"))
		require.Greater(t, len(file.Content), 10_000)
	}
}

func TestCertificate_Download_OnlyWhoMaySeeIt(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	student := s.enrolledStudent(t)
	s.finish(t, student)

	certificate := s.certificates(t, student)[0]

	for _, other := range []*models.User{s.student(t), s.e.NewUser(t, models.RoleInstructor)} {
		_, err := s.e.Svc.Certificate.Download(ctx, testutil.ActorOf(other), certificate.ID)
		testutil.RequireCode(t, err, apperror.CodeForbidden)
	}

	_, err := s.e.Svc.Certificate.Download(ctx, testutil.AdminActor(), uuid.New())
	testutil.RequireCode(t, err, apperror.CodeNotFound)
}

func TestCertificate_Download_EachCertificateHasItsOwnFile(t *testing.T) {
	s := newShop(t, 0, 1, nil, 0)

	ctx := context.Background()
	first, second := s.enrolledStudent(t), s.enrolledStudent(t)
	s.finish(t, first)
	s.finish(t, second)

	one, err := s.e.Svc.Certificate.Download(ctx, testutil.ActorOf(first), s.certificates(t, first)[0].ID)
	require.NoError(t, err)

	two, err := s.e.Svc.Certificate.Download(ctx, testutil.ActorOf(second), s.certificates(t, second)[0].ID)
	require.NoError(t, err)

	require.NotEqual(t, one.FileName, two.FileName)
	require.NotEqual(t, one.Content, two.Content)
}
