# API Endpoints

Base path: `/api/v1`. The list below is the same as the router (`internal/api/router.go`) and the Casbin policy (`internal/api/authz/policy.csv`).

## Conventions

**Access** column: `public` = no token needed (a token, if sent, is still read); `User` = any logged-in user; `Student`, `Instructor`, `SuperAdmin` = that role (SuperAdmin can do everything an Instructor can). Besides the role, the service checks the owner ("is this your course?"); the wrong owner gets `403`/`404`.

**Auth header:** `Authorization: Bearer <access_token>`.

**Success:** `{"success": true, "data": ...}`. Lists carry pagination (`page`, `limit`, `total`).

**Error:** `{"success": false, "error": {"code": "...", "message": "...", "fields": {"field": "reason"}}}`

| Status | Code | When |
|---|---|---|
| 400 | `INVALID_INPUT` | body/query/param is wrong (`fields` says which) |
| 401 | `UNAUTHORIZED` | no/bad/expired token, wrong password, or the session was ended (user blocked, deleted, password or role changed) |
| 403 | `FORBIDDEN` | the role or the owner does not allow it |
| 404 | `NOT_FOUND` | no such resource |
| 409 | `CONFLICT` | duplicate (username, email, enrollment...) |
| 429 | `TOO_MANY_REQUESTS` | rate limit or login lockout; see the `Retry-After` header (seconds) |
| 500 | `INTERNAL` | details are hidden and logged on the server |

**Dates:** `from` / `to` accept a day (`2026-05-01`) or an exact time (RFC 3339). A day in `to` lasts until its end, so `from=2026-05-01&to=2026-05-31` is the whole of May.

**Tokens:** the access token (JWT) is short-lived. The refresh token is opaque and rotates on every refresh. When a user is blocked or deleted, or changes the password, or gets another role, all their earlier access tokens stop working at once.

## 1. Auth (public, rate limited per IP)

| Method | Path | Limit | Notes |
|---|---|---|---|
| POST | `/auth/register` | 5 / 10 min | creates a Student |
| POST | `/auth/login` | 10 / min | body `username`, `password`; returns access + refresh token. 5 wrong passwords for one username lock it for 15 min (429) |
| POST | `/auth/refresh` | 30 / min | body `refresh_token`; returns a new pair |
| POST | `/auth/logout` | 30 / min | body `refresh_token` |
| POST | `/auth/forgot-password` | 3 / 15 min | body `email`; one email address gets at most 3 mails per hour. The link carries a one-time token (stored in Redis) |
| POST | `/auth/reset-password` | 10 / 15 min | body `token`, `new_password`; ends all sessions of the user |

## 2. Users

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/users/me` | User | |
| PUT | `/users/me` | User | first/last name, bio, avatar (`object_key` from an upload) |
| PUT | `/users/me/password` | User | `old_password`, `new_password`; ends other sessions |
| GET | `/users` | SuperAdmin | filters, pagination |
| POST | `/users` | SuperAdmin | |
| GET | `/users/:id` | SuperAdmin | |
| PUT | `/users/:id` | SuperAdmin | a role change ends the user's sessions |
| PATCH | `/users/:id/status` | SuperAdmin | `active` / `blocked`; blocking ends the sessions at once |
| DELETE | `/users/:id` | SuperAdmin | soft delete |

## 3. Uploads

| Method | Path | Access | Notes |
|---|---|---|---|
| POST | `/uploads/presign` | User | body: folder, file name, content type, size; returns a presigned PUT URL and the `object_key`. The file goes to `tmp/…` and is moved to its permanent key when it is attached (the saved key is in the response of the attaching request); a file that is never attached is deleted by the bucket after 1 day. Folder, type and size are validated |

## 4. Categories

| Method | Path | Access |
|---|---|---|
| GET | `/categories` | public |
| GET | `/categories/:id` | public |
| POST | `/categories` | SuperAdmin |
| PUT | `/categories/:id` | SuperAdmin |
| DELETE | `/categories/:id` | SuperAdmin |

## 5. Courses

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/courses` | public | query: `q`, `category_id`, `instructor_id`, `difficulty` (`beginner\|intermediate\|advanced`), `language`, `min_rating` (0-5), `price_type` (`free\|paid`), `status` (`draft\|published`, only for the owner/admin), `sort` (`popular\|rating\|newest\|price`), `page`, `limit`. Visitors see published courses only |
| GET | `/courses/:courseId` | public | instructor, outcomes, requirements, rating; a draft is for its owner |
| POST | `/courses` | Instructor | body has `learning_outcomes[]`, `requirements[]`, price, optional payout (`percentage` 0-100 / `fixed`) |
| PUT | `/courses/:courseId` | Instructor | owner only; outcomes and requirements are replaced with the course |
| PATCH | `/courses/:courseId/status` | Instructor | `draft` / `published` |
| DELETE | `/courses/:courseId` | Instructor | soft delete |

## 6. Modules, lessons, materials

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/courses/:courseId/modules` | public | |
| GET | `/modules/:moduleId` | public | |
| POST | `/courses/:courseId/modules` | Instructor | |
| PUT | `/modules/:moduleId` | Instructor | |
| PATCH | `/modules/:moduleId/order` | Instructor | `order_number` |
| DELETE | `/modules/:moduleId` | Instructor | |
| GET | `/modules/:moduleId/lessons` | public | |
| GET | `/lessons/:lessonId` | public | |
| POST | `/modules/:moduleId/lessons` | Instructor | |
| PUT | `/lessons/:lessonId` | Instructor | |
| PATCH | `/lessons/:lessonId/order` | Instructor | |
| DELETE | `/lessons/:lessonId` | Instructor | |
| GET | `/lessons/:lessonId/materials` | public* | *a non-preview lesson needs an enrolled student, the owner or SuperAdmin; file materials come with a presigned download URL |
| GET | `/materials/:materialId` | public* | same rule |
| POST | `/lessons/:lessonId/materials` | Instructor | type `text` / `video` / `file` |
| PUT | `/materials/:materialId` | Instructor | |
| DELETE | `/materials/:materialId` | Instructor | |

## 7. Enrollments and progress

| Method | Path | Access | Notes |
|---|---|---|---|
| POST | `/courses/:courseId/enrollments` | Student | a paid course creates the payment (and the instructor payout) in the same transaction |
| GET | `/courses/:courseId/enrollments` | Instructor | roster of the course |
| GET | `/enrollments/me` | Student | |
| GET | `/enrollments/:enrollmentId` | User | owner student, course instructor or SuperAdmin |
| PATCH | `/enrollments/:enrollmentId/status` | User | `active` / `dropped` |
| GET | `/courses/:courseId/progress` | Student | percentage, lessons done, quiz state |
| GET | `/lessons/:lessonId/progress` | Student | |
| POST | `/lessons/:lessonId/progress` | Student | mark done/undone. When all lessons are done and the final quiz is passed (if any), the course is `completed` and the certificate is issued automatically |
| GET | `/courses/:courseId/students` | Instructor | |
| GET | `/courses/:courseId/students/:studentId/progress` | Instructor | |

## 8. Quizzes, questions, attempts

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/courses/:courseId/quizzes` | User | |
| POST | `/courses/:courseId/quizzes` | Instructor | final quiz of the course |
| GET | `/modules/:moduleId/quizzes` | User | |
| POST | `/modules/:moduleId/quizzes` | Instructor | |
| GET | `/quizzes/:quizId` | User | |
| PUT | `/quizzes/:quizId` | Instructor | `time_limit` (>0), `pass_threshold` (0-100), `max_attempts` (>0) |
| DELETE | `/quizzes/:quizId` | Instructor | |
| GET | `/quizzes/:quizId/questions` | Instructor | with the right answers; only the author sees them |
| POST | `/quizzes/:quizId/questions` | Instructor | type `single_choice` / `multiple_choice` / `true_false` with options |
| GET | `/questions/:questionId` | Instructor | |
| PUT | `/questions/:questionId` | Instructor | |
| DELETE | `/questions/:questionId` | Instructor | |
| POST | `/quizzes/:quizId/attempts` | Student | starts an attempt; questions and options come shuffled (stable for the attempt), without the right answers |
| GET | `/quizzes/:quizId/attempts` | User | own attempts (student) / all (owner, admin) |
| GET | `/attempts/:attemptId` | User | |
| POST | `/attempts/:attemptId/submit` | Student | answers; returns the score and pass/fail |

## 9. Certificates and reviews

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/certificates/verify/:uniqueId` | public | `LMS-XXXX-XXXX-XXXX`; the QR code of the certificate points here |
| GET | `/certificates/me` | Student | |
| GET | `/certificates/:certificateId` | User | owner or admin |
| GET | `/certificates/:certificateId/download` | User | the PDF (`application/pdf`) |
| GET | `/courses/:courseId/reviews` | public | |
| POST | `/courses/:courseId/reviews` | Student | needs an enrollment; `rating` 1-5, one per student and course |
| PUT | `/reviews/:reviewId` | Student | own review |
| DELETE | `/reviews/:reviewId` | User | own review, or SuperAdmin |

## 10. Payments and finance

| Method | Path | Access | Notes |
|---|---|---|---|
| GET | `/payments` | SuperAdmin | query: `from`, `to`, `course_id`, pagination |
| GET | `/payments/:paymentId` | User | owner student, course instructor or SuperAdmin |
| GET | `/enrollments/:enrollmentId/payment` | User | same |
| GET | `/finance` | SuperAdmin | revenue, instructor payouts, net; query `from`, `to`, `group_by` (`day\|week\|month`) |
| GET | `/finance/revenue` | SuperAdmin | same query |
| GET | `/finance/expenses` | SuperAdmin | payouts to instructors; same query |

## 11. Reports (SuperAdmin)

`GET /reports/{enrollments|revenue|students|progress|quizzes|certificates|instructors|reviews}`

Query: `from`, `to`, `course_id`, `group_by` (`day|week|month`), `format` (`json` default, or `csv` for a file download: `Content-Type: text/csv`, `Content-Disposition: attachment`).
