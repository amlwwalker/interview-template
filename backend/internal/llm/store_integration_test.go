//go:build integration

// Integration tests for the llms catalogue, run against a real database.
//
//	make test-integration
//
// The handler tests use an in-memory fake, which proves the HTTP layer but
// says nothing about whether the SQL or the constraints are right. These run
// the real queries against the real schema.
package llm

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newTestStore(t *testing.T) (*PostgresStore, *pgxpool.Pool, context.Context) {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		// Skipping locally is a convenience. Skipping on CI is a silent lie:
		// the run goes green having tested nothing.
		if os.Getenv("CI") != "" {
			t.Fatal("DATABASE_URL is empty on CI. The integration tests would " +
				"silently skip and the run would pass having tested nothing.")
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

	return NewPostgresStore(pool), pool, ctx
}

// truncate empties the table so a test starts from a known state. Tests that
// want the seeded rows call seedMocks instead.
func truncate(t *testing.T, pool *pgxpool.Pool, ctx context.Context) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE llms RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func insert(t *testing.T, pool *pgxpool.Pool, ctx context.Context,
	slug, name, providerKey, model string, enabled bool, order int) {
	t.Helper()

	_, err := pool.Exec(ctx,
		`INSERT INTO llms (slug, name, provider_key, model, enabled, sort_order)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		slug, name, providerKey, model, enabled, order)
	if err != nil {
		t.Fatalf("insert %s: %v", slug, err)
	}
}

// List is what populates a model picker, so a disabled row must not appear.
func TestIntegrationListReturnsOnlyEnabledRows(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)

	insert(t, pool, ctx, "on", "On", "echo", "m", true, 10)
	insert(t, pool, ctx, "off", "Off", "echo", "m", false, 20)

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (the disabled row must be excluded): %+v", len(got), got)
	}
	if got[0].Slug != "on" {
		t.Errorf("Slug = %q, want %q", got[0].Slug, "on")
	}
}

// Deterministic order: sort_order first, then id as the tiebreak. Without this
// the picker reshuffles between requests.
func TestIntegrationListOrdersBySortOrderThenID(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)

	insert(t, pool, ctx, "third", "Third", "echo", "m", true, 30)
	insert(t, pool, ctx, "first", "First", "echo", "m", true, 10)
	insert(t, pool, ctx, "tie-b", "Tie B", "echo", "m", true, 20)
	insert(t, pool, ctx, "tie-a", "Tie A", "echo", "m", true, 20)

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	want := []string{"first", "tie-b", "tie-a", "third"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Slug != want[i] {
			t.Fatalf("order = %v, want %v", slugsOf(got), want)
		}
	}
}

func slugsOf(items []LLM) []string {
	out := make([]string, 0, len(items))
	for _, l := range items {
		out = append(out, l.Slug)
	}
	return out
}

// Empty slice, never nil, so the JSON is [] rather than null.
func TestIntegrationListIsEmptySliceNotNil(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)

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

// GetBySlug round-trips every column a caller needs to run the model.
func TestIntegrationGetBySlugReturnsTheRow(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)
	insert(t, pool, ctx, "echo-mock", "Echo (mock)", "echo", "echo-v1", true, 10)

	got, err := s.GetBySlug(ctx, "echo-mock")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Name != "Echo (mock)" || got.ProviderKey != "echo" || got.Model != "echo-v1" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

// An unknown slug is a client error, and must be reported as ErrNotFound
// rather than an empty row a caller might mistake for a real one.
func TestIntegrationGetBySlugUnknownReturnsNotFound(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)

	if _, err := s.GetBySlug(ctx, "no-such-slug"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// A disabled LLM is not on offer. Resolving it must fail the same way an
// unknown one does, or a retired model stays quietly reachable by anyone who
// remembers its slug.
func TestIntegrationGetBySlugDisabledReturnsNotFound(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)
	insert(t, pool, ctx, "retired", "Retired", "echo", "m", false, 10)

	if _, err := s.GetBySlug(ctx, "retired"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound for a disabled row", err)
	}
}

// slug is the public selector, so the database must enforce its uniqueness.
func TestIntegrationDuplicateSlugIsRejected(t *testing.T) {
	_, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)
	insert(t, pool, ctx, "dupe", "First", "echo", "m", true, 10)

	_, err := pool.Exec(ctx,
		`INSERT INTO llms (slug, name, provider_key, model) VALUES ($1,$2,$3,$4)`,
		"dupe", "Second", "echo", "m")
	if err == nil {
		t.Fatal("expected the unique constraint to reject a duplicate slug")
	}
}

// The seed migration is what a fresh checkout gets. If it stops inserting the
// mocks, every other integration test still passes while the app has no LLMs.
//
// This re-applies the seed file rather than trusting the table's current
// contents, because the tests above TRUNCATE. Asserting on leftover migration
// state would make this test pass or fail depending on execution order, which
// is worse than not having it.
func TestIntegrationSeedMigrationInsertedTheMockRows(t *testing.T) {
	s, pool, ctx := newTestStore(t)
	truncate(t, pool, ctx)

	seed, err := os.ReadFile("../../../migrations/0004_seed_llms.sql")
	if err != nil {
		t.Fatalf("read seed migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(seed)); err != nil {
		t.Fatalf("apply seed migration: %v", err)
	}

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	want := map[string]string{
		"echo-mock":   "echo",
		"script-mock": "script",
		"error-mock":  "error",
		"slow-mock":   "slow",
		"ghost-mock":  "nonexistent",
	}

	found := map[string]string{}
	for _, l := range got {
		found[l.Slug] = l.ProviderKey
	}

	for slug, key := range want {
		if found[slug] != key {
			t.Errorf("seeded row %q has provider_key %q, want %q", slug, found[slug], key)
		}
	}

	// ghost-mock is load-bearing: it is the configured-but-unrunnable row that
	// the resolution tests depend on. Guard it explicitly so a future tidy-up
	// that deletes it fails here with a reason.
	if found["ghost-mock"] != "nonexistent" {
		t.Error("ghost-mock must remain seeded with an unregistered provider_key")
	}
}
