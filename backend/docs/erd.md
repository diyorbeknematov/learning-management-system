# Database ERD

Source of truth: `migrations/`. `deleted_at` columns (soft delete) are on users, categories, courses, modules, lessons, lesson_materials, enrollments, quizzes, questions and question_options.

Enums: `user_status` (active, blocked), `course_status` (draft, published), `course_difficulty` (beginner, intermediate, advanced), `payout_type` (percentage, fixed), `lesson_material_type` (text, video, file), `enrollment_status` (active, completed, dropped), `question_type` (single_choice, multiple_choice, true_false), `payment_status` (paid).

```mermaid
erDiagram
    roles ||--o{ users : "has"
    users ||--o{ refresh_tokens : "owns"
    users ||--o{ courses : "teaches"
    categories ||--o{ courses : "groups"
    courses ||--o{ course_learning_outcomes : "has"
    courses ||--o{ course_requirements : "has"
    courses ||--o{ modules : "has"
    modules ||--o{ lessons : "has"
    lessons ||--o{ lesson_materials : "has"
    courses ||--o{ enrollments : "has"
    users ||--o{ enrollments : "student"
    users ||--o{ lesson_progress : "student"
    lessons ||--o{ lesson_progress : "tracked"
    courses |o--o{ quizzes : "final quiz"
    modules |o--o{ quizzes : "module quiz"
    quizzes ||--o{ questions : "has"
    questions ||--o{ question_options : "has"
    users ||--o{ quiz_attempts : "student"
    quizzes ||--o{ quiz_attempts : "taken"
    quiz_attempts ||--o{ attempt_answers : "has"
    questions ||--o{ attempt_answers : "answered"
    question_options ||--o{ attempt_answers : "chosen"
    enrollments ||--o| payments : "paid by"
    enrollments ||--o| instructor_payouts : "pays out"
    users ||--o{ instructor_payouts : "instructor"
    courses ||--o{ instructor_payouts : "for"
    users ||--o{ certificates : "earned"
    courses ||--o{ certificates : "for"
    users ||--o{ reviews : "writes"
    courses ||--o{ reviews : "gets"

    roles {
        uuid id PK
        varchar name UK
    }
    users {
        uuid id PK
        varchar username
        varchar password
        varchar first_name
        varchar last_name
        varchar email
        varchar avatar
        text bio
        uuid role_id FK
        user_status status
        timestamptz deleted_at
    }
    refresh_tokens {
        uuid id PK
        uuid user_id FK
        text token_hash UK
        timestamptz expires_at
    }
    categories {
        uuid id PK
        varchar name
        text description
    }
    courses {
        uuid id PK
        uuid instructor_id FK
        uuid category_id FK
        varchar title
        varchar cover
        text description
        course_difficulty difficulty
        int total_duration
        varchar language
        course_status status
        decimal price
        payout_type payout_type
        decimal payout_value
    }
    course_learning_outcomes {
        uuid id PK
        uuid course_id FK
        text content
        int position
    }
    course_requirements {
        uuid id PK
        uuid course_id FK
        text content
        int position
    }
    modules {
        uuid id PK
        uuid course_id FK
        varchar title
        text description
        int order_number
    }
    lessons {
        uuid id PK
        uuid module_id FK
        varchar title
        int duration
        int order_number
        bool is_preview
    }
    lesson_materials {
        uuid id PK
        uuid lesson_id FK
        lesson_material_type type
        text content
        varchar object_key
        varchar file_name
        varchar mime_type
        bigint file_size
    }
    enrollments {
        uuid id PK
        uuid course_id FK
        uuid student_id FK
        enrollment_status status
    }
    lesson_progress {
        uuid id PK
        uuid student_id FK
        uuid lesson_id FK
        bool completed
        timestamptz completed_at
    }
    quizzes {
        uuid id PK
        uuid course_id FK "or module_id"
        uuid module_id FK "or course_id"
        varchar title
        text description
        int time_limit
        int pass_threshold
        int max_attempts
    }
    questions {
        uuid id PK
        uuid quiz_id FK
        text text
        question_type type
        int order_number
    }
    question_options {
        uuid id PK
        uuid question_id FK
        text option_text
        bool is_correct
        int position
    }
    quiz_attempts {
        uuid id PK
        uuid student_id FK
        uuid quiz_id FK
        int attempt_number
        int score
        timestamptz started_at
        timestamptz completed_at
        int time_spent
    }
    attempt_answers {
        uuid id PK
        uuid attempt_id FK
        uuid question_id FK
        uuid option_id FK
    }
    payments {
        uuid id PK
        uuid enrollment_id FK, UK
        decimal amount
        payment_status status
        timestamptz paid_at
    }
    instructor_payouts {
        uuid id PK
        uuid instructor_id FK
        uuid course_id FK
        uuid enrollment_id FK, UK
        payout_type type
        decimal value
        decimal amount
    }
    certificates {
        uuid id PK
        uuid student_id FK
        uuid course_id FK
        timestamptz completion_date
        varchar unique_id UK
        text qr_code
        varchar object_key
    }
    reviews {
        uuid id PK
        uuid student_id FK
        uuid course_id FK
        int rating
        text comment
    }
```

A quiz belongs to a course (final quiz) **or** a module, never both (CHECK). A review has a rating of 1-5. Payments and payouts are one-to-one with an enrollment.

Redis (not in the database): `password_reset:<hash>` (one-time reset token), `login_failures:<username>`, `forgot_password:<email>` and `rate:<name>:<ip>` (counters), `revoked_user:<id>` (the time the user's access tokens were ended; lives as long as an access token).  `cache:<scope>:v<N>:<key>` and `cache_version:<scope>` (cached catalog answers and their version).
