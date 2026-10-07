package testutil

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/internal/config"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/internal/repo"
	"github.com/diyorbeknematov/lms/internal/repo/postgres"
	"github.com/diyorbeknematov/lms/internal/service"
	"github.com/diyorbeknematov/lms/internal/storage/minio"
	"github.com/diyorbeknematov/lms/pkg/apperror"
	"github.com/diyorbeknematov/lms/pkg/token"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	// Password is the password of every user made by NewUser.
	Password       = "Passw0rd!2024"
	accessTokenKey = "test-secret"
)

// FakeRedis keeps values in memory. It stores JSON like the real client and
// ignores the ttl, but remembers it. Counters (Incr) live next to the values
// and are read with Get like in Redis.
type FakeRedis struct {
	mu       sync.Mutex
	Values   map[string][]byte
	TTLs     map[string]time.Duration
	Counters map[string]int64
}

func (f *FakeRedis) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.Values[key] = raw
	f.TTLs[key] = ttl

	return nil
}

func (f *FakeRedis) Get(_ context.Context, key string, dest any) error {
	f.mu.Lock()
	raw, ok := f.Values[key]
	count, counted := f.Counters[key]
	f.mu.Unlock()

	if counted {
		return json.Unmarshal([]byte(strconv.FormatInt(count, 10)), dest)
	}

	if !ok {
		return apperror.ErrNotFound
	}

	return json.Unmarshal(raw, dest)
}

func (f *FakeRedis) GetDel(_ context.Context, key string, dest any) error {
	f.mu.Lock()
	raw, ok := f.Values[key]
	delete(f.Values, key)
	f.mu.Unlock()

	if !ok {
		return apperror.ErrNotFound
	}

	return json.Unmarshal(raw, dest)
}

func (f *FakeRedis) Del(_ context.Context, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, key := range keys {
		delete(f.Values, key)
		delete(f.TTLs, key)
		delete(f.Counters, key)
	}

	return nil
}

// Incr counts a hit. The window never ends in the fake; Reset starts a new one.
func (f *FakeRedis) Incr(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Counters[key]++
	f.TTLs[key] = window

	return f.Counters[key], window, nil
}

// Reset forgets every counter, like the end of all windows.
func (f *FakeRedis) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Counters = map[string]int64{}
}

// FakeStorage is a file storage in memory. Upload puts a file there as if the
// client had uploaded it with a presigned URL.
type FakeStorage struct {
	mu      sync.Mutex
	Objects map[string]minio.ObjectInfo
	Deleted []string
}

func (f *FakeStorage) Upload(key string, size int64, contentType string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.Objects[key] = minio.ObjectInfo{Size: size, ContentType: contentType}
}

func (f *FakeStorage) PresignUpload(_ context.Context, key string) (string, error) {
	return "http://files.test/upload/" + key, nil
}

func (f *FakeStorage) PresignDownload(_ context.Context, key string) (string, error) {
	return "http://files.test/" + key, nil
}

func (f *FakeStorage) Stat(_ context.Context, key string) (*minio.ObjectInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	info, ok := f.Objects[key]
	if !ok {
		return nil, apperror.NotFound("storage", "Stat", "file not found", apperror.ErrNotFound)
	}

	return &info, nil
}

func (f *FakeStorage) Copy(_ context.Context, source, destination string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	info, ok := f.Objects[source]
	if !ok {
		return apperror.NotFound("storage", "Copy", "file not found", apperror.ErrNotFound)
	}

	f.Objects[destination] = info

	return nil
}

func (f *FakeStorage) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.Objects, key)
	f.Deleted = append(f.Deleted, key)

	return nil
}

type FakeMailer struct {
	mu      sync.Mutex
	To      string
	Subject string
	Body    string
	Sent    int
}

func (f *FakeMailer) Send(_ context.Context, to, subject, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.To, f.Subject, f.Body = to, subject, body
	f.Sent++

	return nil
}

// Env is a test environment: the services on a real Postgres with a fake
// Redis and a fake mailer.
type Env struct {
	Svc     *service.Service
	Repo    *repo.Repository
	DB      *postgres.Postgres
	Redis   *FakeRedis
	Mailer  *FakeMailer
	Storage *FakeStorage
	Tokens  *token.Manager
}

// setup connects to Postgres (the test is skipped when it is not available)
// and builds the services with a fake Redis and a fake mailer.
func Setup(t *testing.T) *Env {
	t.Helper()

	db, err := postgres.New(config.Load().DB)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}

	t.Cleanup(db.Close)

	e := &Env{
		Repo:    repo.NewRepository(db.Pool),
		DB:      db,
		Redis:   &FakeRedis{Values: map[string][]byte{}, TTLs: map[string]time.Duration{}, Counters: map[string]int64{}},
		Mailer:  &FakeMailer{},
		Storage: &FakeStorage{Objects: map[string]minio.ObjectInfo{}},
		Tokens:  token.NewManager(accessTokenKey, time.Minute),
	}

	e.Svc = service.New(service.Dependencies{
		Repo:    e.Repo,
		Tokens:  e.Tokens,
		Redis:   e.Redis,
		Mailer:  e.Mailer,
		Storage: e.Storage,
		Config: service.Config{
			RefreshTokenTTL:      time.Hour,
			ResetTokenTTL:        15 * time.Minute,
			ResetPasswordURL:     "http://localhost:3000/reset-password",
			CertificateVerifyURL: "http://localhost:8080/api/v1/certificates/verify",
			AccessTokenTTL:       time.Minute,
		},
	})

	return e
}

func (e *Env) Exec(t *testing.T, query string, args ...any) {
	t.Helper()

	_, err := e.DB.Pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
}

func Uniq() string {
	return uuid.NewString()[:8]
}

func RequireCode(t *testing.T, err error, code apperror.ErrorCode) {
	t.Helper()

	require.Error(t, err)

	appErr, ok := apperror.As(err)
	require.True(t, ok, "not an apperror: %v", err)
	require.Equal(t, code, appErr.Code, appErr.Error())
}

func ActorOf(user *models.User) models.Actor {
	return models.Actor{UserID: user.ID, RoleName: user.RoleName}
}

func AdminActor() models.Actor {
	return models.Actor{UserID: uuid.New(), RoleName: models.RoleSuperAdmin}
}

// cleanupUser removes a user and the rows that belong to the user as a
// student, so it does not matter in which order the cleanups run. A user who
// owns courses needs cleanupCourse to run first (it is registered later, so
// it does).
func (e *Env) CleanupUser(t *testing.T, id uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM attempt_answers WHERE attempt_id IN (SELECT id FROM quiz_attempts WHERE student_id = $1)`,
			`DELETE FROM quiz_attempts WHERE student_id = $1`,
			`DELETE FROM lesson_progress WHERE student_id = $1`,
			`DELETE FROM instructor_payouts WHERE enrollment_id IN (SELECT id FROM enrollments WHERE student_id = $1)`,
			`DELETE FROM payments WHERE enrollment_id IN (SELECT id FROM enrollments WHERE student_id = $1)`,
			`DELETE FROM enrollments WHERE student_id = $1`,
			`DELETE FROM reviews WHERE student_id = $1`,
			`DELETE FROM certificates WHERE student_id = $1`,
			`DELETE FROM refresh_tokens WHERE user_id = $1`,
			`DELETE FROM users WHERE id = $1`,
		} {
			e.Exec(t, query, id)
		}
	})
}

// newUser creates a user with the given role through the user service.
func (e *Env) NewUser(t *testing.T, role string) *models.User {
	t.Helper()

	name := Uniq()

	user, err := e.Svc.User.Create(context.Background(), models.CreateUserRequest{
		FirstName: "Test",
		LastName:  role,
		Username:  "u" + name,
		Email:     name + "@example.com",
		Password:  Password,
		Role:      role,
	})
	require.NoError(t, err)

	e.CleanupUser(t, user.ID)

	return user
}

func (e *Env) NewCategory(t *testing.T) *models.Category {
	t.Helper()

	category, err := e.Svc.Category.Create(context.Background(), models.CreateCategory{Name: "cat-" + Uniq()})
	require.NoError(t, err)

	t.Cleanup(func() {
		e.Exec(t, `DELETE FROM categories WHERE id = $1`, category.ID)
	})

	return category
}

// cleanupCourse removes a course and everything that belongs to it.
func (e *Env) CleanupCourse(t *testing.T, id uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		for _, query := range []string{
			`DELETE FROM attempt_answers WHERE attempt_id IN (SELECT a.id FROM quiz_attempts a JOIN quizzes q ON q.id = a.quiz_id WHERE q.course_id = $1 OR q.module_id IN (SELECT id FROM modules WHERE course_id = $1))`,
			`DELETE FROM quiz_attempts WHERE quiz_id IN (SELECT id FROM quizzes WHERE course_id = $1 OR module_id IN (SELECT id FROM modules WHERE course_id = $1))`,
			`DELETE FROM question_options WHERE question_id IN (SELECT qu.id FROM questions qu JOIN quizzes q ON q.id = qu.quiz_id WHERE q.course_id = $1 OR q.module_id IN (SELECT id FROM modules WHERE course_id = $1))`,
			`DELETE FROM questions WHERE quiz_id IN (SELECT id FROM quizzes WHERE course_id = $1 OR module_id IN (SELECT id FROM modules WHERE course_id = $1))`,
			`DELETE FROM quizzes WHERE course_id = $1 OR module_id IN (SELECT id FROM modules WHERE course_id = $1)`,
			`DELETE FROM lesson_progress WHERE lesson_id IN (SELECT l.id FROM lessons l JOIN modules m ON m.id = l.module_id WHERE m.course_id = $1)`,
			`DELETE FROM lesson_materials WHERE lesson_id IN (SELECT l.id FROM lessons l JOIN modules m ON m.id = l.module_id WHERE m.course_id = $1)`,
			`DELETE FROM lessons WHERE module_id IN (SELECT id FROM modules WHERE course_id = $1)`,
			`DELETE FROM modules WHERE course_id = $1`,
			`DELETE FROM instructor_payouts WHERE course_id = $1`,
			`DELETE FROM payments WHERE enrollment_id IN (SELECT id FROM enrollments WHERE course_id = $1)`,
			`DELETE FROM enrollments WHERE course_id = $1`,
			`DELETE FROM reviews WHERE course_id = $1`,
			`DELETE FROM certificates WHERE course_id = $1`,
			`DELETE FROM course_learning_outcomes WHERE course_id = $1`,
			`DELETE FROM course_requirements WHERE course_id = $1`,
			`DELETE FROM courses WHERE id = $1`,
		} {
			e.Exec(t, query, id)
		}
	})
}

// newCourse creates a draft course owned by the instructor.
func (e *Env) NewCourse(t *testing.T, instructor *models.User, category *models.Category) *models.CourseDetail {
	t.Helper()

	course, err := e.Svc.Course.Create(context.Background(), ActorOf(instructor), models.CreateCourseRequest{
		CategoryID: category.ID,
		Title:      "Course " + Uniq(),
		Price:      20,
	})
	require.NoError(t, err)

	e.CleanupCourse(t, course.ID)

	return course
}

// AddLesson puts a new module with one lesson into the course, so it can be
// published. The module gets a high order number so it never clashes with the
// modules a test made itself.
func (e *Env) AddLesson(t *testing.T, courseID uuid.UUID) uuid.UUID {
	t.Helper()

	ctx := context.Background()

	existing, err := e.Repo.Module.GetListByCourseID(ctx, courseID)
	require.NoError(t, err)

	moduleID, err := e.Repo.Module.Create(ctx, models.CreateModule{
		CourseID:    courseID,
		Title:       "Module",
		OrderNumber: len(existing) + 1000,
	})
	require.NoError(t, err)

	lessonID, err := e.Repo.Lesson.Create(ctx, models.CreateLesson{ModuleID: moduleID, Title: "Lesson", OrderNumber: 1})
	require.NoError(t, err)

	return lessonID
}

// Publish adds a lesson and publishes the course as its owner.
func (e *Env) Publish(t *testing.T, owner *models.User, courseID uuid.UUID) {
	t.Helper()

	e.AddLesson(t, courseID)

	err := e.Svc.Course.UpdateStatus(context.Background(), ActorOf(owner), models.UpdateCourseStatus{
		ID:     courseID,
		Status: models.CourseStatusPublished,
	})
	require.NoError(t, err)
}

// Login logs the user in with the password of NewUser.
func (e *Env) Login(t *testing.T, user *models.User) *models.LoginResponse {
	t.Helper()

	response, err := e.Svc.Auth.Login(context.Background(), models.LoginRequest{Username: user.Username, Password: Password})
	require.NoError(t, err)

	return response
}
