package helpers

// PeriodUnit returns a date_trunc unit for the group_by query parameter and
// falls back to "day" for an empty or unknown value.
func PeriodUnit(groupBy string) string {
	switch groupBy {
	case "week", "month":
		return groupBy
	default:
		return "day"
	}
}
