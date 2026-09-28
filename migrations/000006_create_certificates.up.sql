CREATE TABLE certificates (
    id UUID PRIMARY KEY,
    student_id UUID NOT NULL REFERENCES users(id),
    course_id UUID NOT NULL REFERENCES courses(id),
    completion_date TIMESTAMP NOT NULL,
    unique_id VARCHAR(100) NOT NULL UNIQUE,
    qr_code TEXT,
    object_key VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (student_id, course_id)
);