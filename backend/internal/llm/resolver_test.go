package llm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// fakeStore is an in-memory Store. Because the resolver and handlers depend on
// the Store interface, these tests need no database.
type fakeStore struct {
	byslug map[string]LLM
	err    error // when set, every method returns it
}

func newFakeStore(items ...LLM) *fakeStore {
	f := &fakeStore{byslug: map[string]LLM{}}
	for _, l := range items {
		f.byslug[l.Slug] = l
	}
	return f
}

func (f *fakeStore) List(context.Context) ([]LLM, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]LLM, 0, len(f.byslug))
	for _, l := range f.byslug {
		out = append(out, l)
	}
	return out, nil
}

func (f *fakeStore) GetBySlug(_ context.Context, slug string) (LLM, error) {
	if f.err != nil {
		return LLM{}, f.err
	}
	l, ok := f.byslug[slug]
	if !ok {
		return LLM{}, ErrNotFound
	}
	return l, nil
}

// The seeded catalogue, mirroring migration 0004.
func seededCatalogue() *fakeStore {
	return newFakeStore(
		LLM{ID: 1, Slug: "echo-mock", Name: "Echo (mock)", ProviderKey: KeyEcho, Model: "echo-v1", Enabled: true},
		LLM{ID: 5, Slug: "ghost-mock", Name: "Unregistered (mock)", ProviderKey: "nonexistent", Model: "ghost-v1", Enabled: true},
	)
}

// The happy path: a configured slug yields a usable provider and the row's model.
func TestResolveReturnsAProviderAndTheConfiguredModel(t *testing.T) {
	r := NewResolver(seededCatalogue(), NewDefaultRegistry())

	got, err := r.Resolve(context.Background(), "echo-mock")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	if got.Model != "echo-v1" {
		t.Errorf("Model = %q, want %q", got.Model, "echo-v1")
	}
	if got.Provider == nil {
		t.Fatal("expected a provider")
	}

	// The provider must actually be usable, not merely non-nil.
	resp, err := got.Provider.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
		Model:    got.Model,
	})
	if err != nil {
		t.Fatalf("complete via resolved provider: %v", err)
	}
	if resp.ProviderKey != KeyEcho {
		t.Errorf("ProviderKey = %q, want %q", resp.ProviderKey, KeyEcho)
	}
}

// The case the whole split exists for: a row that is offered but not runnable.
func TestResolveConfiguredButUnregisteredReturnsNotRegistered(t *testing.T) {
	r := NewResolver(seededCatalogue(), NewDefaultRegistry())

	_, err := r.Resolve(context.Background(), "ghost-mock")
	if !errors.Is(err, ErrProviderNotRegistered) {
		t.Errorf("err = %v, want ErrProviderNotRegistered", err)
	}
}

// These two failures must stay distinguishable: one is a client asking for
// something that is not on offer (404), the other is our own configuration
// naming an implementation this build lacks (422). Collapsing them would tell
// an operator a config problem was a user error.
func TestResolveUnknownSlugIsDistinctFromUnregisteredProvider(t *testing.T) {
	r := NewResolver(seededCatalogue(), NewDefaultRegistry())

	_, unknownErr := r.Resolve(context.Background(), "no-such-slug")
	_, ghostErr := r.Resolve(context.Background(), "ghost-mock")

	if !errors.Is(unknownErr, ErrNotFound) {
		t.Errorf("unknown slug: err = %v, want ErrNotFound", unknownErr)
	}
	if errors.Is(unknownErr, ErrProviderNotRegistered) {
		t.Error("an unknown slug must not report as an unregistered provider")
	}
	if errors.Is(ghostErr, ErrNotFound) {
		t.Error("an unregistered provider must not report as a missing slug")
	}
}

// An operator should learn about a broken row from the logs at startup, not
// from the first user who picks it.
func TestCheckCatalogueWarnsAboutUnrunnableRows(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	err := CheckCatalogue(context.Background(), seededCatalogue(), NewDefaultRegistry(), logger)
	if err != nil {
		t.Fatalf("CheckCatalogue returned an error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "ghost-mock") {
		t.Errorf("expected a warning naming ghost-mock, got:\n%s", out)
	}
	if strings.Contains(out, "echo-mock") {
		t.Errorf("did not expect a warning about the runnable echo-mock, got:\n%s", out)
	}
}

// The service must still start. One bad configuration row is not worth the
// availability of every other LLM.
func TestCheckCatalogueDoesNotFailStartup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	if err := CheckCatalogue(context.Background(), seededCatalogue(), NewDefaultRegistry(), logger); err != nil {
		t.Errorf("CheckCatalogue must not fail startup for an unrunnable row, got %v", err)
	}
}
