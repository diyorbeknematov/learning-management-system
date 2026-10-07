CREATE TYPE enrollment_status AS ENUM (
    'active',
    'completed',
    'dropped'
);

CREATE TABLE enrollments (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses(id),
    student_id UUID NOT NULL REFERENCES users(id),
    status enrollment_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX enrollments_student_course_unique
ON enrollments (student_id, course_id)
WHERE deleted_at IS NULL;

CREATE TABLE lesson_progress (
    id UUID PRIMARY KEY,
    student_id UUID NOT NULL REFERENCES users(id),
    lesson_id UUID NOT NULL REFERENCES lessons(id),
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (student_id, lesson_id)
);