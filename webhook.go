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

func Webhook(w http.ResponseWriter, r *http.Request) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.LevelKey = "severity"
	logger, _ := config.Build()
	defer logger.Sync()

	bot, err := linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil {
		http.Error(w, "Error init line bot", http.StatusBadRequest)
		logger.Error("failed to init line bot", zap.Error(err))
		return
	}

	events, err := bot.ParseRequest(r)
	if err != nil {
		http.Error(w, "Error parse request", http.StatusBadRequest)
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
					http.Error(w, "Error initializing firestore", http.StatusInternalServerError)
					logger.Error("failed to initialize firestore", zap.Error(err))
					return
				}
				defer firestoreCli.Close()

				// firestore から会話履歴の取得
				conv, err := firestoreCli.GetConversation(ctx, e.Source.UserID)
				if err != nil {
					http.Error(w, "failed to get conversation from firestore", http.StatusInternalServerError)
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
						ExpiresAt: time.Now().Add(30 * time.Minute),
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
					http.Error(w, "Error parse request", http.StatusInternalServerError)

					_, repErr := bot.ReplyMessage(e.ReplyToken, linebot.NewTextMessage("💥💥システム側で予期せぬエラーが発生しました💥💥")).Do()
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
					http.Error(w, "Error setting conversation in Firestore", http.StatusInternalServerError)
					logger.Error("failed to set conversation in firestore", zap.String("line_user_id", e.Source.UserID), zap.Error(err))
					return
				}

				// logging
				logger.Info("processing result", zap.String("line_user_id", e.Source.UserID), zap.String("line_display_name", prof.DisplayName), zap.String("line_message_id", message.ID), zap.String("line_text_message", query), zap.String("chat_gpt_reply_message", answer))

				reply := linebot.NewTextMessage(answer)
				_, err = bot.ReplyMessage(e.ReplyToken, reply).Do()
				if err != nil {
					http.Error(w, "Error parse request", http.StatusInternalServerError)
					logger.Error("failed to reply", zap.String("line_user_id", e.Source.UserID), zap.String("line_message_id", message.ID), zap.String("line_display_name", prof.DisplayName), zap.String("line_text_message", query), zap.Error(err))
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
	return
}