package consumers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/helpers"
	"github.com/TripConnect/chat-service/models"
	"github.com/segmentio/kafka-go"
)

func ListenPendingMessageQueue() {
	ctx := context.Background()
	pendingTopic, _ := helpers.ReadConfig[string]("kafka.topic.chatting-sys-internal-pending-queue")

	var listener = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{consts.KafkaConnection},
		GroupID:  "chat-service-internal",
		Topic:    pendingTopic,
		MaxBytes: 10e6, // 10MB
	})

	for {
		m, err := listener.ReadMessage(ctx)
		if err != nil {
			fmt.Printf("error while consume message %v", err)
			break
		}

		var chatMessage models.ChatMessageEntity
		if err := json.Unmarshal(m.Value, &chatMessage); err != nil {
			fmt.Printf("error while comsume pending queue %v", err)
			return
		}

		if insertError := models.ChatMessageRepository.Insert(chatMessage); insertError != nil {
			fmt.Printf("failed to create chat message %v", insertError)
			return
		}

		chatMessageDoc := models.NewChatMessageDoc(chatMessage)
		consts.ElasticsearchClient.
			Index(consts.ChatMessageIndex).
			Id(chatMessageDoc.Id.String()).
			Request(&chatMessageDoc).
			Do(ctx)
	}
}
