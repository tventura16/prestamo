package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/prestamos/document-service/internal/models"
	"github.com/prestamos/document-service/internal/repository"
)

// DocumentService orquesta la generación de PDFs y su persistencia.
type DocumentService struct {
	docRepo      *repository.DocumentoRepository
	clienteRepo  *repository.ClienteRepository
	prestamoRepo *repository.PrestamoRepository
	pagoRepo     *repository.PagoRepository
	storePath    string
}

func NewDocumentService(
	doc *repository.DocumentoRepository,
	cli *repository.ClienteRepository,
	prestamo *repository.PrestamoRepository,
	pago *repository.PagoRepository,
	storePath string,
) *DocumentService {
	return &DocumentService{
		docRepo: doc, clienteRepo: cli, prestamoRepo: prestamo, pagoRepo: pago,
		storePath: storePath,
	}
}

// GenerarContrato genera el PDF del contrato y registra metadatos.
func (s *DocumentService) GenerarContrato(ctx context.Context, in models.ContratoInput) (models.Documento, error) {
	prestamo, err := s.prestamoRepo.GetByID(ctx, in.PrestamoID)
	if err != nil {
		return models.Documento{}, err
	}
	cliente, err := s.clienteRepo.GetByID(ctx, prestamo.ClienteID)
	if err != nil {
		return models.Documento{}, err
	}
	cuotas, err := s.prestamoRepo.ListCuotas(ctx, in.PrestamoID)
	if err != nil {
		return models.Documento{}, err
	}

	pdfBytes, err := GenerarContrato(cliente, prestamo, cuotas)
	if err != nil {
		return models.Documento{}, fmt.Errorf("generar pdf: %w", err)
	}

	name := fmt.Sprintf("contrato-%s.pdf", prestamo.ID.String()[:8])
	return s.persist(ctx, persistArgs{
		Tipo:        models.TipoContrato,
		ClienteID:   &cliente.ID,
		PrestamoID:  &prestamo.ID,
		GeneradoPor: in.GeneradoPor,
		Filename:    name,
		Content:     pdfBytes,
	})
}

// GenerarPlanPagos genera un PDF con solo la tabla de cuotas (reutiliza el
// contrato sin la sección legal).
func (s *DocumentService) GenerarPlanPagos(ctx context.Context, in models.PlanPagosInput) (models.Documento, error) {
	prestamo, err := s.prestamoRepo.GetByID(ctx, in.PrestamoID)
	if err != nil {
		return models.Documento{}, err
	}
	cliente, err := s.clienteRepo.GetByID(ctx, prestamo.ClienteID)
	if err != nil {
		return models.Documento{}, err
	}
	cuotas, err := s.prestamoRepo.ListCuotas(ctx, in.PrestamoID)
	if err != nil {
		return models.Documento{}, err
	}

	// Para plan de pagos usamos el mismo contrato (incluye tabla de cuotas).
	// Si quieres uno separado más simple, dividimos después.
	pdfBytes, err := GenerarContrato(cliente, prestamo, cuotas)
	if err != nil {
		return models.Documento{}, fmt.Errorf("generar pdf: %w", err)
	}

	name := fmt.Sprintf("plan-pagos-%s.pdf", prestamo.ID.String()[:8])
	return s.persist(ctx, persistArgs{
		Tipo:        models.TipoPlanPagos,
		ClienteID:   &cliente.ID,
		PrestamoID:  &prestamo.ID,
		GeneradoPor: in.GeneradoPor,
		Filename:    name,
		Content:     pdfBytes,
	})
}

// GenerarRecibo genera el PDF de un recibo de pago.
func (s *DocumentService) GenerarRecibo(ctx context.Context, in models.ReciboInput) (models.Documento, error) {
	pago, err := s.pagoRepo.GetByID(ctx, in.PagoID)
	if err != nil {
		return models.Documento{}, err
	}
	cliente, err := s.clienteRepo.GetByID(ctx, pago.ClienteID)
	if err != nil {
		return models.Documento{}, err
	}

	var cuotaNumero *int
	if pago.CuotaID != nil {
		if n, err := s.prestamoRepo.CuotaNumero(ctx, *pago.CuotaID); err == nil {
			cuotaNumero = &n
		}
	}

	pdfBytes, err := GenerarRecibo(cliente, pago, cuotaNumero)
	if err != nil {
		return models.Documento{}, fmt.Errorf("generar pdf: %w", err)
	}

	recibo := pago.ID.String()[:8]
	if pago.NumeroRecibo != nil {
		recibo = *pago.NumeroRecibo
	}
	name := fmt.Sprintf("recibo-%s.pdf", recibo)
	return s.persist(ctx, persistArgs{
		Tipo:        models.TipoRecibo,
		ClienteID:   &cliente.ID,
		PrestamoID:  &pago.PrestamoID,
		PagoID:      &pago.ID,
		GeneradoPor: in.GeneradoPor,
		Filename:    name,
		Content:     pdfBytes,
	})
}

// GenerarEstadoCuenta agrupa todos los préstamos y pagos del cliente.
func (s *DocumentService) GenerarEstadoCuenta(ctx context.Context, in models.EstadoCuentaInput) (models.Documento, error) {
	cliente, err := s.clienteRepo.GetByID(ctx, in.ClienteID)
	if err != nil {
		return models.Documento{}, err
	}
	prestamos, err := s.prestamoRepo.ListPrestamosPorCliente(ctx, in.ClienteID)
	if err != nil {
		return models.Documento{}, err
	}
	pagos, err := s.pagoRepo.ListByCliente(ctx, in.ClienteID)
	if err != nil {
		return models.Documento{}, err
	}

	pdfBytes, err := GenerarEstadoCuenta(cliente, prestamos, pagos)
	if err != nil {
		return models.Documento{}, fmt.Errorf("generar pdf: %w", err)
	}

	name := fmt.Sprintf("estado-cuenta-%s-%s.pdf",
		cliente.ID.String()[:8], time.Now().Format("20060102"))
	return s.persist(ctx, persistArgs{
		Tipo:        models.TipoEstadoCuenta,
		ClienteID:   &cliente.ID,
		GeneradoPor: in.GeneradoPor,
		Filename:    name,
		Content:     pdfBytes,
	})
}

// ResolveFilePath dado el documento, devuelve la ruta absoluta al PDF.
// Útil para el handler de descarga.
func (s *DocumentService) ResolveFilePath(d models.Documento) string {
	return d.Ruta
}

type persistArgs struct {
	Tipo        models.TipoDocumento
	ClienteID   *uuid.UUID
	PrestamoID  *uuid.UUID
	PagoID      *uuid.UUID
	GeneradoPor *uuid.UUID
	Filename    string
	Content     []byte
}

// persist escribe el PDF al disco y registra los metadatos en la DB.
func (s *DocumentService) persist(ctx context.Context, a persistArgs) (models.Documento, error) {
	// Subdirectorio por tipo para organizar.
	subdir := filepath.Join(s.storePath, string(a.Tipo))
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return models.Documento{}, fmt.Errorf("mkdir store: %w", err)
	}
	fullPath := filepath.Join(subdir, a.Filename)

	if err := os.WriteFile(fullPath, a.Content, 0o644); err != nil {
		return models.Documento{}, fmt.Errorf("write pdf: %w", err)
	}

	hash := sha256.Sum256(a.Content)
	hashHex := hex.EncodeToString(hash[:])
	tamanio := len(a.Content) / 1024

	doc := models.Documento{
		Tipo:          a.Tipo,
		ClienteID:     a.ClienteID,
		PrestamoID:    a.PrestamoID,
		PagoID:        a.PagoID,
		NombreArchivo: a.Filename,
		Ruta:          fullPath,
		HashSHA256:    &hashHex,
		TamanioKB:     &tamanio,
		Estado:        models.EstadoGenerado,
		GeneradoPor:   a.GeneradoPor,
	}

	saved, err := s.docRepo.Insert(ctx, doc)
	if err != nil {
		// Si falla el insert, borramos el archivo para no dejar huérfanos.
		_ = os.Remove(fullPath)
		return models.Documento{}, err
	}
	return saved, nil
}
