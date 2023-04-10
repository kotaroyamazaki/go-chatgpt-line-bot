package chatgpt

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

const (
	RoleUser   string = openai.ChatMessageRoleUser
	RoleSystem string = openai.ChatMessageRoleSystem
)

type Client interface {
	Chat(ctx context.Context, text string, opts ...Option) (string, error)
}

type client struct {
	*openai.Client
}

type Option func(*options)

type options struct {
	messages []*Message
}

type Message struct {
	Role    string
	Content string
}

func WithMessages(messages []*Message) Option {
	return func(o *options) {
		o.messages = messages
	}
}

func New(apiKey string) Client {
	return &client{openai.NewClient(apiKey)}
}

func (c *client) Chat(ctx context.Context, text string, opts ...Option) (string, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	messages := make([]openai.ChatCompletionMessage, len(o.messages)+1)
	for i, m := range o.messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	messages[len(o.messages)] = openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: text,
	}

	resp, err := c.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
