CREATE TYPE question_type AS ENUM (
    'single_choice',
    'multiple_choice',
    'true_false'
);

CREATE TABLE quizzes (
    id UUID PRIMARY KEY,
    course_id UUID REFERENCES courses(id),
    module_id UUID REFERENCES modules(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    time_limit INT NOT NULL,
    pass_threshold INT NOT NULL,
    max_attempts INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CHECK (
        (course_id IS NOT NULL AND module_id IS NULL)
        OR
        (course_id IS NULL AND module_id IS NOT NULL)
    ),

    CHECK (time_limit > 0),
    CHECK (pass_threshold BETWEEN 0 AND 100),
    CHECK (max_attempts > 0)
);

CREATE UNIQUE INDEX quizzes_one_course_level_unique
ON quizzes (course_id)
WHERE course_id IS NOT NULL
  AND deleted_at IS NULL;

CREATE TABLE questions (
    id UUID PRIMARY KEY,
    quiz_id UUID NOT NULL REFERENCES quizzes(id),
    text TEXT NOT NULL,
    type question_type NOT NULL,
    order_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX questions_quiz_order_unique
ON questions (quiz_id, order_number)
WHERE deleted_at IS NULL;

CREATE TABLE question_options (
    id UUID PRIMARY KEY,
    question_id UUID NOT NULL REFERENCES questions(id),
    option_text TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE quiz_attempts (
    id UUID PRIMARY KEY,
    student_id UUID NOT NULL REFERENCES users(id),
    quiz_id UUID NOT NULL REFERENCES quizzes(id),
    attempt_number INT NOT NULL,
    score INT,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    time_spent INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (student_id, quiz_id, attempt_number)
);

CREATE TABLE attempt_answers (
    id UUID PRIMARY KEY,
    attempt_id UUID NOT NULL REFERENCES quiz_attempts(id),
    question_id UUID NOT NULL REFERENCES questions(id),
    option_id UUID NOT NULL REFERENCES question_options(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (attempt_id, question_id, option_id)
);