// Package export convierte reportes a formatos descargables (CSV, XLSX, PDF)
// a partir de una representación tabular común, de modo que cada reporte solo
// declara sus hojas/columnas y no conoce los detalles de cada formato.
package export

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// Sheet es una tabla con encabezados y filas. Los reportes de un solo registro
// (dashboard, diario, mensual) se modelan como una hoja clave/valor.
type Sheet struct {
	Title   string
	Headers []string
	Rows    [][]string
}

// Report es un conjunto de hojas con un nombre de archivo base (sin extensión)
// y un título legible para el encabezado del PDF.
type Report struct {
	Filename string
	Title    string
	Sheets   []Sheet
}

// Format indica el formato solicitado vía ?format=.
type Format string

const (
	FormatJSON Format = "json"
	FormatCSV  Format = "csv"
	FormatXLSX Format = "xlsx"
	FormatPDF  Format = "pdf"
)

// ParseFormat normaliza el query param; "" => json.
func ParseFormat(s string) (Format, bool) {
	switch Format(strings.ToLower(s)) {
	case "", FormatJSON:
		return FormatJSON, true
	case FormatCSV:
		return FormatCSV, true
	case FormatXLSX:
		return FormatXLSX, true
	case FormatPDF:
		return FormatPDF, true
	default:
		return "", false
	}
}

// Respond escribe el reporte en el formato pedido con los headers de descarga
// adecuados. No maneja json (eso queda en el handler, que ya tiene el DTO).
func Respond(c *gin.Context, f Format, r Report) {
	var (
		data        []byte
		contentType string
		ext         string
		err         error
	)
	switch f {
	case FormatCSV:
		data, err = r.toCSV()
		contentType, ext = "text/csv; charset=utf-8", "csv"
	case FormatXLSX:
		data, err = r.toXLSX()
		contentType, ext = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"
	case FormatPDF:
		data, err = r.toPDF()
		contentType, ext = "application/pdf", "pdf"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "format inválido: usa json|csv|xlsx|pdf"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el reporte: " + err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, r.Filename, ext))
	c.Data(http.StatusOK, contentType, data)
}

// toCSV apila las hojas; al haber varias, las separa con una línea en blanco y
// el título de cada hoja (CSV no tiene pestañas).
func (r Report) toCSV() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM UTF-8: Excel respeta los acentos.
	w := csv.NewWriter(&buf)
	for i, s := range r.Sheets {
		if len(r.Sheets) > 1 {
			if i > 0 {
				_ = w.Write([]string{})
			}
			_ = w.Write([]string{s.Title})
		}
		if len(s.Headers) > 0 {
			if err := w.Write(s.Headers); err != nil {
				return nil, err
			}
		}
		for _, row := range s.Rows {
			if err := w.Write(row); err != nil {
				return nil, err
			}
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// toXLSX genera un libro con una pestaña por hoja.
func (r Report) toXLSX() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2D3748"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left"},
	})

	for i, s := range r.Sheets {
		name := sheetName(s.Title, i)
		if i == 0 {
			_ = f.SetSheetName("Sheet1", name)
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}

		row := 1
		if len(s.Headers) > 0 {
			for col, h := range s.Headers {
				cell, _ := excelize.CoordinatesToCellName(col+1, row)
				_ = f.SetCellValue(name, cell, h)
				_ = f.SetCellStyle(name, cell, cell, headerStyle)
			}
			row++
		}
		for _, r := range s.Rows {
			for col, v := range r {
				cell, _ := excelize.CoordinatesToCellName(col+1, row)
				_ = f.SetCellValue(name, cell, v)
			}
			row++
		}
		// Ancho de columnas legible.
		if n := maxCols(s); n > 0 {
			last, _ := excelize.ColumnNumberToName(n)
			_ = f.SetColWidth(name, "A", last, 20)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// toPDF dibuja un encabezado y una tabla por hoja con anchos uniformes.
func (r Report) toPDF() ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "") // landscape: caben más columnas
	pdf.SetMargins(10, 12, 10)
	pdf.AddPage()

	// Transcodifica UTF-8 a cp1252 para que los acentos no salgan corruptos.
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(26, 54, 93)
	pdf.CellFormat(0, 10, tr(r.Title), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 5, "Generado: "+time.Now().Format("2006-01-02 15:04"), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	pageW, _ := pdf.GetPageSize()
	usableW := pageW - 20

	for _, s := range r.Sheets {
		if s.Title != "" && len(r.Sheets) > 1 {
			pdf.SetFont("Helvetica", "B", 12)
			pdf.SetTextColor(45, 55, 72)
			pdf.CellFormat(0, 8, tr(s.Title), "", 1, "L", false, 0, "")
		}
		cols := len(s.Headers)
		if cols == 0 && len(s.Rows) > 0 {
			cols = len(s.Rows[0])
		}
		if cols == 0 {
			continue
		}
		colW := usableW / float64(cols)

		// Encabezado.
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetFillColor(45, 55, 72)
		pdf.SetTextColor(255, 255, 255)
		for _, h := range s.Headers {
			pdf.CellFormat(colW, 7, tr(h), "1", 0, "L", true, 0, "")
		}
		pdf.Ln(-1)

		// Filas.
		pdf.SetFont("Helvetica", "", 8)
		pdf.SetTextColor(40, 40, 40)
		fill := false
		for _, row := range s.Rows {
			if fill {
				pdf.SetFillColor(247, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			for _, v := range row {
				pdf.CellFormat(colW, 6, tr(clip(v, 40)), "1", 0, "L", true, 0, "")
			}
			pdf.Ln(-1)
			fill = !fill
		}
		pdf.Ln(4)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func maxCols(s Sheet) int {
	n := len(s.Headers)
	for _, r := range s.Rows {
		if len(r) > n {
			n = len(r)
		}
	}
	return n
}

// sheetName produce un nombre de pestaña válido (<=31 chars, sin caracteres
// prohibidos) y nunca vacío.
func sheetName(title string, idx int) string {
	name := strings.Map(func(r rune) rune {
		switch r {
		case ':', '\\', '/', '?', '*', '[', ']':
			return '-'
		}
		return r
	}, title)
	if name == "" {
		name = fmt.Sprintf("Hoja %d", idx+1)
	}
	if len(name) > 31 {
		name = name[:31]
	}
	return name
}
