package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	sysContext "github.com/palaforcade/zchat/internal/context"
)

type Client interface {
	GenerateCommand(ctx context.Context, query string, sysCtx *sysContext.SystemContext) (string, error)
}

type AnthropicClient struct {
	client anthropic.Client
	model  string
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient(apiKey, model string) *AnthropicClient {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &AnthropicClient{
		client: client,
		model:  model,
	}
}

// GenerateCommand generates a shell command from a natural language query
func (c *AnthropicClient) GenerateCommand(ctx context.Context, query string, sysCtx *sysContext.SystemContext) (string, error) {
	// Build system prompt
	systemPrompt := buildSystemPrompt(sysCtx)

	// Create message request
	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{
				Type: "text",
				Text: systemPrompt,
			},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(query)),
		},
	})

	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	// Extract response text
	if len(message.Content) == 0 {
		return "", fmt.Errorf("received empty response from API")
	}

	responseText := message.Content[0].Text

	// Parse and clean the response
	command, err := parseCommandFromResponse(responseText)
	if err != nil {
		return "", fmt.Errorf("failed to parse command: %w", err)
	}

	return command, nil
}
