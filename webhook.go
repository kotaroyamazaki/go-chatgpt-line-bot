package linebot

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/kotaroyamazaki/line-bot/go-chatgpt-bot/converter"
	"github.com/kotaroyamazaki/line-bot/go-chatgpt-bot/pkg/chatgpt"
	"github.com/kotaroyamazaki/line-bot/go-chatgpt-bot/pkg/firestore"

	"github.com/line/line-bot-sdk-go/linebot"
	"go.uber.org/zap"
)

const (
	conversationExpirationTime = 30 * time.Minute
)

var logger *zap.Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.LevelKey = "severity"

	var err error
	logger, err = config.Build()
	if err != nil {
		fmt.Println("Failed to initialize logger:", err)
		return
	}
	defer logger.Sync()
}

func Webhook(w http.ResponseWriter, r *http.Request) {
	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		http.Error(w, "Failed to initialize Line bot", http.StatusInternalServerError)
		logger.Error("failed to init line bot", zap.Error(err))
		return
	}

	events, err := bot.ParseRequest(r)
	if err != nil {
		http.Error(w, "Failed to parse Line request", http.StatusBadRequest)
		logger.Error("failed to parse request", zap.Error(err))
		return
	}

	ctx := r.Context()
	for _, e := range events {
		switch e.Type {
		case linebot.EventTypeMessage:
			switch message := e.Message.(type) {
			case *linebot.TextMessage:
				query := message.Text
				// get profile for logs
				prof, err := bot.GetProfile(e.Source.UserID).Do()
				if err != nil {
					prof = &linebot.UserProfileResponse{}
					logger.Warn("get profile", zap.String("line_user_id", e.Source.UserID), zap.String("line_message_id", message.ID))
				}

				firestoreCli, err := firestore.New(ctx, os.Getenv("GCP_PROJECT_ID"))
				if err != nil {
					http.Error(w, "Failed to initialize Firestore", http.StatusInternalServerError)
					logger.Error("failed to initialize firestore", zap.Error(err))
					return
				}
				defer firestoreCli.Close()

				// firestore から会話履歴の取得
				conv, err := firestoreCli.GetConversation(ctx, e.Source.UserID)
				if err != nil {
					http.Error(w, "Failed to get conversation from firestore", http.StatusInternalServerError)
					logger.Error("failed to get conversation from firestore", zap.String("line_user_id", e.Source.UserID), zap.Error(err))
					return
				}
				// 有効期限が切れていたら初期化する
				if conv != nil && time.Now().After(conv.ExpiresAt) {
					conv = nil
				}
				if conv == nil {
					conv = &firestore.Conversation{
						Messages:  make([]*firestore.Message, 0),
						ExpiresAt: time.Now().Add(conversationExpirationTime),
					}
				}

				userMessage := &firestore.Message{
					Role:      chatgpt.RoleUser,
					Content:   query,
					Timestamp: time.Now(),
				}
				conv.Messages = append(conv.Messages, userMessage)

				chatCli := chatgpt.New(os.Getenv("OPENAI_API_KEY"))
				answer, err := chatCli.Chat(ctx, query, chatgpt.WithMessages(converter.ToGPTMessages(conv.Messages)))
				if err != nil {
					http.Error(w, "Failed to call ChatGPT API", http.StatusInternalServerError)

					var repErr error
					if chatgpt.IsErrorTooManyRequest(err) {
						_, repErr = bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage("⚠️API利用制限につき一時的に利用できなくなっている可能性があります。時間を開けて再度ご利用下さい⚠️")).Do()
					} else {
						_, repErr = bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage("💥💥システム側で予期せぬエラーが発生しました💥💥")).Do()
					}
					if repErr != nil {
						logger.Error("failed to reply unexpected error", zap.String("line_user_id", e.Source.UserID), zap.String("line_display_name", prof.DisplayName), zap.String("line_message_id", message.ID), zap.String("line_text_message", query), zap.Error(repErr), zap.NamedError("original error", err))
						return
					}
					logger.Error("failed to call chatgpt api", zap.String("line_user_id", e.Source.UserID), zap.String("line_display_name", prof.DisplayName), zap.String("line_message_id", message.ID), zap.String("line_text_message", query), zap.Error(err))
					return
				}
				gptMessage := &firestore.Message{
					Role:      chatgpt.RoleSystem,
					Content:   answer,
					Timestamp: time.Now(),
				}
				conv.Messages = append(conv.Messages, gptMessage)

				// firestore に会話履歴を保持
				err = firestoreCli.SetConversation(ctx, e.Source.UserID, conv)
				if err != nil {
					http.Error(w, "Failed to set conversation in Firestore", http.StatusInternalServerError)
					logger.Error("failed to set conversation in firestore", zap.String("line_user_id", e.Source.UserID), zap.Error(err))
					return
				}

				// logging
				logger.Info("ChatGPT reply sent",
					zap.String("line_user_id", e.Source.UserID),
					zap.String("line_display_name", prof.DisplayName),
					zap.String("line_message_id", message.ID),
					zap.String("chat_gpt_reply_message", answer))

				reply := linebot.NewTextMessage(answer)
				_, err = bot.ReplyMessage(e.ReplyToken, reply).Do()
				if err != nil {
					http.Error(w, "Failed to reply to user", http.StatusInternalServerError)
					logger.Error("failed to reply", zap.String("line_user_id", e.Source.UserID), zap.String("line_message_id", message.ID), zap.String("line_display_name", prof.DisplayName), zap.String("line_text_message", query), zap.Error(err))
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
	return
}
