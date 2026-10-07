package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/diyorbeknematov/lms/internal/api/response"
	"github.com/diyorbeknematov/lms/internal/models"
	"github.com/diyorbeknematov/lms/pkg/csvexport"
	"github.com/gin-gonic/gin"
)

// periodQuery is the period of a request: ?from=2026-05-01&to=2026-05-31. A
// date can be a day (2026-05-01) or an exact time (2026-05-01T09:30:00Z). They
// are read as text because the day form is not what Gin reads by itself.
type periodQuery struct {
	From *string `form:"from"`
	To   *string `form:"to"`
}

// dates turns the period into times, or tells which parameter is wrong. A day
// in "from" starts at 00:00 and a day in "to" lasts until its last moment, so
// ?from=2026-05-01&to=2026-05-31 covers the whole of May.
func (q periodQuery) dates() (models.DateRange, map[string]string) {
	var (
		dates  models.DateRange
		errors = map[string]string{}
	)

	if q.From != nil && *q.From != "" {
		from, _, err := parseDate(*q.From)
		if err != nil {
			errors["from"] = "must be a date (2026-05-01) or a time (2026-05-01T09:30:00Z)"
		}

		dates.From = &from
	}

	if q.To != nil && *q.To != "" {
		to, day, err := parseDate(*q.To)
		if err != nil {
			errors["to"] = "must be a date (2026-05-01) or a time (2026-05-01T09:30:00Z)"
		}

		if day {
			to = to.Add(24*time.Hour - time.Nanosecond)
		}

		dates.To = &to
	}

	if len(errors) > 0 {
		return models.DateRange{}, errors
	}

	return dates, nil
}

// parseDate reads a day or an exact time, in UTC. day tells that only the day
// was given.
func parseDate(value string) (parsed time.Time, day bool, err error) {
	if parsed, err = time.Parse(time.RFC3339, value); err == nil {
		return parsed, false, nil
	}

	parsed, err = time.Parse(time.DateOnly, value)

	return parsed, true, err
}

// period reads the period of the query. When it is wrong it writes the 400
// answer and returns false.
func period(c *gin.Context, q periodQuery) (models.DateRange, bool) {
	dates, errors := q.dates()
	if errors != nil {
		response.ValidationFailed(c, errors)

		return models.DateRange{}, false
	}

	return dates, true
}

// writeCSV sends rows as a file to download. The file starts with a byte order
// mark, so Excel reads the Uzbek letters in the names right.
func writeCSV(c *gin.Context, name string, rows any) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+name+"-"+time.Now().UTC().Format(time.DateOnly)+`.csv"`)
	c.Status(http.StatusOK)

	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	if err := csvexport.Write(c.Writer, rows); err != nil {
		// the answer has started, so all that is left is to say it in the log
		slog.ErrorContext(c.Request.Context(), "csv export failed", "report", name, "error", err)
	}
}
