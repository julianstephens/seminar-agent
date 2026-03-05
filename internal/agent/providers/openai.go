// Package providers contains concrete language-model backend adapters.
// Each adapter implements agent.Provider so the service layer can swap
// providers without code changes.
package providers

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"

	"github.com/julianstephens/formation/internal/agent"
)

// OpenAI implements agent.Provider using the OpenAI Chat Completions API.
// Create a single instance at startup via New and share it across requests.
type OpenAI struct {
	client *openai.Client
	model  string
}

// New returns an OpenAI provider configured with the supplied API key and
// model name (e.g. "gpt-4o").
func New(apiKey, model string) *OpenAI {
	return &OpenAI{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// Complete satisfies agent.Provider.
// It converts the agent.Message slice into OpenAI chat messages, calls the
// Chat Completions endpoint, and returns the first choice's content.
func (o *OpenAI) Complete(ctx context.Context, messages []agent.Message) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:    o.model,
		Messages: toOpenAIMessages(messages),
	}

	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("openai chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai returned empty choices list")
	}

	return resp.Choices[0].Message.Content, nil
}

// toOpenAIMessages converts the provider-agnostic Message slice to the
// openai SDK's ChatCompletionMessage type.
func toOpenAIMessages(msgs []agent.Message) []openai.ChatCompletionMessage {
	out := make([]openai.ChatCompletionMessage, len(msgs))
	for i, m := range msgs {
		out[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return out
}
