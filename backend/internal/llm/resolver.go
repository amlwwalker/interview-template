package llm

import (
	"context"
	"log/slog"
)

// Resolver turns a public slug into something that can actually answer.
//
// It is the one place the two halves of the design meet: the catalogue, which
// says what is on offer, and the registry, which says what this build can run.
// Keeping the join here means neither half has to know about the other.
type Resolver struct {
	store    Store
	registry *Registry
}

// NewResolver wires a catalogue to a registry.
func NewResolver(store Store, registry *Registry) *Resolver {
	return &Resolver{store: store, registry: registry}
}

// Resolved is a runnable LLM: the configuration row plus the implementation
// that serves it.
type Resolved struct {
	LLM      LLM
	Provider Provider
	Model    string
}

// Resolve looks up a slug and binds it to its implementation.
//
// The two failure modes stay deliberately distinct, because they mean
// different things to different people:
//
//   - ErrNotFound — the client asked for an LLM that is not on offer. Their
//     problem, a 404.
//   - ErrProviderNotRegistered — the catalogue names an implementation this
//     build does not have. Our problem, a 422, and worth an operator's
//     attention even though it only affects this one LLM.
//
// Collapsing them into one error would tell an operator that a configuration
// fault was a user typo.
func (r *Resolver) Resolve(ctx context.Context, slug string) (Resolved, error) {
	row, err := r.store.GetBySlug(ctx, slug)
	if err != nil {
		return Resolved{}, err
	}

	provider, err := r.registry.Get(row.ProviderKey)
	if err != nil {
		return Resolved{}, err
	}

	return Resolved{LLM: row, Provider: provider, Model: row.Model}, nil
}

// CheckCatalogue logs a warning for every enabled LLM whose provider_key has
// no implementation in this build.
//
// It deliberately returns nil for that condition rather than an error. Failing
// startup would catch configuration drift sooner, but would also mean one bad
// row takes down every other LLM — a trade that is not worth making. The
// warning gives an operator the same information without the outage.
//
// An error here means the catalogue itself could not be read, which is a
// genuine startup problem and is returned.
func CheckCatalogue(ctx context.Context, store Store, registry *Registry, logger *slog.Logger) error {
	items, err := store.List(ctx)
	if err != nil {
		return err
	}

	for _, l := range items {
		if _, err := registry.Get(l.ProviderKey); err != nil {
			logger.Warn("llm is configured but not runnable in this build",
				"slug", l.Slug,
				"provider_key", l.ProviderKey,
				"registered", registry.Keys(),
			)
		}
	}

	return nil
}
