package certpdf_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/diyorbeknematov/lms/pkg/certpdf"
	"github.com/stretchr/testify/require"
)

func sample() certpdf.Certificate {
	return certpdf.Certificate{
		StudentName:    "Diyorbek Ne'matov",
		CourseTitle:    "Go Backend Development",
		InstructorName: "Alisher Valiyev",
		CompletionDate: time.Date(2026, 10, 7, 9, 30, 0, 0, time.UTC),
		UniqueID:       "LMS-7K2M-QX9P-4DHB",
		VerifyURL:      "http://localhost:8080/api/v1/certificates/verify/LMS-7K2M-QX9P-4DHB",
	}
}

func TestRender_IsAPDFWithAQRCode(t *testing.T) {
	pdf, err := certpdf.Render(sample())
	require.NoError(t, err)

	require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")))
	require.True(t, bytes.HasSuffix(bytes.TrimSpace(pdf), []byte("%%EOF")))
	require.Greater(t, len(pdf), 10_000, "the fonts are inside")

	require.Contains(t, string(pdf), "/Subtype /Image", "the QR code is a picture on the page")
	require.Contains(t, string(pdf), "841.89 595.28", "A4, landscape")
	require.Equal(t, 1, strings.Count(string(pdf), "/Type /Page\n"), "one page")
}

func TestRender_DifferentCertificatesDiffer(t *testing.T) {
	other := sample()
	other.UniqueID = "LMS-AAAA-BBBB-CCCC"
	other.VerifyURL = "http://localhost:8080/api/v1/certificates/verify/LMS-AAAA-BBBB-CCCC"

	first, err := certpdf.Render(sample())
	require.NoError(t, err)

	second, err := certpdf.Render(other)
	require.NoError(t, err)

	require.NotEqual(t, first, second)
}

func TestRender_AnyAlphabetAndAnyLength(t *testing.T) {
	cases := map[string]certpdf.Certificate{
		"cyrillic": {StudentName: "Алишер Валиев", CourseTitle: "Основы Go", InstructorName: "Зарина Каримова"},
		"uzbek":    {StudentName: "Oʻtkir Gʻaniyev", CourseTitle: "Dasturlash asoslari", InstructorName: "Shahnoza Oʻrinboyeva"},
		"accents":  {StudentName: "José Müller-Łukasz", CourseTitle: "Données & Œuvres", InstructorName: "Zoë Ångström"},
		"very long": {
			StudentName:    strings.Repeat("Abdurahmonjon ", 40),
			CourseTitle:    strings.Repeat("A very long course title ", 40),
			InstructorName: strings.Repeat("Instructor ", 40),
		},
		"one letter": {StudentName: "A", CourseTitle: "B", InstructorName: "C"},
		"empty":      {},
		"html":       {StudentName: "<b>Ali</b> & \"Vali\"", CourseTitle: "<script>alert(1)</script>", InstructorName: "'; DROP TABLE users;--"},
	}

	for name, c := range cases {
		c.CompletionDate = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		c.UniqueID = "LMS-TEST-TEST-TEST"
		c.VerifyURL = "https://lms.example.com/verify/LMS-TEST-TEST-TEST"

		pdf, err := certpdf.Render(c)
		require.NoError(t, err, name)
		require.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")), name)
	}
}

func TestRender_ALongAddressStillMakesAQRCode(t *testing.T) {
	c := sample()
	c.VerifyURL = "https://lms.example.com/verify/" + strings.Repeat("x", 800)

	pdf, err := certpdf.Render(c)
	require.NoError(t, err)
	require.Contains(t, string(pdf), "/Subtype /Image")
}

// pdftotext is not on every machine, so this test is skipped without it. It
// reads the text back from the file, which proves that what is written on
// the page is what was asked for, in any alphabet.
func TestRender_TheTextCanBeReadBack(t *testing.T) {
	reader, err := exec.LookPath("pdftotext")
	if err != nil {
		t.Skip("pdftotext is not installed")
	}

	c := sample()
	c.StudentName = "Алишер Ne'matov"
	c.CourseTitle = "Основы Go"

	pdf, err := certpdf.Render(c)
	require.NoError(t, err)

	file := filepath.Join(t.TempDir(), "certificate.pdf")
	require.NoError(t, os.WriteFile(file, pdf, 0o600))

	output, err := exec.Command(reader, "-layout", file, "-").Output()
	require.NoError(t, err)

	for _, want := range []string{"CERTIFICATE", "Алишер Ne'matov", "Основы Go", "Alisher Valiyev", "7 October 2026", "LMS-7K2M-QX9P-4DHB", "Scan to verify"} {
		require.Contains(t, string(output), want)
	}
}
