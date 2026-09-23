package record

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is the Store implementation backed by Postgres.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore wires a store to a connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Compile-time assertion that we satisfy the interface the handlers want.
var _ Store = (*PostgresStore)(nil)

const columns = `id, name, description, active, created_at, updated_at`

func (s *PostgresStore) List(ctx context.Context) ([]Record, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+columns+` FROM records ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()

	// Start non-nil so an empty table serialises as [] rather than null.
	records := make([]Record, 0)

	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Active, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate records: %w", err)
	}

	return records, nil
}

func (s *PostgresStore) Get(ctx context.Context, id int64) (Record, error) {
	var r Record

	err := s.pool.QueryRow(ctx,
		`SELECT `+columns+` FROM records WHERE id = $1`, id,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Active, &r.CreatedAt, &r.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("get record: %w", err)
	}

	return r, nil
}

func (s *PostgresStore) Create(ctx context.Context, p CreateParams) (Record, error) {
	var r Record

	err := s.pool.QueryRow(ctx,
		`INSERT INTO records (name, description, active) VALUES ($1, $2, $3) RETURNING `+columns,
		p.Name, p.Description, p.Active,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Active, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return Record{}, fmt.Errorf("create record: %w", err)
	}

	return r, nil
}

// Replace overwrites every mutable column — the PUT semantic. No COALESCE here:
// each parameter is written unconditionally, so anything the client left out of
// the body lands in the row as its zero value.
func (s *PostgresStore) Replace(ctx context.Context, id int64, p ReplaceParams) (Record, error) {
	var r Record

	err := s.pool.QueryRow(ctx, `
		UPDATE records
		SET name        = $2,
		    description = $3,
		    active      = $4,
		    updated_at  = NOW()
		WHERE id = $1
		RETURNING `+columns,
		id, p.Name, p.Description, p.Active,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Active, &r.CreatedAt, &r.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("replace record: %w", err)
	}

	return r, nil
}

// Update applies a partial update in a single statement — the PATCH semantic.
// COALESCE lets a NULL parameter mean "leave this column alone", so we avoid
// both a read-modify-write race and the usual pile of dynamically built SQL.
func (s *PostgresStore) Update(ctx context.Context, id int64, p UpdateParams) (Record, error) {
	var r Record

	err := s.pool.QueryRow(ctx, `
		UPDATE records
		SET name        = COALESCE($2, name),
		    description = COALESCE($3, description),
		    active      = COALESCE($4, active),
		    updated_at  = NOW()
		WHERE id = $1
		RETURNING `+columns,
		id, p.Name, p.Description, p.Active,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Active, &r.CreatedAt, &r.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("update record: %w", err)
	}

	return r, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM records WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
