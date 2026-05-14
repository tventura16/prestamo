package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/prestamos/payment-service/internal/models"
	"github.com/prestamos/payment-service/internal/repository"
)

type PaymentService struct {
	pagoRepo *repository.PagoRepository
	loanRepo *repository.LoanRepository
}

func NewPaymentService(pago *repository.PagoRepository, loan *repository.LoanRepository) *PaymentService {
	return &PaymentService{pagoRepo: pago, loanRepo: loan}
}

// Register aplica un pago a una cuota.
//
// Política de distribución del monto recibido:
//  1. Cubre mora_acumulada primero.
//  2. Después cubre el interés pendiente de la cuota (hasta lo no pagado en
//     pagos previos).
//  3. El remanente va a capital.
//
// Si la suma deja saldo_pendiente == 0 y mora_acumulada == 0, la cuota pasa
// a estado "pagada". Si todas las cuotas del préstamo quedan en cero, el
// préstamo pasa a "finalizado".
//
// LIMITACIÓN: la transacción cubre el lock+actualización en DB prestamos.
// La inserción del pago en DB pagos es una transacción aparte. Si esta
// segunda falla tras commit de prestamos, queda inconsistencia operativa
// (cuota actualizada sin recibo). En v2 esto se resuelve con outbox/eventos.
func (s *PaymentService) Register(ctx context.Context, in models.CreatePagoInput) (models.PagoResult, error) {
	// 1. Pagos previos para esta cuota (para distribuir correctamente).
	interesYaPagado, capitalYaPagado, err := s.pagoRepo.PreviousPaidByCuota(ctx, in.CuotaID)
	if err != nil {
		return models.PagoResult{}, fmt.Errorf("previous paid: %w", err)
	}

	// 2. Transacción en DB prestamos: lock + cálculo + update cuota/préstamo.
	tx, err := s.loanRepo.Pool().Begin(ctx)
	if err != nil {
		return models.PagoResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	snap, err := s.loanRepo.LockCuotaWithPrestamo(ctx, tx, in.CuotaID)
	if err != nil {
		return models.PagoResult{}, err
	}
	if snap.Estado == "pagada" {
		return models.PagoResult{}, repository.ErrCuotaPagada
	}

	totalAdeudado := round2(snap.SaldoPendiente + snap.MoraAcumulada)
	if in.MontoPagado > totalAdeudado+0.005 {
		return models.PagoResult{}, fmt.Errorf("%w: adeudado=%.2f, recibido=%.2f",
			repository.ErrOverpayment, totalAdeudado, in.MontoPagado)
	}

	// 3. Distribución: mora → interés → capital.
	restante := in.MontoPagado

	moraPagada := math.Min(restante, snap.MoraAcumulada)
	moraPagada = round2(moraPagada)
	restante = round2(restante - moraPagada)

	interesDebido := round2(snap.Interes - interesYaPagado)
	if interesDebido < 0 {
		interesDebido = 0
	}
	interesPagado := round2(math.Min(restante, interesDebido))
	restante = round2(restante - interesPagado)

	capitalDebido := round2(snap.Capital - capitalYaPagado)
	if capitalDebido < 0 {
		capitalDebido = 0
	}
	capitalPagado := round2(math.Min(restante, capitalDebido))
	restante = round2(restante - capitalPagado)

	// Si por redondeo queda un centavo, atribuirlo a capital.
	if restante > 0 && capitalPagado+restante <= capitalDebido+0.01 {
		capitalPagado = round2(capitalPagado + restante)
	}

	// 4. Nuevo estado de la cuota.
	newSaldo := round2(snap.SaldoPendiente - (interesPagado + capitalPagado))
	if newSaldo < 0 {
		newSaldo = 0
	}
	newMora := round2(snap.MoraAcumulada - moraPagada)
	if newMora < 0 {
		newMora = 0
	}

	estado := "parcial"
	marcarFecha := false
	if newSaldo == 0 && newMora == 0 {
		estado = "pagada"
		marcarFecha = true
	}

	cuotaInfo, err := s.loanRepo.UpdateCuotaAfterPayment(ctx, tx, in.CuotaID, newSaldo, newMora, estado, marcarFecha)
	if err != nil {
		return models.PagoResult{}, err
	}

	// 5. Si todas las cuotas del préstamo quedaron en cero → finalizado.
	prestamoInfo := models.PrestamoInfo{ID: snap.PrestamoID, Estado: snap.EstadoPrestamo}
	if estado == "pagada" {
		allPaid, err := s.loanRepo.AllCuotasPagadas(ctx, tx, snap.PrestamoID)
		if err != nil {
			return models.PagoResult{}, err
		}
		if allPaid {
			prestamoInfo, err = s.loanRepo.MarkPrestamoFinalizado(ctx, tx, snap.PrestamoID)
			if err != nil {
				return models.PagoResult{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PagoResult{}, fmt.Errorf("commit prestamos tx: %w", err)
	}

	// 6. Registrar el pago en DB pagos (transacción aparte). Si falla acá,
	//    se debe reconciliar manualmente (ver LIMITACIÓN arriba).
	tipo := models.TipoTotal
	if estado != "pagada" {
		tipo = models.TipoParcial
	}

	pago := models.Pago{
		ClienteID:     snap.ClienteID,
		PrestamoID:    snap.PrestamoID,
		CuotaID:       &in.CuotaID,
		FechaPago:     time.Now(),
		MontoPagado:   round2(in.MontoPagado),
		CapitalPagado: capitalPagado,
		InteresPagado: interesPagado,
		MoraPagada:    moraPagada,
		Tipo:          tipo,
		MetodoPago:    in.MetodoPago,
		UsuarioID:     in.UsuarioID,
		Observaciones: in.Observaciones,
	}

	movimientos := []models.Movimiento{
		{Concepto: "mora", Monto: moraPagada},
		{Concepto: "interes", Monto: interesPagado},
		{Concepto: "capital", Monto: capitalPagado},
	}

	pagoInsertado, movsInsertados, err := s.pagoRepo.InsertPago(ctx, pago, movimientos)
	if err != nil {
		return models.PagoResult{}, fmt.Errorf("insert pago (cuota %s ya actualizada): %w", in.CuotaID, err)
	}

	return models.PagoResult{
		Pago:        pagoInsertado,
		Movimientos: movsInsertados,
		Cuota:       cuotaInfo,
		Prestamo:    prestamoInfo,
	}, nil
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}

// PagoNotFound es alias semántico para uso del handler.
var PagoNotFound = errors.New("pago no encontrado")
