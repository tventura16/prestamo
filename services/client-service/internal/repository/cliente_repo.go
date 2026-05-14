package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prestamos/client-service/internal/models"
)

var (
	ErrNotFound    = errors.New("cliente no encontrado")
	ErrDuplicateCI = errors.New("ya existe un cliente con ese CI")
)

type ClienteRepository struct {
	pool *pgxpool.Pool
}

func NewClienteRepository(pool *pgxpool.Pool) *ClienteRepository {
	return &ClienteRepository{pool: pool}
}

const selectColumns = `id, nombres, apellidos, ci, fecha_nacimiento, telefono,
		direccion, email, estado, foto_url, created_at, updated_at`

func scan(row pgx.Row) (models.Cliente, error) {
	var c models.Cliente
	err := row.Scan(
		&c.ID, &c.Nombres, &c.Apellidos, &c.CI, &c.FechaNacimiento,
		&c.Telefono, &c.Direccion, &c.Email, &c.Estado, &c.FotoURL,
		&c.CreatedAt, &c.UpdatedAt,
	)
	return c, err
}

func (r *ClienteRepository) Create(ctx context.Context, in models.CreateClienteInput) (models.Cliente, error) {
	fecha, err := time.Parse("2006-01-02", in.FechaNacimiento)
	if err != nil {
		return models.Cliente{}, fmt.Errorf("parse fecha_nacimiento: %w", err)
	}

	estado := models.EstadoActivo
	if in.Estado != nil {
		estado = *in.Estado
	}

	query := `INSERT INTO clientes
		(nombres, apellidos, ci, fecha_nacimiento, telefono, direccion, email, estado, foto_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING ` + selectColumns

	row := r.pool.QueryRow(ctx, query,
		in.Nombres, in.Apellidos, in.CI, fecha,
		in.Telefono, in.Direccion, in.Email, estado, in.FotoURL,
	)

	c, err := scan(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Cliente{}, ErrDuplicateCI
		}
		return models.Cliente{}, fmt.Errorf("insert cliente: %w", err)
	}
	return c, nil
}

func (r *ClienteRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Cliente, error) {
	query := `SELECT ` + selectColumns + ` FROM clientes WHERE id = $1`
	c, err := scan(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Cliente{}, ErrNotFound
		}
		return models.Cliente{}, fmt.Errorf("get cliente: %w", err)
	}
	return c, nil
}

func (r *ClienteRepository) List(ctx context.Context, page, limit int, search string) (models.ListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	args := []any{}
	where := ""
	if s := strings.TrimSpace(search); s != "" {
		where = ` WHERE (nombres ILIKE $1 OR apellidos ILIKE $1 OR ci ILIKE $1)`
		args = append(args, "%"+s+"%")
	}

	var total int64
	countArgs := append([]any{}, args...)
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM clientes`+where, countArgs...).Scan(&total); err != nil {
		return models.ListResult{}, fmt.Errorf("count clientes: %w", err)
	}

	listQuery := `SELECT ` + selectColumns + ` FROM clientes` + where +
		fmt.Sprintf(` ORDER BY apellidos, nombres LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return models.ListResult{}, fmt.Errorf("list clientes: %w", err)
	}
	defer rows.Close()

	items := make([]models.Cliente, 0, limit)
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return models.ListResult{}, fmt.Errorf("scan cliente: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return models.ListResult{}, fmt.Errorf("rows: %w", err)
	}

	return models.ListResult{Items: items, Total: total, Page: page, Limit: limit}, nil
}

func (r *ClienteRepository) Update(ctx context.Context, id uuid.UUID, in models.UpdateClienteInput) (models.Cliente, error) {
	sets := []string{}
	args := []any{}
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}

	if in.Nombres != nil {
		add("nombres", *in.Nombres)
	}
	if in.Apellidos != nil {
		add("apellidos", *in.Apellidos)
	}
	if in.CI != nil {
		add("ci", *in.CI)
	}
	if in.FechaNacimiento != nil {
		fecha, err := time.Parse("2006-01-02", *in.FechaNacimiento)
		if err != nil {
			return models.Cliente{}, fmt.Errorf("parse fecha_nacimiento: %w", err)
		}
		add("fecha_nacimiento", fecha)
	}
	if in.Telefono != nil {
		add("telefono", *in.Telefono)
	}
	if in.Direccion != nil {
		add("direccion", *in.Direccion)
	}
	if in.Email != nil {
		add("email", *in.Email)
	}
	if in.Estado != nil {
		add("estado", *in.Estado)
	}
	if in.FotoURL != nil {
		add("foto_url", *in.FotoURL)
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	args = append(args, id)
	query := fmt.Sprintf(
		`UPDATE clientes SET %s WHERE id = $%d RETURNING %s`,
		strings.Join(sets, ", "), len(args), selectColumns,
	)

	c, err := scan(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Cliente{}, ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Cliente{}, ErrDuplicateCI
		}
		return models.Cliente{}, fmt.Errorf("update cliente: %w", err)
	}
	return c, nil
}

func (r *ClienteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM clientes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete cliente: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
