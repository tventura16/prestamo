package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/prestamos/payment-service/internal/events"
	"github.com/prestamos/payment-service/internal/models"
	"github.com/prestamos/payment-service/internal/repository"
)

type PaymentService struct {
	pagoRepo *repository.PagoRepository
	loanRepo *repository.LoanRepository
	logger   *slog.Logger
}

func NewPaymentService(pago *repository.PagoRepository, loan *repository.LoanRepository, logger *slog.Logger) *PaymentService {
	return &PaymentService{pagoRepo: pago, loanRepo: loan, logger: logger}
}

// Distribucion es el resultado puro de repartir un monto recibido sobre una
// cuota, en el orden mora → interés → capital.
type Distribucion struct {
	Mora        float64
	Interes     float64
	Capital     float64
	NewSaldo    float64
	NewMora     float64
	Estado      string // "parcial" | "pagada"
	MarcarFecha bool
}

// distribuir calcula cómo se reparte `monto` sobre una cuota dado su snapshot
// y lo ya pagado previamente. Función PURA (sin I/O) para poder testear la
// lógica financiera de forma aislada. Devuelve ErrOverpayment si el monto
// excede lo adeudado (saldo + mora).
func distribuir(snap repository.CuotaSnapshot, interesYaPagado, capitalYaPagado, monto float64) (Distribucion, error) {
	totalAdeudado := round2(snap.SaldoPendiente + snap.MoraAcumulada)
	if monto > totalAdeudado+0.005 {
		return Distribucion{}, fmt.Errorf("%w: adeudado=%.2f, recibido=%.2f",
			repository.ErrOverpayment, totalAdeudado, monto)
	}

	restante := monto

	mora := round2(math.Min(restante, snap.MoraAcumulada))
	restante = round2(restante - mora)

	interesDebido := round2(snap.Interes - interesYaPagado)
	if interesDebido < 0 {
		interesDebido = 0
	}
	interes := round2(math.Min(restante, interesDebido))
	restante = round2(restante - interes)

	capitalDebido := round2(snap.Capital - capitalYaPagado)
	if capitalDebido < 0 {
		capitalDebido = 0
	}
	capital := round2(math.Min(restante, capitalDebido))
	restante = round2(restante - capital)

	// Un centavo residual por redondeo se atribuye a capital.
	if restante > 0 && capital+restante <= capitalDebido+0.01 {
		capital = round2(capital + restante)
	}

	newSaldo := round2(snap.SaldoPendiente - (interes + capital))
	if newSaldo < 0 {
		newSaldo = 0
	}
	newMora := round2(snap.MoraAcumulada - mora)
	if newMora < 0 {
		newMora = 0
	}

	d := Distribucion{
		Mora: mora, Interes: interes, Capital: capital,
		NewSaldo: newSaldo, NewMora: newMora, Estado: "parcial",
	}
	if newSaldo == 0 && newMora == 0 {
		d.Estado = "pagada"
		d.MarcarFecha = true
	}
	return d, nil
}

// Register registra un pago de forma correcta y recuperable.
//
// Garantías:
//   - Idempotencia: si se reenvía con la misma Idempotency-Key, devuelve la
//     respuesta original sin volver a cobrar (replayed=true).
//   - El dinero nunca se pierde: el pago + el evento de outbox se commitean
//     atomicamente en la DB pagos ANTES de commitear la cuota en la DB
//     prestamos. Si el commit de prestamos falla, el consumer del outbox
//     aplica la cuota de forma idempotente.
//
// Devuelve (resultado, replayed, error).
func (s *PaymentService) Register(ctx context.Context, in models.CreatePagoInput, idempotencyKey string) (models.PagoResult, bool, error) {
	// 1. Idempotencia: ¿ya procesamos esta clave?
	if idempotencyKey != "" {
		if resp, found, err := s.pagoRepo.FindByIdempotencyKey(ctx, idempotencyKey); err != nil {
			return models.PagoResult{}, false, err
		} else if found {
			var r models.PagoResult
			if err := json.Unmarshal(resp, &r); err != nil {
				return models.PagoResult{}, false, fmt.Errorf("decode idempotent response: %w", err)
			}
			return r, true, nil
		}
	}

	// 2. Pagos previos de la cuota (para distribuir parciales correctamente).
	interesYa, capitalYa, err := s.pagoRepo.PreviousPaidByCuota(ctx, in.CuotaID)
	if err != nil {
		return models.PagoResult{}, false, fmt.Errorf("previous paid: %w", err)
	}

	// 3. TX-prestamos: lock de la cuota + snapshot + distribución + validación.
	ltx, err := s.loanRepo.Pool().Begin(ctx)
	if err != nil {
		return models.PagoResult{}, false, fmt.Errorf("begin prestamos tx: %w", err)
	}
	defer ltx.Rollback(ctx)

	snap, err := s.loanRepo.LockCuotaWithPrestamo(ctx, ltx, in.CuotaID)
	if err != nil {
		return models.PagoResult{}, false, err
	}
	if snap.Estado == "pagada" {
		return models.PagoResult{}, false, repository.ErrCuotaPagada
	}

	dist, err := distribuir(snap, interesYa, capitalYa, in.MontoPagado)
	if err != nil {
		return models.PagoResult{}, false, err
	}

	// 4. Aplicar a la cuota (mismo lock). Idempotente vía pago_aplicaciones.
	pagoID := uuid.New()
	applyRes, err := s.loanRepo.ApplyPagoToCuota(ctx, ltx, repository.PagoAplicacion{
		PagoID:  pagoID,
		CuotaID: in.CuotaID,
		Capital: dist.Capital,
		Interes: dist.Interes,
		Mora:    dist.Mora,
	})
	if err != nil {
		return models.PagoResult{}, false, err
	}

	// 5. TX-pagos: pago + movimientos + outbox + idempotencia (atómico).
	tipo := models.TipoTotal
	if dist.Estado != "pagada" {
		tipo = models.TipoParcial
	}
	pago := models.Pago{
		ID:            pagoID,
		ClienteID:     snap.ClienteID,
		PrestamoID:    snap.PrestamoID,
		CuotaID:       &in.CuotaID,
		FechaPago:     time.Now(),
		MontoPagado:   round2(in.MontoPagado),
		CapitalPagado: dist.Capital,
		InteresPagado: dist.Interes,
		MoraPagada:    dist.Mora,
		Tipo:          tipo,
		MetodoPago:    in.MetodoPago,
		UsuarioID:     in.UsuarioID,
		Observaciones: in.Observaciones,
	}
	movimientos := []models.Movimiento{
		{Concepto: "mora", Monto: dist.Mora},
		{Concepto: "interes", Monto: dist.Interes},
		{Concepto: "capital", Monto: dist.Capital},
	}

	ptx, err := s.pagoRepo.Pool().Begin(ctx)
	if err != nil {
		return models.PagoResult{}, false, fmt.Errorf("begin pagos tx: %w", err)
	}
	defer ptx.Rollback(ctx)

	pagoIns, movsIns, err := s.pagoRepo.InsertPagoTx(ctx, ptx, pago, movimientos)
	if err != nil {
		return models.PagoResult{}, false, fmt.Errorf("insert pago: %w", err)
	}

	evt := events.PagoRegistrado{
		PagoID:     pagoIns.ID,
		CuotaID:    in.CuotaID,
		PrestamoID: snap.PrestamoID,
		ClienteID:  snap.ClienteID,
		Capital:    dist.Capital,
		Interes:    dist.Interes,
		Mora:       dist.Mora,
		OcurridoEn: pagoIns.FechaPago,
	}
	payload, err := evt.Marshal()
	if err != nil {
		return models.PagoResult{}, false, fmt.Errorf("marshal event: %w", err)
	}
	if err := repository.InsertOutboxTx(ctx, ptx, repository.OutboxEvent{
		AggregateType: events.AggregatePago,
		AggregateID:   pagoIns.ID,
		EventType:     events.TypePagoRegistrado,
		Payload:       payload,
	}); err != nil {
		return models.PagoResult{}, false, err
	}

	result := models.PagoResult{
		Pago:        pagoIns,
		Movimientos: movsIns,
		Cuota:       applyRes.Cuota,
		Prestamo:    applyRes.Prestamo,
	}

	if idempotencyKey != "" {
		respJSON, err := json.Marshal(result)
		if err != nil {
			return models.PagoResult{}, false, fmt.Errorf("encode response: %w", err)
		}
		if err := s.pagoRepo.SaveIdempotencyKeyTx(ctx, ptx, idempotencyKey, pagoIns.ID, respJSON); err != nil {
			return models.PagoResult{}, false, err
		}
	}

	// 6. Commit pagos PRIMERO: el dinero y el evento quedan durables.
	if err := ptx.Commit(ctx); err != nil {
		return models.PagoResult{}, false, fmt.Errorf("commit pagos tx: %w", err)
	}

	// 7. Commit prestamos. Si falla AQUÍ, el dinero ya está registrado y el
	//    evento de outbox sin publicar → el consumer aplicará la cuota. La
	//    respuesta refleja el estado al que converge.
	if err := ltx.Commit(ctx); err != nil {
		s.logger.Error("commit prestamos falló tras registrar pago; el outbox reconciliará la cuota",
			"pago_id", pagoIns.ID, "cuota_id", in.CuotaID, "err", err)
	}

	return result, false, nil
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}

// PagoNotFound es alias semántico para uso del handler.
var PagoNotFound = errors.New("pago no encontrado")
