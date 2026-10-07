package csvexport_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/pkg/csvexport"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type row struct {
	ID       uuid.UUID  `csv:"id"`
	Name     string     `csv:"name"`
	Count    int        `csv:"count"`
	Rate     float64    `csv:"rate"`
	Active   bool       `csv:"active"`
	At       time.Time  `csv:"at"`
	LastSeen *time.Time `csv:"last_seen"`
	Note     *string    `csv:"note"`
	Skipped  string
	Hidden   string `csv:"-"`
}

func write(t *testing.T, rows any) string {
	t.Helper()

	var out bytes.Buffer

	require.NoError(t, csvexport.Write(&out, rows))

	return out.String()
}

func TestWrite(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	at := time.Date(2026, 5, 10, 12, 30, 0, 0, time.UTC)
	note := "ok"

	out := write(t, []row{
		{ID: id, Name: "Ali", Count: 3, Rate: 66.67, Active: true, At: at, LastSeen: &at, Note: &note, Skipped: "x", Hidden: "y"},
		{ID: id, Name: "Vali", Count: 0, Rate: 100, Active: false, At: at},
	})

	require.Equal(t, "id,name,count,rate,active,at,last_seen,note\n"+
		"11111111-2222-3333-4444-555555555555,Ali,3,66.67,true,2026-05-10T12:30:00Z,2026-05-10T12:30:00Z,ok\n"+
		"11111111-2222-3333-4444-555555555555,Vali,0,100,false,2026-05-10T12:30:00Z,,\n", out)
}

func TestWrite_Pointers(t *testing.T) {
	out := write(t, []*row{{Name: "Ali"}})

	require.Contains(t, out, "Ali")
}

func TestWrite_EmptySliceGivesTheHeader(t *testing.T) {
	require.Equal(t, "id,name,count,rate,active,at,last_seen,note\n", write(t, []row{}))
	require.Equal(t, "id,name,count,rate,active,at,last_seen,note\n", write(t, []row(nil)))
}

func TestWrite_QuotesCommasQuotesAndLineBreaks(t *testing.T) {
	out := write(t, []row{{Name: "Doe, John \"JD\"\nSecond line"}})

	require.Contains(t, out, `"Doe, John ""JD""`+"\n"+`Second line"`)
}

func TestWrite_FormulasAreNotRun(t *testing.T) {
	for _, name := range []string{"=1+1", "+1", "-1", "@SUM(A1)", "\tcmd"} {
		out := write(t, []row{{Name: name}})

		require.Contains(t, out, "'"+name, name)
	}

	require.Contains(t, write(t, []row{{Name: "Ali=1"}}), ",Ali=1,", "only the start of a cell matters")
}

func TestWrite_NegativeNumbersStayNumbers(t *testing.T) {
	out := write(t, []row{{Rate: -12.5, Count: -3}})

	require.Contains(t, out, ",-3,-12.5,")
	require.NotContains(t, out, "'-")
}

func TestWrite_TimesAreInUTC(t *testing.T) {
	tashkent := time.FixedZone("UTC+5", 5*60*60)
	local := time.Date(2026, 5, 10, 17, 0, 0, 0, tashkent)

	require.Contains(t, write(t, []row{{At: local}}), "2026-05-10T12:00:00Z")
}

func TestWrite_Errors(t *testing.T) {
	var out bytes.Buffer

	require.Error(t, csvexport.Write(&out, nil))
	require.Error(t, csvexport.Write(&out, "text"))
	require.Error(t, csvexport.Write(&out, row{}))
	require.Error(t, csvexport.Write(&out, []string{"a"}))
	require.Error(t, csvexport.Write(&out, []struct{ A int }{{1}}), "a struct without csv tags")
}
