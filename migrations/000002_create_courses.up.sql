CREATE TYPE course_status AS ENUM (
    'draft',
    'published'
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
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE courses (
    id UUID PRIMARY KEY,
    instructor_id UUID NOT NULL REFERENCES users(id),
    category_id UUID NOT NULL REFERENCES categories(id),
    title VARCHAR(255) NOT NULL,
    cover VARCHAR(500),
    description TEXT,
    difficulty VARCHAR(50),
    total_duration INT,
    language VARCHAR(50),
    status course_status NOT NULL DEFAULT 'draft',
    price DECIMAL(12, 2) NOT NULL DEFAULT 0,
    payout_type payout_type,
    payout_value DECIMAL(12, 2),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,

    CHECK (price >= 0),

    CHECK (
        payout_type IS NULL
        OR (
            payout_type = 'percentage'
            AND payout_value IS NOT NULL
            AND payout_value BETWEEN 0 AND 100
        )
        OR (
            payout_type = 'fixed'
            AND payout_value IS NOT NULL
            AND payout_value >= 0
        )
    )
);

CREATE TABLE modules (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses(id),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    order_number INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE TABLE lessons (
    id UUID PRIMARY KEY,
    module_id UUID NOT NULL REFERENCES modules(id),
    title VARCHAR(255) NOT NULL,
    duration INT,
    order_number INT NOT NULL,
    is_preview BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
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
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE UNIQUE INDEX modules_course_order_unique
ON modules (course_id, order_number)
WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX lessons_module_order_unique
ON lessons (module_id, order_number)
WHERE deleted_at IS NULL;