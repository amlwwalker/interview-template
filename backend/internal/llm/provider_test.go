package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// A conversation that has gone back and forth, used wherever a test needs
// history rather than a single opening message.
func threeTurnConversation() []Message {
	return []Message{
		{Role: RoleUser, Content: "first"},
		{Role: RoleAssistant, Content: "reply one"},
		{Role: RoleUser, Content: "second"},
		{Role: RoleAssistant, Content: "reply two"},
		{Role: RoleUser, Content: "third"},
	}
}

// The echo provider is the deterministic one: same input, same output, so
// tests downstream can assert exact strings rather than shapes.
func TestEchoProviderEchoesTheLastUserMessage(t *testing.T) {
	p := newEchoProvider()

	got, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
		Model:    "echo-v1",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if got.Content != "echo: hello" {
		t.Errorf("Content = %q, want %q", got.Content, "echo: hello")
	}
}

// The *last user* message, not the last message. A trailing assistant turn is
// normal when a conversation is being regenerated, and echoing it would mean
// the provider was reading the wrong end of the history.
func TestEchoProviderIgnoresATrailingAssistantMessage(t *testing.T) {
	p := newEchoProvider()

	got, err := p.Complete(context.Background(), Request{
		Messages: []Message{
			{Role: RoleUser, Content: "the real prompt"},
			{Role: RoleAssistant, Content: "an earlier answer"},
		},
		Model: "echo-v1",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if got.Content != "echo: the real prompt" {
		t.Errorf("Content = %q, want it to echo the last *user* message", got.Content)
	}
}

// Usage must be internally consistent, so a caller summing tokens across a
// conversation gets a number that means something.
func TestEchoProviderReportsConsistentUsage(t *testing.T) {
	p := newEchoProvider()

	got, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "one two three"}},
		Model:    "echo-v1",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if got.Usage.TotalTokens != got.Usage.PromptTokens+got.Usage.CompletionTokens {
		t.Errorf("usage does not add up: %+v", got.Usage)
	}
	if got.Usage.PromptTokens == 0 {
		t.Error("expected a non-zero prompt token count")
	}
}

// The model is configuration carried on the llms row. It must survive the
// round trip so a persisted response records which model actually answered.
func TestEchoProviderReturnsTheModelItWasGiven(t *testing.T) {
	p := newEchoProvider()

	got, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
		Model:    "echo-v7",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if got.Model != "echo-v7" {
		t.Errorf("Model = %q, want %q", got.Model, "echo-v7")
	}
}

// The script provider exists to prove history reaches the provider at all.
func TestScriptProviderReturnsItsFirstLineOnTheOpeningTurn(t *testing.T) {
	p := newScriptProvider()

	got, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
		Model:    "script-v1",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if got.Content != scriptLines[0] {
		t.Errorf("Content = %q, want the first script line %q", got.Content, scriptLines[0])
	}
}

// This is the test that would catch a provider being handed only the latest
// message: with full history the third user turn must differ from the first.
func TestScriptProviderAdvancesWithTheNumberOfUserTurns(t *testing.T) {
	p := newScriptProvider()

	first, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "first"}},
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	third, err := p.Complete(context.Background(), Request{
		Messages: threeTurnConversation(),
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if first.Content == third.Content {
		t.Fatalf("turn 1 and turn 3 both returned %q; history is not reaching the provider", first.Content)
	}
	if third.Content != scriptLines[2] {
		t.Errorf("Content = %q, want the third script line %q", third.Content, scriptLines[2])
	}
}

// A long conversation must not index past the end of the script.
func TestScriptProviderWrapsRatherThanPanicking(t *testing.T) {
	p := newScriptProvider()

	msgs := make([]Message, 0, (len(scriptLines)+3)*2)
	for i := 0; i < len(scriptLines)+3; i++ {
		msgs = append(msgs, Message{Role: RoleUser, Content: "turn"})
		msgs = append(msgs, Message{Role: RoleAssistant, Content: "reply"})
	}

	got, err := p.Complete(context.Background(), Request{Messages: msgs})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got.Content == "" {
		t.Error("expected the script to wrap and return a line")
	}
}

// The error provider makes the failure branch reachable. errors.Is matters:
// callers must be able to distinguish this from a context error.
func TestErrorProviderReturnsAMatchableSentinel(t *testing.T) {
	p := newErrorProvider()

	_, err := p.Complete(context.Background(), Request{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Errorf("err = %v, want it to match ErrProviderUnavailable", err)
	}
}

// The slow provider exists so timeout handling is verified rather than assumed.
func TestSlowProviderRespectsAContextDeadline(t *testing.T) {
	p := newSlowProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := p.Complete(ctx, Request{Messages: []Message{{Role: RoleUser, Content: "hello"}}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

// Cancellation is distinct from a deadline: the caller gave up deliberately,
// and the error must say so rather than reporting a timeout.
func TestSlowProviderRespectsCancellation(t *testing.T) {
	p := newSlowProvider()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := p.Complete(ctx, Request{Messages: []Message{{Role: RoleUser, Content: "hello"}}})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// Returning the context error is not enough — it has to return *promptly*.
// A provider that runs to completion and then reports the deadline has not
// actually honoured cancellation, and under load that is the difference
// between a slow request and an exhausted connection pool.
func TestSlowProviderReturnsPromptlyOnCancellation(t *testing.T) {
	p := newSlowProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, err := p.Complete(ctx, Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}}); err == nil {
		t.Fatal("expected a context error")
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("took %v to honour a 50ms deadline; cancellation is not being observed", elapsed)
	}
}

// An empty conversation is a caller bug. Every provider must reject it the
// same way, or the routing layer has to special-case each one.
func TestEveryProviderRejectsAnEmptyMessageList(t *testing.T) {
	for key, p := range newDefaultProviders() {
		t.Run(key, func(t *testing.T) {
			_, err := p.Complete(context.Background(), Request{Messages: nil})
			if !errors.Is(err, ErrEmptyConversation) {
				t.Errorf("err = %v, want ErrEmptyConversation", err)
			}
		})
	}
}

// Whatever a provider does, a successful response must identify itself and
// say why it stopped. Table-driven across every provider that can succeed.
func TestSuccessfulResponsesIdentifyThemselves(t *testing.T) {
	for key, p := range newDefaultProviders() {
		t.Run(key, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			got, err := p.Complete(ctx, Request{
				Messages: []Message{{Role: RoleUser, Content: "hello"}},
				Model:    "m-1",
			})
			if err != nil {
				// error and slow providers are expected to fail; they are
				// covered by their own tests above.
				return
			}

			if got.ProviderKey != key {
				t.Errorf("ProviderKey = %q, want %q", got.ProviderKey, key)
			}
			if got.FinishReason != FinishReasonStop {
				t.Errorf("FinishReason = %q, want %q", got.FinishReason, FinishReasonStop)
			}
			if strings.TrimSpace(got.Content) == "" {
				t.Error("successful response has empty content")
			}
		})
	}
}
