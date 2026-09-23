// Package llm is the routing layer's foundation: the contract every language
// model is reached through, the registry that binds a configured LLM to an
// implementation, and the catalogue of LLMs on offer.
//
// The central idea is a deliberate split. The database says which LLMs are
// *offered*; the registry says which can actually *run*. They are not the same
// list, and the gap between them is resolved when a request names an LLM —
// never at startup, so one stale configuration row cannot take the service
// down.
package llm

import (
	"context"
	"errors"
	"strings"
	"time"
)

// Sentinel errors. Callers distinguish these with errors.Is, which is why the
// providers wrap rather than replace them.
var (
	// ErrEmptyConversation means the caller sent no messages. That is a bug in
	// the caller, not a failure of the model.
	ErrEmptyConversation = errors.New("conversation has no messages")

	// ErrProviderUnavailable means the model was reachable in configuration but
	// failed to answer. Distinct from a context error, which means we gave up.
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// Role identifies who authored a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// FinishReason says why generation stopped.
type FinishReason string

const (
	FinishReasonStop   FinishReason = "stop"
	FinishReasonLength FinishReason = "length"
	FinishReasonError  FinishReason = "error"
)

// Message is one turn in a conversation.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// Request is everything a provider needs to answer. Note what is absent: no
// conversation id, no user, no database handle. History arrives here in full,
// which is what lets the next message in a conversation go to a different
// model than the last one with no state to migrate.
type Request struct {
	Messages  []Message // full history, oldest first, never empty
	Model     string    // from the llms row, passed straight through
	System    string    // optional system prompt, kept out of Messages
	MaxTokens int       // 0 means the provider's own default
}

// LastUserMessage returns the most recent message authored by the user.
//
// The *last user* message, not the last message: a trailing assistant turn is
// normal when a conversation is being regenerated.
func (r Request) LastUserMessage() (Message, bool) {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role == RoleUser {
			return r.Messages[i], true
		}
	}
	return Message{}, false
}

// UserTurns counts how many turns the user has taken. Providers that vary
// their reply by position in the conversation use this.
func (r Request) UserTurns() int {
	n := 0
	for _, m := range r.Messages {
		if m.Role == RoleUser {
			n++
		}
	}
	return n
}

// Usage reports token consumption.
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// Response is one completion.
//
// ProviderKey and Model are echoed back out even though the caller supplied
// them. When a reply is eventually persisted against a conversation, the row
// records which model actually produced it — reading that from the response
// rather than the request keeps it honest if routing ever substitutes one.
type Response struct {
	Content      string        `json:"content"`
	ProviderKey  string        `json:"providerKey"`
	Model        string        `json:"model"`
	Usage        Usage         `json:"usage"`
	FinishReason FinishReason  `json:"finishReason"`
	Latency      time.Duration `json:"-"`
}

// Provider is the entire surface the rest of the application sees. One method,
// stateless, no knowledge of conversations or storage.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// estimateTokens is a deliberately crude word count. Real providers report
// real numbers; the mocks only need to be self-consistent.
func estimateTokens(s string) int {
	n := len(strings.Fields(s))
	if n == 0 && strings.TrimSpace(s) != "" {
		return 1
	}
	return n
}

// validate applies the rules every provider shares, so the routing layer does
// not have to special-case one of them.
func validate(req Request) error {
	if len(req.Messages) == 0 {
		return ErrEmptyConversation
	}
	return nil
}
