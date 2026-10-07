// Package certpdf draws a certificate of completion as a PDF file: the name of
// the student and the course, the date, the instructor, the id of the
// certificate and a QR code that opens the page that checks it.
package certpdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
)

// The fonts are DejaVu Sans (see fonts/LICENSE). They are inside the program,
// because the standard fonts of PDF have no Cyrillic letters, and the names
// of students can be in any alphabet.
//
//go:embed fonts/DejaVuSans.ttf
var regularFont []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var boldFont []byte

const (
	family = "DejaVu"

	pageWidth  = 297.0 // A4, landscape, in millimetres
	pageHeight = 210.0
	margin     = 18.0
)

// Certificate is what is written on the page.
type Certificate struct {
	StudentName    string
	CourseTitle    string
	InstructorName string
	CompletionDate time.Time
	UniqueID       string
	// VerifyURL is where the QR code leads: the page that checks the
	// certificate.
	VerifyURL string
}

// Render draws the certificate and returns the PDF file. The file is not
// byte for byte the same on every call (the PDF library writes some parts in a
// random order), but it always looks the same.
func Render(c Certificate) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")

	pdf.SetCompression(true)
	pdf.SetCreationDate(c.CompletionDate)
	pdf.SetTitle("Certificate of Completion — "+c.CourseTitle, true)
	pdf.SetAuthor("Learning Management System", true)
	pdf.SetAutoPageBreak(false, 0)

	pdf.AddUTF8FontFromBytes(family, "", regularFont)
	pdf.AddUTF8FontFromBytes(family, "B", boldFont)

	pdf.AddPage()

	border(pdf)

	pdf.SetTextColor(23, 43, 77)
	text(pdf, 34, "B", 38, "CERTIFICATE")

	pdf.SetTextColor(90, 100, 120)
	text(pdf, 53, "", 16, "OF COMPLETION")

	pdf.SetTextColor(70, 80, 100)
	text(pdf, 72, "", 14, "This certifies that")

	pdf.SetTextColor(23, 43, 77)
	fitted(pdf, 90, "B", 32, 16, c.StudentName)

	pdf.SetTextColor(70, 80, 100)
	text(pdf, 108, "", 14, "has successfully completed the course")

	pdf.SetTextColor(23, 43, 77)
	fitted(pdf, 124, "B", 22, 12, c.CourseTitle)

	footer(pdf, c)

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("certpdf: %w", err)
	}

	var out bytes.Buffer

	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("certpdf: %w", err)
	}

	return out.Bytes(), nil
}

// border draws a double frame around the page.
func border(pdf *fpdf.Fpdf) {
	pdf.SetDrawColor(23, 43, 77)
	pdf.SetLineWidth(1.2)
	pdf.Rect(8, 8, pageWidth-16, pageHeight-16, "D")

	pdf.SetDrawColor(180, 150, 80)
	pdf.SetLineWidth(0.4)
	pdf.Rect(11, 11, pageWidth-22, pageHeight-22, "D")
}

// text writes one line, centered.
func text(pdf *fpdf.Fpdf, y float64, style string, size float64, value string) {
	pdf.SetFont(family, style, size)
	pdf.SetXY(margin, y)
	pdf.CellFormat(pageWidth-2*margin, size*0.45, value, "", 0, "C", false, 0, "")
}

// fitted writes one centered line that always fits the page: a long text gets a
// smaller font, down to minSize, and if even that is too wide it is cut with
// "...".
func fitted(pdf *fpdf.Fpdf, y float64, style string, size, minSize float64, value string) {
	width := pageWidth - 2*margin

	for size > minSize {
		pdf.SetFont(family, style, size)

		if pdf.GetStringWidth(value) <= width {
			break
		}

		size--
	}

	pdf.SetFont(family, style, size)

	for pdf.GetStringWidth(value) > width && utf8.RuneCountInString(value) > 1 {
		runes := []rune(value)
		value = string(runes[:len(runes)-2]) + "…"
	}

	text(pdf, y, style, size, value)
}

// footer has the date, the instructor, the id and the QR code.
func footer(pdf *fpdf.Fpdf, c Certificate) {
	const lineY = 168.0

	pdf.SetDrawColor(120, 130, 150)
	pdf.SetLineWidth(0.3)

	// date (left) and instructor (middle), each over a line with its label
	signature(pdf, margin+10, lineY, 70, c.CompletionDate.UTC().Format("2 January 2006"), "Date")
	signature(pdf, margin+100, lineY, 70, c.InstructorName, "Instructor")

	qr(pdf, c.VerifyURL)

	pdf.SetTextColor(90, 100, 120)
	pdf.SetFont(family, "", 9)
	pdf.SetXY(margin, 192)
	pdf.CellFormat(pageWidth-2*margin-45, 4, "Certificate ID: "+c.UniqueID, "", 0, "L", false, 0, "")
}

// signature writes a value above a line and a label below it.
func signature(pdf *fpdf.Fpdf, x, y, width float64, value, label string) {
	pdf.SetTextColor(23, 43, 77)

	size := 13.0

	for size > 8 {
		pdf.SetFont(family, "", size)

		if pdf.GetStringWidth(value) <= width {
			break
		}

		size--
	}

	pdf.SetFont(family, "", size)

	for pdf.GetStringWidth(value) > width && utf8.RuneCountInString(value) > 1 {
		runes := []rune(value)
		value = string(runes[:len(runes)-2]) + "…"
	}

	pdf.SetXY(x, y-8)
	pdf.CellFormat(width, 6, value, "", 0, "C", false, 0, "")

	pdf.Line(x, y, x+width, y)

	pdf.SetTextColor(90, 100, 120)
	pdf.SetFont(family, "", 9)
	pdf.SetXY(x, y+1.5)
	pdf.CellFormat(width, 4, label, "", 0, "C", false, 0, "")
}

// qr draws the QR code of the address in the right bottom corner.
func qr(pdf *fpdf.Fpdf, address string) {
	const (
		size = 30.0
		x    = pageWidth - margin - size - 8
		y    = 150.0
	)

	png, err := qrcode.Encode(address, qrcode.Medium, 256)
	if err != nil {
		pdf.SetError(fmt.Errorf("qr code: %w", err))

		return
	}

	pdf.RegisterImageOptionsReader("qr", fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(png))
	pdf.ImageOptions("qr", x, y, size, size, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	pdf.SetTextColor(90, 100, 120)
	pdf.SetFont(family, "", 8)
	pdf.SetXY(x-5, y+size+1)
	pdf.CellFormat(size+10, 4, "Scan to verify", "", 0, "C", false, 0, "")
}
