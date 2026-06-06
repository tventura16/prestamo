package service

import (
	"math"
	"testing"
	"time"

	"github.com/prestamos/loan-service/internal/models"
)

func sumaCapitales(plan []CuotaPlan) float64 {
	var s float64
	for _, c := range plan {
		s += c.Capital
	}
	return round2(s)
}

// El invariante financiero central: la suma de capitales debe igualar
// exactamente el monto prestado (el último capital absorbe el redondeo).
func TestSumaCapitalesIgualaMonto(t *testing.T) {
	casos := []struct {
		monto float64
		tasa  float64
		n     int
	}{
		{1000, 0.05, 12},
		{1000, 0, 10},
		{15000.50, 0.025, 24},
		{500, 0.10, 6},
		{99999.99, 0.0333, 36},
		{1, 0.05, 3}, // monto chico, fuerza redondeo
	}
	for _, c := range casos {
		plan := GenerarPlanFrances(c.monto, c.tasa, c.n, models.FrecuenciaMensual, time.Now())
		if got := sumaCapitales(plan); got != round2(c.monto) {
			t.Errorf("monto=%.2f tasa=%.4f n=%d: suma capitales=%.2f, esperado=%.2f",
				c.monto, c.tasa, c.n, got, c.monto)
		}
		if len(plan) != c.n {
			t.Errorf("monto=%.2f: se generaron %d cuotas, esperado %d", c.monto, len(plan), c.n)
		}
	}
}

// Tasa 0: cuota = monto/n, sin interés.
func TestTasaCero(t *testing.T) {
	plan := GenerarPlanFrances(1000, 0, 10, models.FrecuenciaMensual, time.Now())
	for _, c := range plan {
		if c.Interes != 0 {
			t.Errorf("cuota %d: interés=%.2f, esperado 0 con tasa 0", c.Numero, c.Interes)
		}
	}
	if sumaCapitales(plan) != 1000 {
		t.Errorf("suma capitales=%.2f, esperado 1000", sumaCapitales(plan))
	}
}

// Amortización francesa: cuota constante (salvo el ajuste de redondeo final),
// interés decreciente y capital creciente.
func TestCuotaConstanteYProgresion(t *testing.T) {
	plan := GenerarPlanFrances(1000, 0.05, 12, models.FrecuenciaMensual, time.Now())

	// Cuota constante en las cuotas intermedias.
	cuota := plan[0].Total
	for i := 0; i < len(plan)-1; i++ {
		if math.Abs(plan[i].Total-cuota) > 0.01 {
			t.Errorf("cuota %d total=%.2f difiere de la cuota constante %.2f",
				plan[i].Numero, plan[i].Total, cuota)
		}
	}
	// Interés decreciente, capital creciente.
	for i := 1; i < len(plan); i++ {
		if plan[i].Interes > plan[i-1].Interes+0.001 {
			t.Errorf("interés no decreciente entre cuota %d (%.2f) y %d (%.2f)",
				plan[i-1].Numero, plan[i-1].Interes, plan[i].Numero, plan[i].Interes)
		}
		if plan[i].Capital < plan[i-1].Capital-0.001 {
			t.Errorf("capital no creciente entre cuota %d (%.2f) y %d (%.2f)",
				plan[i-1].Numero, plan[i-1].Capital, plan[i].Numero, plan[i].Capital)
		}
	}
}

// Cada cuota: total == capital + interés.
func TestTotalEsCapitalMasInteres(t *testing.T) {
	plan := GenerarPlanFrances(15000, 0.03, 18, models.FrecuenciaQuincenal, time.Now())
	for _, c := range plan {
		if round2(c.Capital+c.Interes) != c.Total {
			t.Errorf("cuota %d: capital(%.2f)+interés(%.2f) != total(%.2f)",
				c.Numero, c.Capital, c.Interes, c.Total)
		}
		if c.SaldoPendiente != c.Total {
			t.Errorf("cuota %d: saldo_pendiente inicial (%.2f) debe igualar total (%.2f)",
				c.Numero, c.SaldoPendiente, c.Total)
		}
	}
}

// Fechas de vencimiento según frecuencia.
func TestVencimientoPorFrecuencia(t *testing.T) {
	desembolso := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	casos := []struct {
		frec     models.Frecuencia
		cuota    int
		esperado time.Time
	}{
		{models.FrecuenciaDiaria, 1, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{models.FrecuenciaSemanal, 1, time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC)},
		{models.FrecuenciaQuincenal, 2, time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)},
		{models.FrecuenciaMensual, 3, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range casos {
		got := vencimiento(desembolso, c.frec, c.cuota)
		if !got.Equal(c.esperado) {
			t.Errorf("frecuencia=%s cuota=%d: vencimiento=%s, esperado=%s",
				c.frec, c.cuota, got.Format("2006-01-02"), c.esperado.Format("2006-01-02"))
		}
	}
}

// Ejemplo conocido: 1000 al 5% periódico en 12 cuotas → cuota ≈ 112.83.
func TestEjemploConocido(t *testing.T) {
	plan := GenerarPlanFrances(1000, 0.05, 12, models.FrecuenciaMensual, time.Now())
	cuotaEsperada := 112.83 // P·i·(1+i)^n / ((1+i)^n − 1)
	if math.Abs(plan[0].Total-cuotaEsperada) > 0.01 {
		t.Errorf("cuota constante=%.2f, esperado ≈%.2f", plan[0].Total, cuotaEsperada)
	}
	// La primera cuota: interés = 1000·0.05 = 50.00
	if plan[0].Interes != 50.00 {
		t.Errorf("interés cuota 1=%.2f, esperado 50.00", plan[0].Interes)
	}
}
