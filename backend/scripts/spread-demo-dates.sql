-- Moves the payments, the payouts and the enrollments of the demo data to random
-- days of the last six months, so the charts of the finance page have a history.
-- For demos only: it changes the dates of ALL payments and enrollments.
UPDATE payments SET paid_at = now() - (random() * 180 || ' days')::interval, created_at = paid_at;

UPDATE instructor_payouts p
SET created_at = (SELECT paid_at FROM payments WHERE enrollment_id = p.enrollment_id);

UPDATE enrollments SET created_at = (SELECT paid_at FROM payments WHERE enrollment_id = enrollments.id)
WHERE EXISTS (SELECT 1 FROM payments WHERE enrollment_id = enrollments.id);
