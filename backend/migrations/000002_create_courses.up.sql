CREATE TYPE course_status AS ENUM (
    'draft',
    'published'
);

CREATE TYPE course_difficulty AS ENUM (
    'beginner',
    'intermediate',
    'advanced'
);

CREATE TYPE payout_type AS ENUM (
    'percentage',
    'fixed'
);

CREATE TYPE lesson_material_type AS ENUM (
    'text',
    'video',
    'file'
);

CREATE TABLE categories (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE courses (
    id UUID PRIMARY KEY,
    instructor_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES categories(id),
    title VARCHAR(255) NOT NULL,
    cover VARCHAR(500),
    description TEXT,
    difficulty course_difficulty,
    total_duration INT,
    language VARCHAR(50),
    status course_status NOT NULL DEFAULT 'draft',
    price DECIMAL(12, 2) NOT NULL DEFAULT 0,
    payout_type payout_type,
    payout_value DECIMAL(12, 2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CHECK (price >= 0),

    CHECK (
        (payout_type IS NULL AND payout_value IS NULL)
        OR (payout_type = 'percentage' AND payout_value BETWEEN 0 AND 100)
        OR (payout_type = 'fixed' AND payout_value >= 0)
    )
);

CREATE TABLE course_learning_outcomes (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses(id),
    content TEXT NOT NULL,
    position INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE course_requirements (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses(id),
    content TEXT NOT NULL,
    position INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE modules (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    order_number INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE lessons (
    id UUID PRIMARY KEY,
    module_id UUID NOT NULL REFERENCES modules(id),
    title VARCHAR(255) NOT NULL,
    duration INT,
    order_number INT NOT NULL,
    is_preview BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE lesson_materials (
    id UUID PRIMARY KEY,
    lesson_id UUID NOT NULL REFERENCES lessons(id),
    type lesson_material_type NOT NULL,
    content TEXT,
    object_key VARCHAR(500),
    file_name VARCHAR(255),
    mime_type VARCHAR(100),
    file_size BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,

    CHECK (
        (type IN ('text', 'video') AND content IS NOT NULL)
        OR (type = 'file' AND object_key IS NOT NULL)
    )
);

CREATE UNIQUE INDEX categories_name_unique_active
ON categories(name)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX modules_course_order_unique
ON modules (course_id, order_number)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX lessons_module_order_unique
ON lessons (module_id, order_number)
WHERE deleted_at IS NULL;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX courses_title_trgm_idx
ON courses USING gin (title gin_trgm_ops)
WHERE deleted_at IS NULL;

CREATE INDEX courses_instructor_id_idx ON courses (instructor_id);
CREATE INDEX courses_category_id_idx ON courses (category_id);
CREATE INDEX courses_status_idx ON courses (status) WHERE deleted_at IS NULL;

CREATE INDEX course_learning_outcomes_course_id_idx ON course_learning_outcomes (course_id);
CREATE INDEX course_requirements_course_id_idx ON course_requirements (course_id);
CREATE INDEX lesson_materials_lesson_id_idx ON lesson_materials (lesson_id);
