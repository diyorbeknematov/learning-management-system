# API Endpoints of Learning Management System

Base path: `/api/v1`

Public (no auth): `GET /courses`, `GET /courses/:courseId`, `GET /categories`, `GET /categories/:id`, `GET /courses/:courseId/reviews`, `GET /certificates/verify/:uniqueId`, all `/auth/*` except `logout`.

### 1. Auth
- POST /api/v1/auth/register
- POST /api/v1/auth/login
- POST /api/v1/auth/refresh
- POST /api/v1/auth/logout                (body: refresh token)
- POST /api/v1/auth/forgot-password       (body: email; reset token is stored in Redis with TTL)
- POST /api/v1/auth/reset-password        (body: token, new_password)

### 2. Users
- GET    /api/v1/users
- GET    /api/v1/users/:id
- POST   /api/v1/users
- PUT    /api/v1/users/:id
- PATCH  /api/v1/users/:id/status
- DELETE /api/v1/users/:id

- GET    /api/v1/users/me
- PUT    /api/v1/users/me                 (name, bio, avatar object_key; no password)
- PUT    /api/v1/users/me/password        (body: old_password, new_password)

### 3. Categories
- GET    /api/v1/categories
- GET    /api/v1/categories/:id
- POST   /api/v1/categories
- PUT    /api/v1/categories/:id
- DELETE /api/v1/categories/:id

### 4. Courses
- GET    /api/v1/courses
  - query: `q`, `category_id`, `difficulty`, `min_rating`, `price_type=free|paid`, `language`, `instructor_id`, `sort=popular|rating|newest|price`, `page`, `limit`
- GET    /api/v1/courses/:courseId          (includes instructor, syllabus, outcomes, requirements, average rating)
- POST   /api/v1/courses                    (body includes `learning_outcomes[]` and `requirements[]`)
- PUT    /api/v1/courses/:courseId          (outcomes and requirements are replaced together with the course)
- PATCH  /api/v1/courses/:courseId/status
- DELETE /api/v1/courses/:courseId

### 5. Modules
- GET    /api/v1/courses/:courseId/modules
- POST   /api/v1/courses/:courseId/modules

- GET    /api/v1/modules/:moduleId
- PUT    /api/v1/modules/:moduleId
- PATCH  /api/v1/modules/:moduleId/order
- DELETE /api/v1/modules/:moduleId

### 6. Lessons
- GET    /api/v1/modules/:moduleId/lessons
- POST   /api/v1/modules/:moduleId/lessons

- GET    /api/v1/lessons/:lessonId
- PUT    /api/v1/lessons/:lessonId
- PATCH  /api/v1/lessons/:lessonId/order
- DELETE /api/v1/lessons/:lessonId

### 7. Lesson Materials
- GET    /api/v1/lessons/:lessonId/materials
- POST   /api/v1/lessons/:lessonId/materials

- GET    /api/v1/materials/:materialId
- PUT    /api/v1/materials/:materialId
- DELETE /api/v1/materials/:materialId

### 8. Uploads (MinIO)
- POST /api/v1/uploads/presign            (body: purpose=`avatar|course_cover|material`, file_name, content_type; returns upload_url and object_key)

Client uploads the file straight to MinIO with `upload_url`, then sends `object_key` in the create/update request (user, course, material) so it is saved in Postgres.

### 9. Enrollments
- GET    /api/v1/courses/:courseId/enrollments
- POST   /api/v1/courses/:courseId/enrollments

- GET    /api/v1/enrollments/:enrollmentId
- PATCH  /api/v1/enrollments/:enrollmentId/status
- GET    /api/v1/enrollments/me

### 10. Student Progress
- GET  /api/v1/courses/:courseId/progress
- GET  /api/v1/lessons/:lessonId/progress
- POST /api/v1/lessons/:lessonId/progress

### 11. Instructor → Students & Progress
- GET /api/v1/courses/:courseId/students
- GET /api/v1/courses/:courseId/students/:studentId/progress

### 12. Quizzes
- GET    /api/v1/courses/:courseId/quizzes
- POST   /api/v1/courses/:courseId/quizzes

- GET    /api/v1/modules/:moduleId/quizzes
- POST   /api/v1/modules/:moduleId/quizzes

- GET    /api/v1/quizzes/:quizId
- PUT    /api/v1/quizzes/:quizId
- DELETE /api/v1/quizzes/:quizId

### 13. Questions
- GET    /api/v1/quizzes/:quizId/questions
- POST   /api/v1/quizzes/:quizId/questions

- GET    /api/v1/questions/:questionId
- PUT    /api/v1/questions/:questionId
- DELETE /api/v1/questions/:questionId

### 14. Quiz Attempts
- POST /api/v1/quizzes/:quizId/attempts
- GET  /api/v1/quizzes/:quizId/attempts

- GET  /api/v1/attempts/:attemptId
- POST /api/v1/attempts/:attemptId/submit

### 15. Certificates
- GET /api/v1/certificates/me
- GET /api/v1/certificates/:certificateId
- GET /api/v1/certificates/:certificateId/download
- GET /api/v1/certificates/verify/:uniqueId

### 16. Reviews
- GET    /api/v1/courses/:courseId/reviews
- POST   /api/v1/courses/:courseId/reviews

- PUT    /api/v1/reviews/:reviewId
- DELETE /api/v1/reviews/:reviewId

### 17. Payments
- GET /api/v1/payments
- GET /api/v1/payments/:paymentId
- GET /api/v1/enrollments/:enrollmentId/payment

### 18. Finance
All accept `from`, `to` and `group_by=day|week|month`.
- GET /api/v1/finance                     (revenue, expenses, net_profit, status=profit|loss)
- GET /api/v1/finance/revenue
- GET /api/v1/finance/expenses            (instructor payouts)

### 19. Reports
All accept `from`, `to`, optional `course_id`, and `format=json|csv` (`csv` returns a downloadable file).
- GET /api/v1/reports/enrollments
- GET /api/v1/reports/revenue
- GET /api/v1/reports/students
- GET /api/v1/reports/progress
- GET /api/v1/reports/quizzes
- GET /api/v1/reports/certificates
- GET /api/v1/reports/instructors
- GET /api/v1/reports/reviews
