// Package agent – Provider interface for language-model backends.
// Any struct that exposes a Complete method with the signature below
// automatically satisfies both Provider (agent package) and the AgentClient
// contract used by the service layer, keeping the two layers decoupled.
package agent

import "context"

// Provider is the interface satisfied by any language-model backend.
// The OpenAI adapter in internal/agent/providers/openai.go is the primary
// implementation; additional providers (Anthropic, local Ollama, etc.) can be
// added without touching the service layer.
type Provider interface {
	// Complete sends the ordered message list to the model and returns the
	// assistant's response text.  The caller is responsible for context
	// cancellation and timeout management.
	Complete(ctx context.Context, messages []Message) (string, error)

	// CompleteStream sends messages to the model and streams the response in chunks.
	// The chunkFn callback is invoked for each token/chunk as it arrives from the provider.
	// Returns the complete accumulated response text and any error.
	// The caller is responsible for context cancellation and timeout management.
	CompleteStream(ctx context.Context, messages []Message, chunkFn func(chunk string) error) (string, error)
}
