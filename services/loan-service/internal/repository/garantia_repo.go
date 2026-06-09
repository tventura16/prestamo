package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/loan-service/internal/models"
)

var ErrGarantiaNotFound = errors.New("garantía no encontrada")

type GarantiaRepository struct {
	pool *pgxpool.Pool
}

func NewGarantiaRepository(pool *pgxpool.Pool) *GarantiaRepository {
	return &GarantiaRepository{pool: pool}
}

const garantiaColumns = `id, prestamo_id, nombre_archivo, ruta, mime,
		tamanio_bytes, descripcion, subido_por, created_at`

func scanGarantia(row pgx.Row) (models.Garantia, error) {
	var g models.Garantia
	err := row.Scan(
		&g.ID, &g.PrestamoID, &g.NombreArchivo, &g.Ruta, &g.Mime,
		&g.TamanioBytes, &g.Descripcion, &g.SubidoPor, &g.CreatedAt,
	)
	return g, err
}

// PrestamoExists indica si el préstamo existe (validación previa a la subida).
func (r *GarantiaRepository) PrestamoExists(ctx context.Context, prestamoID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM prestamos WHERE id = $1)`, prestamoID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check prestamo: %w", err)
	}
	return exists, nil
}

func (r *GarantiaRepository) Insert(ctx context.Context, g models.Garantia) (models.Garantia, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO prestamo_garantias
		(prestamo_id, nombre_archivo, ruta, mime, tamanio_bytes, descripcion, subido_por)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+garantiaColumns,
		g.PrestamoID, g.NombreArchivo, g.Ruta, g.Mime, g.TamanioBytes, g.Descripcion, g.SubidoPor,
	)
	out, err := scanGarantia(row)
	if err != nil {
		return models.Garantia{}, fmt.Errorf("insert garantia: %w", err)
	}
	return out, nil
}

func (r *GarantiaRepository) ListByPrestamo(ctx context.Context, prestamoID uuid.UUID) ([]models.Garantia, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+garantiaColumns+` FROM prestamo_garantias WHERE prestamo_id = $1 ORDER BY created_at`,
		prestamoID)
	if err != nil {
		return nil, fmt.Errorf("list garantias: %w", err)
	}
	defer rows.Close()

	out := make([]models.Garantia, 0)
	for rows.Next() {
		g, err := scanGarantia(rows)
		if err != nil {
			return nil, fmt.Errorf("scan garantia: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// Get devuelve la garantía (incluida la ruta interna) para servir el archivo.
func (r *GarantiaRepository) Get(ctx context.Context, id uuid.UUID) (models.Garantia, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+garantiaColumns+` FROM prestamo_garantias WHERE id = $1`, id)
	g, err := scanGarantia(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Garantia{}, ErrGarantiaNotFound
		}
		return models.Garantia{}, fmt.Errorf("get garantia: %w", err)
	}
	return g, nil
}

// Delete borra el registro y devuelve la fila eliminada (para borrar el archivo).
func (r *GarantiaRepository) Delete(ctx context.Context, id uuid.UUID) (models.Garantia, error) {
	row := r.pool.QueryRow(ctx,
		`DELETE FROM prestamo_garantias WHERE id = $1 RETURNING `+garantiaColumns, id)
	g, err := scanGarantia(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Garantia{}, ErrGarantiaNotFound
		}
		return models.Garantia{}, fmt.Errorf("delete garantia: %w", err)
	}
	return g, nil
}
