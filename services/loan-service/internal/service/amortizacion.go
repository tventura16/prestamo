package service

import (
	"math"
	"time"

	"github.com/prestamos/loan-service/internal/models"
)

// CuotaPlan representa una cuota calculada antes de insertar en DB.
//
// SaldoPendiente es lo que queda por pagar de ESTA cuota (inicia en Total y
// decrece con cada pago aplicado). Para conocer el saldo restante del
// préstamo, sumar saldo_pendiente de todas las cuotas no pagadas.
type CuotaPlan struct {
	Numero           int
	FechaVencimiento time.Time
	Capital          float64
	Interes          float64
	Total            float64
	SaldoPendiente   float64
}

// GenerarPlanFrances calcula un plan de pagos por amortización francesa
// (cuotas constantes). La tasa se interpreta como periódica, no anual.
//
// Fórmula:
//   cuota = P · i · (1+i)^n / ((1+i)^n − 1)
//   interés_k = saldo_k−1 · i
//   capital_k = cuota − interés_k
//
// Si i == 0, cuota = P/n.
//
// El último capital absorbe el redondeo para que la suma de capitales sea P.
func GenerarPlanFrances(monto, tasa float64, n int, frecuencia models.Frecuencia, fechaDesembolso time.Time) []CuotaPlan {
	plan := make([]CuotaPlan, 0, n)

	var cuotaConstante float64
	if tasa == 0 {
		cuotaConstante = monto / float64(n)
	} else {
		factor := math.Pow(1+tasa, float64(n))
		cuotaConstante = monto * tasa * factor / (factor - 1)
	}
	cuotaConstante = round2(cuotaConstante)

	saldo := monto
	capitalAcumulado := 0.0

	for k := 1; k <= n; k++ {
		var interes, capital, total float64

		if k == n {
			// Última cuota: absorbe redondeo en capital.
			interes = round2(saldo * tasa)
			capital = round2(monto - capitalAcumulado)
			total = round2(capital + interes)
			saldo = 0
		} else {
			interes = round2(saldo * tasa)
			capital = round2(cuotaConstante - interes)
			total = round2(capital + interes)
			capitalAcumulado = round2(capitalAcumulado + capital)
			saldo = round2(saldo - capital)
		}

		plan = append(plan, CuotaPlan{
			Numero:           k,
			FechaVencimiento: vencimiento(fechaDesembolso, frecuencia, k),
			Capital:          capital,
			Interes:          interes,
			Total:            total,
			SaldoPendiente:   total, // inicia en el total de la cuota
		})
	}
	return plan
}

func vencimiento(desembolso time.Time, f models.Frecuencia, k int) time.Time {
	switch f {
	case models.FrecuenciaDiaria:
		return desembolso.AddDate(0, 0, k)
	case models.FrecuenciaSemanal:
		return desembolso.AddDate(0, 0, k*7)
	case models.FrecuenciaQuincenal:
		return desembolso.AddDate(0, 0, k*15)
	case models.FrecuenciaMensual:
		return desembolso.AddDate(0, k, 0)
	}
	return desembolso.AddDate(0, k, 0)
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}
