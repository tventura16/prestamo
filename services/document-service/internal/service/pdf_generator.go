package service

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"

	"github.com/prestamos/document-service/internal/models"
)

const (
	marginX   = 15.0
	pageWidth = 210.0 // A4 portrait mm
)

// addHeader pinta el encabezado común (logo + nombre empresa + fecha).
func addHeader(pdf *gofpdf.Fpdf, titulo string) {
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(26, 54, 93) // azul navy (igual que frontend)
	pdf.CellFormat(0, 10, "Sistema de Prestamos", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 5, "Generado: "+time.Now().Format("2006-01-02 15:04"), "", 1, "L", false, 0, "")

	pdf.Ln(4)
	pdf.SetDrawColor(26, 54, 93)
	pdf.SetLineWidth(0.5)
	pdf.Line(marginX, pdf.GetY(), pageWidth-marginX, pdf.GetY())
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(26, 54, 93)
	pdf.CellFormat(0, 8, titulo, "", 1, "C", false, 0, "")
	pdf.Ln(2)
}

func addFooter(pdf *gofpdf.Fpdf) {
	pdf.SetY(-15)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 5, fmt.Sprintf("Pagina %d", pdf.PageNo()), "", 0, "C", false, 0, "")
}

func clienteSection(pdf *gofpdf.Fpdf, c models.Cliente) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 6, "Datos del cliente", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(0, 0, 0)

	rows := [][2]string{
		{"Nombre", fmt.Sprintf("%s %s", c.Nombres, c.Apellidos)},
		{"CI", c.CI},
		{"Fecha nacimiento", c.FechaNacimiento.Format("2006-01-02")},
	}
	if c.Telefono != nil {
		rows = append(rows, [2]string{"Telefono", *c.Telefono})
	}
	if c.Email != nil {
		rows = append(rows, [2]string{"Email", *c.Email})
	}
	if c.Direccion != nil {
		rows = append(rows, [2]string{"Direccion", *c.Direccion})
	}

	for _, r := range rows {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(45, 6, r[0]+":", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.CellFormat(0, 6, r[1], "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)
}

func keyValueRow(pdf *gofpdf.Fpdf, key, val string) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(50, 6, key+":", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 6, val, "", 1, "L", false, 0, "")
}

func tableHeader(pdf *gofpdf.Fpdf, headers []string, widths []float64) {
	pdf.SetFillColor(26, 54, 93)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 9)
}

// ─────────────────────────────────────────────
// Contrato de préstamo
// ─────────────────────────────────────────────
func GenerarContrato(c models.Cliente, p models.Prestamo, cuotas []models.Cuota) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginX, 15, marginX)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AliasNbPages("")
	pdf.AddPage()

	addHeader(pdf, "CONTRATO DE PRESTAMO")

	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5,
		"Entre la entidad prestamista y el(la) prestatario(a) identificado(a) a continuacion, "+
			"se celebra el presente contrato de prestamo conforme a las siguientes condiciones:", "", "J", false)
	pdf.Ln(3)

	clienteSection(pdf, c)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 6, "Condiciones del prestamo", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	keyValueRow(pdf, "Identificador", p.ID.String())
	keyValueRow(pdf, "Monto aprobado", fmt.Sprintf("BOB %.2f", p.MontoAprobado))
	keyValueRow(pdf, "Tasa de interes", fmt.Sprintf("%.2f %% por periodo", p.TasaInteres*100))
	keyValueRow(pdf, "Cantidad de cuotas", fmt.Sprintf("%d", p.NumCuotas))
	keyValueRow(pdf, "Frecuencia", p.Frecuencia)
	if p.FechaDesembolso != nil {
		keyValueRow(pdf, "Fecha desembolso", p.FechaDesembolso.Format("2006-01-02"))
	}

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 6, "Plan de pagos", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	widths := []float64{15, 30, 30, 30, 30, 35}
	tableHeader(pdf, []string{"#", "Vencimiento", "Capital", "Interes", "Total", "Saldo"}, widths)

	totalCapital, totalInteres, totalCuota := 0.0, 0.0, 0.0
	for _, cu := range cuotas {
		pdf.CellFormat(widths[0], 6, fmt.Sprintf("%d", cu.Numero), "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[1], 6, cu.FechaVencimiento.Format("2006-01-02"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(widths[2], 6, fmt.Sprintf("%.2f", cu.Capital), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[3], 6, fmt.Sprintf("%.2f", cu.Interes), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[4], 6, fmt.Sprintf("%.2f", cu.Total), "1", 0, "R", false, 0, "")
		pdf.CellFormat(widths[5], 6, fmt.Sprintf("%.2f", cu.SaldoPendiente), "1", 1, "R", false, 0, "")
		totalCapital += cu.Capital
		totalInteres += cu.Interes
		totalCuota += cu.Total
	}

	// Fila de totales
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(237, 242, 247)
	pdf.CellFormat(widths[0]+widths[1], 6, "TOTAL", "1", 0, "C", true, 0, "")
	pdf.CellFormat(widths[2], 6, fmt.Sprintf("%.2f", totalCapital), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[3], 6, fmt.Sprintf("%.2f", totalInteres), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[4], 6, fmt.Sprintf("%.2f", totalCuota), "1", 0, "R", true, 0, "")
	pdf.CellFormat(widths[5], 6, "", "1", 1, "", true, 0, "")

	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(0, 5,
		"El prestatario se compromete a pagar las cuotas en las fechas establecidas. "+
			"El incumplimiento generara mora calculada conforme a las politicas de la entidad.", "", "J", false)

	pdf.Ln(15)
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(80, 6, "_______________________________", "", 0, "C", false, 0, "")
	pdf.CellFormat(0, 6, "_______________________________", "", 1, "C", false, 0, "")
	pdf.CellFormat(80, 5, "Prestatario", "", 0, "C", false, 0, "")
	pdf.CellFormat(0, 5, "Prestamista", "", 1, "C", false, 0, "")

	addFooter(pdf)

	return outputBytes(pdf)
}

// ─────────────────────────────────────────────
// Recibo de pago
// ─────────────────────────────────────────────
func GenerarRecibo(c models.Cliente, p models.Pago, cuotaNumero *int) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginX, 15, marginX)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	titulo := "RECIBO DE PAGO"
	if p.NumeroRecibo != nil {
		titulo = "RECIBO " + *p.NumeroRecibo
	}
	addHeader(pdf, titulo)

	keyValueRow(pdf, "Fecha de pago", p.FechaPago.Format("2006-01-02 15:04"))
	keyValueRow(pdf, "Cliente", fmt.Sprintf("%s %s (CI %s)", c.Nombres, c.Apellidos, c.CI))
	keyValueRow(pdf, "Prestamo", p.PrestamoID.String())
	if cuotaNumero != nil {
		keyValueRow(pdf, "Cuota aplicada", fmt.Sprintf("#%d", *cuotaNumero))
	}
	keyValueRow(pdf, "Metodo de pago", p.MetodoPago)
	keyValueRow(pdf, "Tipo de pago", p.Tipo)

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 6, "Desglose del pago", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	widths := []float64{100, 60}
	tableHeader(pdf, []string{"Concepto", "Monto (BOB)"}, widths)

	rows := [][2]string{
		{"Mora",    fmt.Sprintf("%.2f", p.MoraPagada)},
		{"Interes", fmt.Sprintf("%.2f", p.InteresPagado)},
		{"Capital", fmt.Sprintf("%.2f", p.CapitalPagado)},
	}
	for _, r := range rows {
		pdf.CellFormat(widths[0], 7, r[0], "1", 0, "L", false, 0, "")
		pdf.CellFormat(widths[1], 7, r[1], "1", 1, "R", false, 0, "")
	}
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(237, 242, 247)
	pdf.CellFormat(widths[0], 8, "TOTAL PAGADO", "1", 0, "L", true, 0, "")
	pdf.CellFormat(widths[1], 8, fmt.Sprintf("%.2f", p.MontoPagado), "1", 1, "R", true, 0, "")

	if p.Anulado {
		pdf.Ln(8)
		pdf.SetTextColor(220, 38, 38)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 8, "** PAGO ANULADO **", "", 1, "C", false, 0, "")
	}

	addFooter(pdf)
	return outputBytes(pdf)
}

// ─────────────────────────────────────────────
// Estado de cuenta (cliente)
// ─────────────────────────────────────────────
func GenerarEstadoCuenta(c models.Cliente, prestamos []models.Prestamo, pagos []models.Pago) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginX, 15, marginX)
	pdf.SetAutoPageBreak(true, 20)
	pdf.AliasNbPages("")
	pdf.AddPage()

	addHeader(pdf, "ESTADO DE CUENTA")
	clienteSection(pdf, c)

	// Tabla de préstamos
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 6, "Prestamos del cliente", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	if len(prestamos) == 0 {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.CellFormat(0, 6, "Sin prestamos registrados.", "", 1, "L", false, 0, "")
	} else {
		widths := []float64{55, 25, 25, 25, 25, 25}
		tableHeader(pdf, []string{"ID", "Monto", "Cuotas", "Estado", "Solicitud", "Tasa"}, widths)
		for _, p := range prestamos {
			pdf.CellFormat(widths[0], 6, p.ID.String()[:8]+"...", "1", 0, "L", false, 0, "")
			pdf.CellFormat(widths[1], 6, fmt.Sprintf("%.2f", p.MontoAprobado), "1", 0, "R", false, 0, "")
			pdf.CellFormat(widths[2], 6, fmt.Sprintf("%d", p.NumCuotas), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[3], 6, p.Estado, "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[4], 6, p.FechaSolicitud.Format("2006-01-02"), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[5], 6, fmt.Sprintf("%.2f%%", p.TasaInteres*100), "1", 1, "R", false, 0, "")
		}
	}

	pdf.Ln(6)

	// Tabla de pagos
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(45, 55, 72)
	pdf.CellFormat(0, 6, "Historial de pagos", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	if len(pagos) == 0 {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.CellFormat(0, 6, "Sin pagos registrados.", "", 1, "L", false, 0, "")
	} else {
		widths := []float64{30, 28, 25, 25, 25, 27, 20}
		tableHeader(pdf, []string{"Recibo", "Fecha", "Monto", "Capital", "Interes", "Metodo", "Tipo"}, widths)

		total := 0.0
		for _, p := range pagos {
			recibo := ""
			if p.NumeroRecibo != nil {
				recibo = *p.NumeroRecibo
			}
			pdf.CellFormat(widths[0], 6, recibo, "1", 0, "L", false, 0, "")
			pdf.CellFormat(widths[1], 6, p.FechaPago.Format("2006-01-02"), "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[2], 6, fmt.Sprintf("%.2f", p.MontoPagado), "1", 0, "R", false, 0, "")
			pdf.CellFormat(widths[3], 6, fmt.Sprintf("%.2f", p.CapitalPagado), "1", 0, "R", false, 0, "")
			pdf.CellFormat(widths[4], 6, fmt.Sprintf("%.2f", p.InteresPagado), "1", 0, "R", false, 0, "")
			pdf.CellFormat(widths[5], 6, p.MetodoPago, "1", 0, "C", false, 0, "")
			pdf.CellFormat(widths[6], 6, p.Tipo, "1", 1, "C", false, 0, "")
			total += p.MontoPagado
		}

		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(237, 242, 247)
		pdf.CellFormat(widths[0]+widths[1], 7, "TOTAL PAGADO", "1", 0, "L", true, 0, "")
		pdf.CellFormat(widths[2], 7, fmt.Sprintf("%.2f", total), "1", 0, "R", true, 0, "")
		pdf.CellFormat(widths[3]+widths[4]+widths[5]+widths[6], 7, "", "1", 1, "", true, 0, "")
	}

	addFooter(pdf)
	return outputBytes(pdf)
}

// outputBytes serializa el PDF a bytes.
func outputBytes(pdf *gofpdf.Fpdf) ([]byte, error) {
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}
	w := &byteWriter{}
	if err := pdf.Output(w); err != nil {
		return nil, fmt.Errorf("output pdf: %w", err)
	}
	return w.buf, nil
}

type byteWriter struct{ buf []byte }

func (w *byteWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}
