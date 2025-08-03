package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/TripConnect/chat-service/common"
	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/helpers"
	"github.com/TripConnect/chat-service/models"
	"github.com/segmentio/kafka-go"
	pb "github.com/tripconnect/go-proto-lib/protos"
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

		var kafkaPendingMessage models.KafkaPendingMessage
		if err := json.Unmarshal(m.Value, &kafkaPendingMessage); err != nil {
			fmt.Printf("error while comsume pending queue %v", err)
			return
		}

		correlationId := kafkaPendingMessage.CorrelationId

		if insertError := models.ChatMessageRepository.Insert(kafkaPendingMessage); insertError != nil {
			fmt.Printf("failed to create chat message %v", insertError)
			return
		}
		entity := models.NewChatMessageEntity(kafkaPendingMessage)
		chatMessageDoc := models.NewChatMessageDoc(entity)
		consts.ElasticsearchClient.
			Index(consts.ChatMessageIndex).
			Id(chatMessageDoc.Id.String()).
			Request(&chatMessageDoc).
			Do(ctx)

		newChatMessageTopic, _ := helpers.ReadConfig[string]("kafka.topic.chatting-fct-new-message")
		ack := &pb.CreateChatMessageAck{
			CorrelationId: correlationId,
		}
		if err := common.Publish(ctx, newChatMessageTopic, ack); err != nil {
			log.Printf("Saga chat message failed %s", err.Error())
		}
	}
}
