package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/models"
	"github.com/segmentio/kafka-go"
	"github.com/tripconnect/go-common-utils/common"
	"github.com/tripconnect/go-common-utils/helper"
)

func ListenPendingMessageQueue() {
	ctx := context.Background()
	pendingTopic, _ := helper.ReadConfig[string]("kafka.topic.chatting-sys-internal-pending-queue")

	var listener = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{common.KafkaConnection},
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

		var kafkaPendingMessage models.KafkaPendingMessage
		if err := json.Unmarshal(m.Value, &kafkaPendingMessage); err != nil {
			fmt.Printf("error while comsume pending queue %v", err)
			return
		}

		// Saving related
		entity := models.NewChatMessageEntity(kafkaPendingMessage)
		if insertError := models.ChatMessageRepository.Insert(entity); insertError != nil {
			fmt.Printf("failed to create chat message %v", insertError)
			return
		}
		chatMessageDoc := models.NewChatMessageDoc(entity)
		_, saveEsErr := common.ElasticsearchClient.
			Index(consts.ChatMessageIndex).
			Id(chatMessageDoc.Id.String()).
			Request(&chatMessageDoc).
			Do(ctx)
		if saveEsErr != nil {
			fmt.Printf("Failed to save es: %v", saveEsErr)
		}

		// Saga related
		sentChatMessageTopic, _ := helper.ReadConfig[string]("kafka.topic.chatting-fct-sent-message")
		ack := &models.KafkaSentMessage{
			Id:             entity.Id,
			ConversationId: entity.ConversationId,
			FromUserId:     entity.FromUserId,
			Content:        entity.Content,
			SentTime:       entity.SentTime,
			CreatedAt:      entity.CreatedAt,
		}
		if err := common.Publish(ctx, sentChatMessageTopic, ack); err != nil {
			log.Printf("Saga chat message failed %s", err.Error())
		}
	}
}
