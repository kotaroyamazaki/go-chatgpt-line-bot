package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

type Conversation struct {
	Messages  []*Message
	ExpiresAt time.Time
}

const (
	CollectionName = "conversations"
	Expiration     = 30 * time.Minute
)

type Client interface {
	GetConversation(ctx context.Context, userID string) (*Conversation, error)
	SetConversation(ctx context.Context, userID string, conv *Conversation) error
	Close() error
}

type client struct {
	fsCli *firestore.Client
}

func New(ctx context.Context, projectID string) (Client, error) {
	cli, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &client{fsCli: cli}, nil
}

func (c *client) GetConversation(ctx context.Context, userID string) (*Conversation, error) {
	doc, err := c.fsCli.Collection(CollectionName).Doc(userID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	var conv Conversation
	err = doc.DataTo(&conv)
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func (c *client) SetConversation(ctx context.Context, userID string, conv *Conversation) error {
	_, err := c.fsCli.Collection(CollectionName).Doc(userID).Set(ctx, conv)
	return err
}

func (c *client) Close() error {
	return c.fsCli.Close()
}
