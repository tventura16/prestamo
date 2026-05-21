package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/document-service/internal/models"
)

var ErrDocumentoNotFound = errors.New("documento no encontrado")

type DocumentoRepository struct {
	pool *pgxpool.Pool
}

func NewDocumentoRepository(pool *pgxpool.Pool) *DocumentoRepository {
	return &DocumentoRepository{pool: pool}
}

const docColumns = `id, tipo::text, cliente_id, prestamo_id, pago_id,
		nombre_archivo, ruta, hash_sha256, tamanio_kb, estado::text, error_mensaje,
		generado_por, generado_at`

func scanDoc(row pgx.Row) (models.Documento, error) {
	var d models.Documento
	err := row.Scan(
		&d.ID, &d.Tipo, &d.ClienteID, &d.PrestamoID, &d.PagoID,
		&d.NombreArchivo, &d.Ruta, &d.HashSHA256, &d.TamanioKB, &d.Estado,
		&d.ErrorMensaje, &d.GeneradoPor, &d.GeneradoAt,
	)
	return d, err
}

func (r *DocumentoRepository) Insert(ctx context.Context, d models.Documento) (models.Documento, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO documentos_generados
		(tipo, cliente_id, prestamo_id, pago_id, nombre_archivo, ruta, hash_sha256,
		 tamanio_kb, estado, generado_por)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING `+docColumns,
		d.Tipo, d.ClienteID, d.PrestamoID, d.PagoID, d.NombreArchivo, d.Ruta,
		d.HashSHA256, d.TamanioKB, d.Estado, d.GeneradoPor,
	)
	out, err := scanDoc(row)
	if err != nil {
		return models.Documento{}, fmt.Errorf("insert documento: %w", err)
	}
	return out, nil
}

func (r *DocumentoRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Documento, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+docColumns+` FROM documentos_generados WHERE id = $1`, id)
	d, err := scanDoc(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Documento{}, ErrDocumentoNotFound
		}
		return models.Documento{}, fmt.Errorf("get documento: %w", err)
	}
	return d, nil
}

func (r *DocumentoRepository) List(ctx context.Context, page, limit int, tipo string) (models.ListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	args := []any{}
	where := ""
	if tipo != "" {
		args = append(args, tipo)
		where = fmt.Sprintf(" WHERE tipo = $%d", len(args))
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM documentos_generados`+where, args...).Scan(&total); err != nil {
		return models.ListResult{}, fmt.Errorf("count: %w", err)
	}

	q := `SELECT ` + docColumns + ` FROM documentos_generados` + where +
		fmt.Sprintf(` ORDER BY generado_at DESC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return models.ListResult{}, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	items := make([]models.Documento, 0, limit)
	for rows.Next() {
		d, err := scanDoc(rows)
		if err != nil {
			return models.ListResult{}, fmt.Errorf("scan: %w", err)
		}
		items = append(items, d)
	}
	return models.ListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}
