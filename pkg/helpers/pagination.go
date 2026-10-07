package helpers

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

// Pagination returns a valid limit and offset for a 1-based page number.
func Pagination(page, limit int) (int, int) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if page <= 0 {
		page = 1
	}

	return limit, (page - 1) * limit
}
