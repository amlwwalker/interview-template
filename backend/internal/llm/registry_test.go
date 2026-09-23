package llm

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// stubProvider is a provider that does nothing, used where a test only cares
// about identity rather than behaviour.
type stubProvider struct{ name string }

func (s *stubProvider) Complete(context.Context, Request) (Response, error) {
	return Response{Content: s.name}, nil
}

// The registry returns the exact implementation registered, not a copy or a
// lookalike — identity matters because providers are stateful singletons.
func TestRegistryReturnsTheProviderRegisteredUnderAKey(t *testing.T) {
	want := &stubProvider{name: "mine"}

	r := NewRegistry(map[string]Provider{"mine": want})

	got, err := r.Get("mine")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != want {
		t.Error("Get returned a different provider than the one registered")
	}
}

// The case this whole design exists to survive: a configuration row naming an
// implementation this build does not have.
func TestRegistryUnknownKeyReturnsAMatchableError(t *testing.T) {
	r := NewRegistry(map[string]Provider{"mine": &stubProvider{}})

	_, err := r.Get("nonexistent")
	if !errors.Is(err, ErrProviderNotRegistered) {
		t.Errorf("err = %v, want ErrProviderNotRegistered", err)
	}
}

// An empty key must be an error, not a nil provider — otherwise a row with a
// blank provider_key produces a nil-pointer panic at the call site instead of
// a 422.
func TestRegistryEmptyKeyReturnsAnErrorNotANilProvider(t *testing.T) {
	r := NewRegistry(map[string]Provider{"mine": &stubProvider{}})

	got, err := r.Get("")
	if !errors.Is(err, ErrProviderNotRegistered) {
		t.Errorf("err = %v, want ErrProviderNotRegistered", err)
	}
	if got != nil {
		t.Error("expected a nil provider alongside the error")
	}
}

// Registering twice under one key means two implementations disagree about who
// serves it. That is a programming error, and it should surface at startup
// rather than as whichever one happened to win.
func TestRegistryPanicsOnDuplicateRegistration(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic on duplicate registration")
		}
	}()

	r := NewRegistry(nil)
	r.MustRegister("dupe", &stubProvider{})
	r.MustRegister("dupe", &stubProvider{})
}

// The default registry is what main wires up. If a key goes missing, every
// llms row pointing at it silently stops working, so pin the set.
func TestDefaultRegistryHoldsExactlyTheFourMockProviders(t *testing.T) {
	r := NewDefaultRegistry()

	want := []string{KeyEcho, KeyScript, KeyError, KeySlow}
	for _, key := range want {
		if _, err := r.Get(key); err != nil {
			t.Errorf("Get(%q) = %v, want it registered", key, err)
		}
	}

	if got := len(r.Keys()); got != len(want) {
		t.Errorf("registry holds %d providers (%v), want %d", got, r.Keys(), len(want))
	}
}

// Keys is what the startup warning iterates, so it must be stable to read.
func TestRegistryKeysAreSorted(t *testing.T) {
	r := NewRegistry(map[string]Provider{
		"zebra": &stubProvider{}, "alpha": &stubProvider{}, "mike": &stubProvider{},
	})

	got := r.Keys()
	want := []string{"alpha", "mike", "zebra"}

	if len(got) != len(want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Keys() = %v, want %v", got, want)
		}
	}
}

// The registry is read by every concurrent request. Run under -race.
func TestRegistryConcurrentReadsAreRaceFree(t *testing.T) {
	r := NewDefaultRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.Get(KeyEcho); err != nil {
				t.Errorf("get: %v", err)
			}
			_ = r.Keys()
		}()
	}
	wg.Wait()
}
