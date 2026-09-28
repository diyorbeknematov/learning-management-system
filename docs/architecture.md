# System Architecture

## 1. Architecture Overview

The Learning Management System follows a layered backend architecture built with Go.

```text
Client
  │
  ▼
Go REST API
  │
  ▼
JWT Authentication
  │
  ▼
Casbin Authorization
  │
  ▼
Handler
  │
  ▼
Service
  │
  ▼
Repository
  │
  ▼
PostgreSQL
```

Additional infrastructure is used for caching and object storage:

```text
                    ┌─────────────┐
                    │   Client    │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   Go API    │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         PostgreSQL      Redis        MinIO
         Database        Cache     Object Storage
```

The application follows a layered architecture to keep HTTP handling, business logic, data access, and infrastructure concerns separated.

---

## 2. Project Structure

The backend uses a centralized layered structure:

```text
internal/
├── handler/
├── service/
├── repository/
├── middleware/
└── models/
```

### Handler

Responsible for:

* HTTP request handling
* Request parsing
* Input validation
* Calling services
* Returning HTTP responses

### Service

Contains business logic and use cases.

Examples:

* Course ownership checks
* Enrollment rules
* Quiz attempt rules
* Certificate generation conditions
* Payment and instructor payout logic
* Transaction boundaries

### Repository

Responsible for database access.

Examples:

* Creating and retrieving users
* Course queries
* Enrollment queries
* Quiz queries
* Progress queries

### Middleware

Responsible for cross-cutting HTTP concerns.

Examples:

* JWT authentication
* Request ID
* Authorization
* Logging
* Rate limiting

### Models

Contains request, response, database, and domain-related structures.

---

# 3. Authentication

Authentication determines **who the user is**.

The system uses:

* JWT access tokens
* Refresh tokens
* bcrypt password hashing

### Access Token

The client sends the access token using:

```text
Authorization: Bearer <access_token>
```

The middleware verifies:

* Token signature
* Token expiration
* Token validity

The user ID and role are extracted from the token and passed to the application context.

Access tokens are short-lived.

### Refresh Token

Refresh tokens are stored in the database as hashes.

Each login/session creates a separate refresh token.

When a refresh request is made:

```text
Client
  ↓
Refresh Token
  ↓
Hash Token
  ↓
Find Token in Database
  ↓
Check Expiration & User Status
  ↓
Delete Old Refresh Token
  ↓
Create New Refresh Token
  ↓
Return New Access + Refresh Tokens
```

This provides refresh token rotation.

### Logout

Logout removes the corresponding refresh token from the database.

Other active sessions remain unaffected.

### Password Reset

Password reset tokens are stored temporarily in Redis with a short TTL.

After the password is successfully changed:

* The reset token is deleted.
* The password is replaced with a new bcrypt hash.
* All existing refresh tokens for the user are revoked.

---

# 4. Authorization

Authorization determines **what the authenticated user is allowed to do**.

The system uses **Casbin** for role-based access control.

Roles:

* SuperAdmin
* Instructor
* Student

The authorization flow is:

```text
JWT Authentication
        ↓
     user_id
        ↓
       role
        ↓
      Casbin
        ↓
   Handler / Service
```

Casbin handles general permissions such as:

```text
Instructor → course:update
Student    → course:read
SuperAdmin → user:create
```

However, role permission alone is not enough.

The service layer handles resource ownership and business rules.

For example:

```text
Instructor
    ↓
PUT /courses/:id
    ↓
Casbin: course:update
    ↓
Service:
course.instructor_id == authenticated_user_id
```

Therefore:

* **Casbin** answers: "Can this role perform this action?"
* **Service** answers: "Can this user perform it on this specific resource?"

---

# 5. Error Handling

The API uses a consistent error response format.

Example:

```json
{
  "success": false,
  "error": {
    "code": "INVALID_CREDENTIALS",
    "message": "Invalid username or password"
  }
}
```

Common HTTP status codes:

| Status | Usage                              |
| ------ | ---------------------------------- |
| 400    | Bad request / malformed request    |
| 401    | Authentication required or invalid |
| 403    | Permission denied                  |
| 404    | Resource not found                 |
| 409    | Resource conflict                  |
| 422    | Business validation error          |
| 500    | Unexpected server error            |

The service and repository layers use typed application errors where appropriate.

The handler layer maps these errors to HTTP responses.

Internal errors are not exposed directly to clients.

---

# 6. Validation

Validation is performed at multiple levels.

## Request Validation

The handler validates request data such as:

* Required fields
* String lengths
* Email format
* Password requirements
* Numeric ranges
* Request structure

## Business Validation

The service validates business rules such as:

* Instructor can modify only their own courses.
* Student must be enrolled before accessing protected course content.
* A course must satisfy required conditions before publishing.
* Quiz attempt limits must be respected.
* A student must complete required lessons and pass the final quiz before receiving a certificate.

## Database Validation

PostgreSQL enforces data integrity using:

* `NOT NULL`
* `UNIQUE`
* Foreign keys
* `CHECK`
* Partial unique indexes

Frontend validation is considered a user-experience feature and is never trusted as a security boundary.

---

# 7. Database

The primary database is PostgreSQL.

The database is designed around the following main entities:

```text
Users
Roles
Courses
Categories
Modules
Lessons
Lesson Materials
Enrollments
Payments
Instructor Payouts
Lesson Progress
Quizzes
Questions
Question Options
Quiz Attempts
Attempt Answers
Certificates
Reviews
Refresh Tokens
```

The complete entity relationship diagram is available in:

```text
docs/erd.png
```

The database schema is designed using DBML and can be viewed using dbdiagram.io.

---

# 8. Transactions

Transactions are controlled at the service/use-case level.

A transaction is used when several database operations must succeed or fail together.

### Example: Course Enrollment

For a paid course:

```text
BEGIN
  │
  ├── Create Enrollment
  ├── Create Payment
  └── Create Instructor Payout
  │
COMMIT
```

If any operation fails:

```text
ROLLBACK
```

### Example: Quiz Submission

Quiz submission is also transactional:

```text
BEGIN
  │
  ├── Validate Answers
  ├── Store Attempt Answers
  ├── Calculate Score
  └── Update Quiz Attempt
  │
COMMIT
```

The repository provides transaction support, while the service determines where a transaction is required.

---

# 9. Redis

Redis is used as an additional infrastructure component.

Primary use cases:

* Caching
* Temporary password reset data
* Temporary authentication-related data

The application follows a cache-aside approach.

Example:

```text
GET /courses
      │
      ▼
   Redis
   /   \
 HIT   MISS
  │      │
  │      ▼
  │   PostgreSQL
  │      │
  │      ▼
  │    Redis
  │
  ▼
Response
```

Potential cache keys include:

```text
courses:published
course:{course_id}
categories
```

When relevant data changes, the corresponding cache is invalidated.

Rate limiting may also use Redis and is considered as a later infrastructure layer.

---

# 10. Object Storage

MinIO is used for storing large binary objects.

The following data is stored in MinIO:

* Course covers
* Videos
* Files
* Certificate PDFs

PostgreSQL stores metadata and object keys rather than binary files.

Example:

```text
courses/{courseID}/cover/{uuid}.jpg

courses/{courseID}/lessons/{lessonID}/videos/{uuid}.mp4

courses/{courseID}/lessons/{lessonID}/files/{uuid}.pdf

certificates/{certificateID}/certificate.pdf
```

## Upload Flow

Large files are uploaded directly from the client to MinIO using a presigned URL.

```text
Client
  │
  │ Request upload URL
  ▼
Go API
  │
  │ Authorization
  ▼
Presigned URL
  │
  ▼
Client ───────────► MinIO
  │
  │ Upload complete
  ▼
Go API
  │
  ▼
PostgreSQL
```

The backend controls the object key and verifies the upload before storing the material metadata.

Protected files are stored in a private bucket.

Downloads use short-lived presigned URLs after authorization.

---

# 11. Course and Content Access

Course visibility depends on the course status and user role.

Courses have:

```text
DRAFT
PUBLISHED
```

Students can access only published courses.

Instructors can access and manage their own courses.

SuperAdmins can manage all courses.

Ownership is checked through relationships such as:

```text
Lesson
  ↓
Module
  ↓
Course
  ↓
Instructor
```

This allows the service layer to determine whether an instructor owns a particular lesson or module.

---

# 12. Enrollment and Progress

Students can enroll in published courses.

For free courses:

```text
Course
  ↓
Enrollment
  ↓
ACTIVE
```

For paid courses:

```text
Course
  ↓
Enrollment
  ↓
Payment
  ↓
Instructor Payout
```

These operations are performed within a transaction.

Lesson progress is stored per student and lesson.

Course progress is derived from lesson progress rather than stored as a separate percentage.

Only required lessons (`is_preview = false`) are considered when determining course completion.

---

# 13. Quiz System

A quiz can belong to:

* A course
* A module

The relationship is exclusive:

```text
Course Quiz:
course_id != NULL
module_id = NULL

Module Quiz:
course_id = NULL
module_id != NULL
```

A course-level quiz represents the final course quiz.

Only one course-level quiz is allowed per course.

Supported question types:

```text
SINGLE_CHOICE
MULTIPLE_CHOICE
TRUE_FALSE
```

Correct answers are validated by the service layer.

Correct answer information is never exposed to students through the API.

Quiz options can be randomized when presented to students.

---

# 14. Quiz Attempts

Students create an attempt before submitting answers.

The system checks:

* Student enrollment
* Maximum attempts
* Quiz availability

The server is responsible for enforcing the quiz time limit.

The frontend timer is not trusted for security.

Answers are submitted together:

```json
{
  "answers": [
    {
      "question_id": "q1",
      "option_ids": ["o1"]
    },
    {
      "question_id": "q2",
      "option_ids": ["o4", "o5"]
    }
  ]
}
```

The submission is processed inside a database transaction.

---

# 15. Course Completion and Certificates

A student completes a course when:

1. All required lessons are completed.
2. The final course-level quiz is passed.

After these conditions are satisfied:

```text
Enrollment
    ↓
COMPLETED
    ↓
Certificate Generated
```

Certificates are generated automatically.

The certificate contains:

* Student
* Course
* Instructor
* Completion date
* Unique certificate ID
* QR code

The generated certificate PDF is stored in MinIO.

The database stores its object key.

The QR code points to a public certificate verification endpoint:

```text
GET /api/v1/certificates/verify/:uniqueId
```

Each student can have only one certificate for a course.

---

# 16. Reviews

Students can leave a review only after completing the course.

Each student can have one review per course.

Review fields include:

* Rating: 1–5
* Comment

The average course rating is calculated dynamically from reviews and is not stored as a separate database field.

---

# 17. Payments and Instructor Payouts

Payment processing is currently emulated.

For paid enrollment:

```text
Course Price
    ↓
Payment
    ↓
Instructor Payout
```

The frontend never provides the payment amount.

The backend uses the course price stored in PostgreSQL.

Instructor payout policy is defined at the course level:

```text
PERCENTAGE
FIXED
```

When an enrollment is created, the payout record stores a snapshot of the policy:

```text
type
value
amount
```

This preserves historical financial data even if the course payout policy changes later.

---

# 18. Financial Reporting

No separate profit/loss table is required.

Financial information is derived from existing records.

```text
Revenue
  = PAID payments

Expenses
  = Instructor payouts

Net Profit / Loss
  = Revenue - Expenses
```

Financial reporting is available only to SuperAdmins.

Reports can be filtered by:

* Day
* Week
* Month
* Custom date range

---

# 19. Logging and Observability

The backend uses structured logging.

Important log fields include:

* Request ID
* User ID
* HTTP method
* Request path
* Status code
* Request duration
* Error information

Example:

```json
{
  "level": "error",
  "request_id": "req-123",
  "user_id": "user-123",
  "method": "POST",
  "path": "/api/v1/quizzes/123/attempts",
  "status": 500,
  "error": "database error"
}
```

Sensitive information must never be logged.

This includes:

* Passwords
* Access tokens
* Refresh tokens
* Password reset tokens
* Other secrets

---

# 22. Deployment

The initial deployment architecture is containerized using Docker.

The development/initial deployment environment can contain:

```text
Docker Compose
│
├── Go API
├── PostgreSQL
├── Redis
├── MinIO
```
