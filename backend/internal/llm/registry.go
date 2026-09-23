package llm

import (
	"errors"
	"fmt"
	"sort"
)

// ErrProviderNotRegistered means an llms row named a provider_key that this
// build has no implementation for.
//
// This is a supported state, not a crash. Configuration lives in the database
// and changes without a deploy; implementations ship with the binary. When the
// two disagree the single request naming that LLM fails, and everything else
// keeps serving.
var ErrProviderNotRegistered = errors.New("provider not registered")

// Registry maps the provider_key on an llms row to a compiled Provider.
//
// It is built once at startup and only read afterwards, so it needs no lock:
// the map is never written after construction. MustRegister exists for wiring
// during startup and panics rather than returning an error, because a
// duplicate key is a programming mistake that should stop the process.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry builds a registry from a set of providers. The map is copied, so
// a caller mutating theirs afterwards cannot race with readers.
func NewRegistry(providers map[string]Provider) *Registry {
	copied := make(map[string]Provider, len(providers))
	for k, v := range providers {
		copied[k] = v
	}
	return &Registry{providers: copied}
}

// NewDefaultRegistry is the set compiled into this build: the four mocks.
// Real providers register here as they are added.
func NewDefaultRegistry() *Registry {
	return NewRegistry(newDefaultProviders())
}

// MustRegister adds a provider, panicking if the key is already taken.
//
// Two implementations claiming one key means whichever registered last wins
// silently. Failing at startup is the cheaper outcome by a wide margin.
func (r *Registry) MustRegister(key string, p Provider) {
	if _, exists := r.providers[key]; exists {
		panic(fmt.Sprintf("llm: provider %q registered twice", key))
	}
	r.providers[key] = p
}

// Get returns the provider for a key, or ErrProviderNotRegistered.
//
// The error is always accompanied by a nil provider. A blank key takes the
// same path as an unknown one, so a row with an empty provider_key produces a
// clean 422 rather than a nil-pointer panic at the call site.
func (r *Registry) Get(key string) (Provider, error) {
	p, ok := r.providers[key]
	if !ok {
		return nil, fmt.Errorf("%q: %w", key, ErrProviderNotRegistered)
	}
	return p, nil
}

// Keys returns every registered key, sorted. Used by the startup check that
// warns about configured-but-unrunnable rows, and by tests pinning the set.
func (r *Registry) Keys() []string {
	keys := make([]string, 0, len(r.providers))
	for k := range r.providers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
