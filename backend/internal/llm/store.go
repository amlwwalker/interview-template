package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound means no LLM is on offer under that slug — either no row exists
// or the row is disabled. The two are deliberately indistinguishable to a
// caller: a retired model should not be resolvable by anyone who remembers its
// slug, and should not advertise that it once existed.
var ErrNotFound = errors.New("llm not found")

// LLM is one row of the catalogue: an LLM a user may route a conversation to.
//
// ProviderKey is internal. It binds this row to compiled code and is never
// serialised to a client — exposing it would invite selecting by it, which is
// what Slug is for.
type LLM struct {
	ID          int64     `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	ProviderKey string    `json:"-"`
	Model       string    `json:"model"`
	Enabled     bool      `json:"enabled"`
	SortOrder   int       `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Store is the catalogue contract. The handlers depend on this interface
// rather than on Postgres, which is what lets the handler tests run with no
// database.
type Store interface {
	List(ctx context.Context) ([]LLM, error)
	GetBySlug(ctx context.Context, slug string) (LLM, error)
}

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

const columns = `id, slug, name, provider_key, model, enabled, sort_order, created_at, updated_at`

func scanLLM(row pgx.Row) (LLM, error) {
	var l LLM
	err := row.Scan(&l.ID, &l.Slug, &l.Name, &l.ProviderKey, &l.Model,
		&l.Enabled, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt)
	return l, err
}

// List returns the LLMs currently on offer, in display order.
//
// Enabled rows only: this populates a picker, and a disabled row is one that
// has been retired. sort_order then id gives a stable sequence, so the list
// does not reshuffle between requests.
func (s *PostgresStore) List(ctx context.Context) ([]LLM, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+columns+` FROM llms WHERE enabled ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list llms: %w", err)
	}
	defer rows.Close()

	// Start non-nil so an empty catalogue serialises as [] rather than null.
	llms := make([]LLM, 0)

	for rows.Next() {
		l, err := scanLLM(rows)
		if err != nil {
			return nil, fmt.Errorf("scan llm: %w", err)
		}
		llms = append(llms, l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate llms: %w", err)
	}

	return llms, nil
}

// GetBySlug resolves the public selector to a row.
//
// Filtering on enabled here is what makes a retired LLM genuinely unreachable
// rather than merely hidden from the list.
func (s *PostgresStore) GetBySlug(ctx context.Context, slug string) (LLM, error) {
	l, err := scanLLM(s.pool.QueryRow(ctx,
		`SELECT `+columns+` FROM llms WHERE slug = $1 AND enabled`, slug))

	if errors.Is(err, pgx.ErrNoRows) {
		return LLM{}, ErrNotFound
	}
	if err != nil {
		return LLM{}, fmt.Errorf("get llm: %w", err)
	}

	return l, nil
}
