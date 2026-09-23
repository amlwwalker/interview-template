package llm

import (
	"context"
	"fmt"
	"time"
)

// The mock providers. During development these are the only implementations,
// so they are the contract everything else is built and tested against.
//
// Two of them answer and two of them fail, on purpose. A layer whose error
// paths are never executed in testing is a layer whose error paths are
// unverified, and those are the ones that matter when a real provider starts
// timing out in production.

// Provider keys, which are what an llms row stores in provider_key.
const (
	KeyEcho   = "echo"
	KeyScript = "script"
	KeyError  = "error"
	KeySlow   = "slow"
)

// echoProvider returns a predictable transformation of the last user message.
// Deterministic output is what lets tests further up assert exact strings.
type echoProvider struct{}

func newEchoProvider() *echoProvider { return &echoProvider{} }

func (p *echoProvider) Complete(ctx context.Context, req Request) (Response, error) {
	if err := validate(req); err != nil {
		return Response{}, err
	}

	// Even a provider that does no work honours cancellation, so that a caller
	// that has already given up never receives a late success.
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	last, ok := req.LastUserMessage()
	if !ok {
		return Response{}, ErrEmptyConversation
	}

	content := "echo: " + last.Content
	prompt := estimateTokens(last.Content)
	completion := estimateTokens(content)

	return Response{
		Content:     content,
		ProviderKey: KeyEcho,
		Model:       req.Model,
		Usage: Usage{
			PromptTokens:     prompt,
			CompletionTokens: completion,
			TotalTokens:      prompt + completion,
		},
		FinishReason: FinishReasonStop,
	}, nil
}

// scriptLines is the fixed script the script provider walks through. Exported
// within the package so tests assert against it rather than duplicating it.
var scriptLines = []string{
	"Hello — what would you like to talk about?",
	"That's interesting. Tell me more.",
	"I see. What makes you say that?",
	"Understood. Anything else?",
}

// scriptProvider answers according to how far into the conversation it is.
// This is the provider that proves history actually arrives: if a caller only
// forwarded the latest message, every turn would return line one.
type scriptProvider struct{}

func newScriptProvider() *scriptProvider { return &scriptProvider{} }

func (p *scriptProvider) Complete(ctx context.Context, req Request) (Response, error) {
	if err := validate(req); err != nil {
		return Response{}, err
	}
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	// Turn 1 is index 0. Wrap rather than panic on a conversation longer than
	// the script — a mock should not be able to crash the service.
	turns := req.UserTurns()
	if turns == 0 {
		turns = 1
	}
	content := scriptLines[(turns-1)%len(scriptLines)]

	prompt := 0
	for _, m := range req.Messages {
		prompt += estimateTokens(m.Content)
	}
	completion := estimateTokens(content)

	return Response{
		Content:     content,
		ProviderKey: KeyScript,
		Model:       req.Model,
		Usage: Usage{
			PromptTokens:     prompt,
			CompletionTokens: completion,
			TotalTokens:      prompt + completion,
		},
		FinishReason: FinishReasonStop,
	}, nil
}

// errorProvider always fails. It exists so the failure branch of the routing
// layer is reachable in a test rather than only in production.
type errorProvider struct{}

func newErrorProvider() *errorProvider { return &errorProvider{} }

func (p *errorProvider) Complete(_ context.Context, req Request) (Response, error) {
	if err := validate(req); err != nil {
		return Response{}, err
	}
	return Response{}, fmt.Errorf("mock provider %q always fails: %w", KeyError, ErrProviderUnavailable)
}

// slowProvider never answers. It blocks until the caller's context is done and
// then reports why, which is how timeout and cancellation handling upstream
// gets verified instead of assumed.
type slowProvider struct{}

func newSlowProvider() *slowProvider { return &slowProvider{} }

func (p *slowProvider) Complete(ctx context.Context, req Request) (Response, error) {
	if err := validate(req); err != nil {
		return Response{}, err
	}

	select {
	case <-ctx.Done():
		// Return the context's own error so errors.Is finds DeadlineExceeded
		// or Canceled at the call site.
		return Response{}, ctx.Err()
	case <-time.After(time.Hour):
		// Unreachable in practice. Present so the select has a second arm and
		// the intent — "this never answers" — is explicit.
		return Response{}, ErrProviderUnavailable
	}
}

// newDefaultProviders is the set compiled into this build, keyed by the value
// an llms row carries in provider_key.
//
// A slug configured with a key absent from this map is not runnable. That is a
// supported state, not a crash: it resolves to ErrProviderNotRegistered when a
// request names it, and affects no other LLM.
func newDefaultProviders() map[string]Provider {
	return map[string]Provider{
		KeyEcho:   newEchoProvider(),
		KeyScript: newScriptProvider(),
		KeyError:  newErrorProvider(),
		KeySlow:   newSlowProvider(),
	}
}
