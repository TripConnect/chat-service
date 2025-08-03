package common

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TripConnect/chat-service/consts"
	"github.com/segmentio/kafka-go"
)

func Publish(ctx context.Context, topic string, data interface{}) error {
	// Publish message to Kafka, data should be passed as pointer
	if valueBytes, err := json.Marshal(data); err == nil {
		consts.KafkaPublisher.WriteMessages(ctx, kafka.Message{
			Topic: topic,
			Value: []byte(valueBytes),
		})
	} else {
		return fmt.Errorf("Publish kafka message failed %v", err)
	}

	return nil
}
