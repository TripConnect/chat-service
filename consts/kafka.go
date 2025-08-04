package consts

import (
	"fmt"

	"github.com/TripConnect/chat-service/helpers"
	"github.com/segmentio/kafka-go"
)

var kafkaAddres, _ = helpers.ReadConfig[string]("kafka.host")
var kafkaPort, _ = helpers.ReadConfig[int]("kafka.port")
var KafkaConnection = fmt.Sprintf("%s:%d", kafkaAddres, kafkaPort)

var KafkaPublisher = &kafka.Writer{
	Addr:                   kafka.TCP(KafkaConnection),
	Balancer:               &kafka.LeastBytes{},
	AllowAutoTopicCreation: true,
}
