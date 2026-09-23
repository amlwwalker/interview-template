//go:build integration

// Integration tests for PostgresStore, run against a real database.
//
//	make test-integration
//
// The handler tests use an in-memory fake, which proves the HTTP layer but
// says nothing about whether the SQL is right. These run the actual queries,
// so a COALESCE that silently stops coalescing is caught here rather than in
// production.
package record

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) (*PostgresStore, context.Context) {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		// Skipping locally is a convenience — you may not have Postgres running.
		// Skipping on CI is a silent lie: the run goes green having tested
		// nothing, which is worse than failing.
		if os.Getenv("CI") != "" {
			t.Fatal("DATABASE_URL is empty on CI. The integration tests would " +
				"silently skip and the run would pass having tested nothing. " +
				"Check the workflow sets TEST_DB_URL for `make test-integration`.")
		}
		t.Skip("DATABASE_URL not set; run `make test-integration`")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	// Each test starts from a known-empty table. RESTART IDENTITY keeps ids
	// predictable so failures are readable.
	if _, err := pool.Exec(ctx, `TRUNCATE records RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return NewPostgresStore(pool), ctx
}

func seedRow(t *testing.T, s *PostgresStore, ctx context.Context) Record {
	t.Helper()

	rec, err := s.Create(ctx, CreateParams{
		Name:        "seeded",
		Description: "seeded description",
		Active:      true,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return rec
}

// Create writes every column and returns the generated id and timestamps.
func TestIntegrationCreate(t *testing.T) {
	s, ctx := newTestStore(t)

	got, err := s.Create(ctx, CreateParams{Name: "alpha", Description: "d", Active: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if got.ID == 0 {
		t.Error("expected a generated id")
	}
	if got.Name != "alpha" || got.Description != "d" || !got.Active {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("expected timestamps to be populated by the database")
	}
}

// The database CHECK constraint rejects a blank name even if Go validation is bypassed.
func TestIntegrationCreateRejectsBlankNameAtTheDatabase(t *testing.T) {
	s, ctx := newTestStore(t)

	if _, err := s.Create(ctx, CreateParams{Name: "   "}); err == nil {
		t.Fatal("expected the CHECK constraint to reject a whitespace-only name")
	}
}

// List returns an empty slice, never nil, so the JSON is [] rather than null.
func TestIntegrationListIsEmptySliceNotNil(t *testing.T) {
	s, ctx := newTestStore(t)

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if got == nil {
		t.Fatal("List returned nil; it must return an empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

// List returns rows in ascending id order, as the SQL promises.
func TestIntegrationListIsOrderedById(t *testing.T) {
	s, ctx := newTestStore(t)

	for _, name := range []string{"first", "second", "third"} {
		if _, err := s.Create(ctx, CreateParams{Name: name}); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	want := []string{"first", "second", "third"}
	for i, w := range want {
		if got[i].Name != w {
			t.Errorf("position %d = %q, want %q", i, got[i].Name, w)
		}
	}
}

// A missing row surfaces as ErrNotFound, not a scan error.
func TestIntegrationGetMissingReturnsErrNotFound(t *testing.T) {
	s, ctx := newTestStore(t)

	if _, err := s.Get(ctx, 999); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// Replace overwrites every column: this is the PUT semantic, in SQL.
func TestIntegrationReplaceResetsOmittedColumns(t *testing.T) {
	s, ctx := newTestStore(t)
	seeded := seedRow(t, s, ctx)

	// Only Name carries a value; the zero values must land in the row.
	got, err := s.Replace(ctx, seeded.ID, ReplaceParams{Name: "replaced"})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}

	if got.Name != "replaced" {
		t.Errorf("name = %q, want %q", got.Name, "replaced")
	}
	if got.Description != "" {
		t.Errorf("description = %q, want \"\" — PUT must reset it", got.Description)
	}
	if got.Active {
		t.Error("active = true, want false — PUT must reset it")
	}

	// And it is actually on disk, not just in the RETURNING row.
	reread, err := s.Get(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if reread.Description != "" || reread.Active {
		t.Errorf("persisted row disagrees with the returned row: %+v", reread)
	}
}

// Update's COALESCE leaves untouched columns alone: the PATCH semantic, in SQL.
func TestIntegrationUpdateLeavesOmittedColumnsAlone(t *testing.T) {
	s, ctx := newTestStore(t)
	seeded := seedRow(t, s, ctx)

	name := "patched"
	got, err := s.Update(ctx, seeded.ID, UpdateParams{Name: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if got.Name != "patched" {
		t.Errorf("name = %q, want %q", got.Name, "patched")
	}
	if got.Description != "seeded description" {
		t.Errorf("description = %q, want it untouched", got.Description)
	}
	if !got.Active {
		t.Error("active = false, want it untouched (true)")
	}
}

// A false pointer must reach the column; only a nil pointer means "leave alone".
func TestIntegrationUpdateCanWriteFalse(t *testing.T) {
	s, ctx := newTestStore(t)
	seeded := seedRow(t, s, ctx) // active = true

	active := false
	got, err := s.Update(ctx, seeded.ID, UpdateParams{Active: &active})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if got.Active {
		t.Error("active = true, want false")
	}
	if got.Name != "seeded" {
		t.Errorf("name = %q, want it untouched", got.Name)
	}
}

// updated_at moves on write; created_at does not.
func TestIntegrationUpdateTouchesUpdatedAtOnly(t *testing.T) {
	s, ctx := newTestStore(t)
	seeded := seedRow(t, s, ctx)

	time.Sleep(5 * time.Millisecond)

	name := "touched"
	got, err := s.Update(ctx, seeded.ID, UpdateParams{Name: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	if !got.UpdatedAt.After(seeded.UpdatedAt) {
		t.Errorf("updated_at did not advance: %v then %v", seeded.UpdatedAt, got.UpdatedAt)
	}
	if !got.CreatedAt.Equal(seeded.CreatedAt) {
		t.Errorf("created_at changed: %v then %v", seeded.CreatedAt, got.CreatedAt)
	}
}

// Replace and Update both map a missing row to ErrNotFound.
func TestIntegrationReplaceAndUpdateMissingReturnErrNotFound(t *testing.T) {
	s, ctx := newTestStore(t)

	if _, err := s.Replace(ctx, 999, ReplaceParams{Name: "x"}); err != ErrNotFound {
		t.Errorf("Replace err = %v, want ErrNotFound", err)
	}

	name := "x"
	if _, err := s.Update(ctx, 999, UpdateParams{Name: &name}); err != ErrNotFound {
		t.Errorf("Update err = %v, want ErrNotFound", err)
	}
}

// Delete actually removes the row from Postgres.
func TestIntegrationDeleteRemovesTheRow(t *testing.T) {
	s, ctx := newTestStore(t)
	seeded := seedRow(t, s, ctx)

	if err := s.Delete(ctx, seeded.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := s.Get(ctx, seeded.ID); err != ErrNotFound {
		t.Errorf("after delete, Get err = %v, want ErrNotFound", err)
	}
}

// Deleting a missing row reports ErrNotFound via RowsAffected.
func TestIntegrationDeleteMissingReturnsErrNotFound(t *testing.T) {
	s, ctx := newTestStore(t)

	if err := s.Delete(ctx, 999); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
