package converter

import (
	"github.com/kotaroyamazaki/line-bot/go-chatgpt-bot/pkg/chatgpt"
	"github.com/kotaroyamazaki/line-bot/go-chatgpt-bot/pkg/firestore"
)

func ToGPTMessage(msg *firestore.Message) *chatgpt.Message {
	return &chatgpt.Message{
		Role:    msg.Role,
		Content: msg.Content,
	}
}

func ToGPTMessages(msg []*firestore.Message) []*chatgpt.Message {
	a := make([]*chatgpt.Message, 0, len(msg))
	for _, v := range msg {
		a = append(a, ToGPTMessage(v))
	}
	return a
}
