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

var (
	ErrGarantiaNotFound = errors.New("garantía no encontrada")
	ErrImagenNotFound   = errors.New("imagen no encontrada")
)

type GarantiaRepository struct {
	pool *pgxpool.Pool
}

func NewGarantiaRepository(pool *pgxpool.Pool) *GarantiaRepository {
	return &GarantiaRepository{pool: pool}
}

// ─── Garantías ───

const garantiaColumns = `id, prestamo_id, subtipo, descripcion, valor_estimado,
		moneda, cliente_garante_id, datos, created_at, updated_at`

func scanGarantia(row pgx.Row) (models.Garantia, error) {
	var g models.Garantia
	err := row.Scan(
		&g.ID, &g.PrestamoID, &g.Subtipo, &g.Descripcion, &g.ValorEstimado,
		&g.Moneda, &g.ClienteGaranteID, &g.Datos, &g.CreatedAt, &g.UpdatedAt,
	)
	return g, err
}

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
	row := r.pool.QueryRow(ctx, `INSERT INTO garantias
		(prestamo_id, subtipo, descripcion, valor_estimado, moneda, cliente_garante_id, datos)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+garantiaColumns,
		g.PrestamoID, g.Subtipo, g.Descripcion, g.ValorEstimado, g.Moneda, g.ClienteGaranteID, g.Datos,
	)
	out, err := scanGarantia(row)
	if err != nil {
		return models.Garantia{}, fmt.Errorf("insert garantia: %w", err)
	}
	return out, nil
}

func (r *GarantiaRepository) ListByPrestamo(ctx context.Context, prestamoID uuid.UUID) ([]models.Garantia, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+garantiaColumns+` FROM garantias WHERE prestamo_id = $1 ORDER BY created_at`,
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

func (r *GarantiaRepository) Get(ctx context.Context, id uuid.UUID) (models.Garantia, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+garantiaColumns+` FROM garantias WHERE id = $1`, id)
	g, err := scanGarantia(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Garantia{}, ErrGarantiaNotFound
		}
		return models.Garantia{}, fmt.Errorf("get garantia: %w", err)
	}
	return g, nil
}

func (r *GarantiaRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM garantias WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete garantia: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGarantiaNotFound
	}
	return nil
}

// ─── Imágenes de garantía ───

const imagenColumns = `id, garantia_id, nombre_archivo, ruta, mime,
		tamanio_bytes, descripcion, subido_por, created_at`

func scanImagen(row pgx.Row) (models.GarantiaImagen, error) {
	var m models.GarantiaImagen
	err := row.Scan(
		&m.ID, &m.GarantiaID, &m.NombreArchivo, &m.Ruta, &m.Mime,
		&m.TamanioBytes, &m.Descripcion, &m.SubidoPor, &m.CreatedAt,
	)
	return m, err
}

func (r *GarantiaRepository) InsertImagen(ctx context.Context, m models.GarantiaImagen) (models.GarantiaImagen, error) {
	row := r.pool.QueryRow(ctx, `INSERT INTO garantia_imagenes
		(garantia_id, nombre_archivo, ruta, mime, tamanio_bytes, descripcion, subido_por)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+imagenColumns,
		m.GarantiaID, m.NombreArchivo, m.Ruta, m.Mime, m.TamanioBytes, m.Descripcion, m.SubidoPor,
	)
	out, err := scanImagen(row)
	if err != nil {
		return models.GarantiaImagen{}, fmt.Errorf("insert imagen: %w", err)
	}
	return out, nil
}

func (r *GarantiaRepository) ListImagenesByGarantia(ctx context.Context, garantiaID uuid.UUID) ([]models.GarantiaImagen, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+imagenColumns+` FROM garantia_imagenes WHERE garantia_id = $1 ORDER BY created_at`,
		garantiaID)
	if err != nil {
		return nil, fmt.Errorf("list imagenes: %w", err)
	}
	defer rows.Close()

	out := make([]models.GarantiaImagen, 0)
	for rows.Next() {
		m, err := scanImagen(rows)
		if err != nil {
			return nil, fmt.Errorf("scan imagen: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *GarantiaRepository) GetImagen(ctx context.Context, id uuid.UUID) (models.GarantiaImagen, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+imagenColumns+` FROM garantia_imagenes WHERE id = $1`, id)
	m, err := scanImagen(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GarantiaImagen{}, ErrImagenNotFound
		}
		return models.GarantiaImagen{}, fmt.Errorf("get imagen: %w", err)
	}
	return m, nil
}

func (r *GarantiaRepository) DeleteImagen(ctx context.Context, id uuid.UUID) (models.GarantiaImagen, error) {
	row := r.pool.QueryRow(ctx,
		`DELETE FROM garantia_imagenes WHERE id = $1 RETURNING `+imagenColumns, id)
	m, err := scanImagen(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GarantiaImagen{}, ErrImagenNotFound
		}
		return models.GarantiaImagen{}, fmt.Errorf("delete imagen: %w", err)
	}
	return m, nil
}
