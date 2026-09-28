CREATE TYPE payment_status AS ENUM (
    'paid'
);

CREATE TABLE payments (
    id UUID PRIMARY KEY,
    enrollment_id UUID NOT NULL UNIQUE REFERENCES enrollments(id),
    amount DECIMAL(12, 2) NOT NULL,
    status payment_status NOT NULL DEFAULT 'paid',
    paid_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (amount >= 0)
);

CREATE TABLE instructor_payouts (
    id UUID PRIMARY KEY,
    instructor_id UUID NOT NULL REFERENCES users(id),
    course_id UUID NOT NULL REFERENCES courses(id),
    enrollment_id UUID NOT NULL UNIQUE REFERENCES enrollments(id),
    type payout_type NOT NULL,
    value DECIMAL(12, 2) NOT NULL,
    amount DECIMAL(12, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (value >= 0),
    CHECK (amount >= 0),

    CHECK (
        (type = 'percentage' AND value BETWEEN 0 AND 100)
        OR
        (type = 'fixed' AND value >= 0)
    )
);