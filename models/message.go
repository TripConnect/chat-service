package models

import (
	"time"

	"github.com/TripConnect/chat-service/consts"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/kristoiv/gocqltable/recipes"
	pb "github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatMessageEntity struct {
	Id             gocql.UUID `cql:"id"`
	ConversationId string     `cql:"conversation_id"`
	FromUserId     gocql.UUID `cql:"from_user_id"`
	Content        string     `cql:"content"`
	SentTime       time.Time  `cql:"sent_time"`
	CreatedAt      time.Time  `cql:"created_at"`
}

type ChatMessageDocument struct {
	Id             gocql.UUID `json:"id"`
	ConversationId string     `json:"conversation_id"`
	FromUserId     gocql.UUID `json:"from_user_id"`
	Content        string     `json:"content"`
	CreatedAt      int        `json:"created_at"`
}

type KafkaPendingMessage struct {
	CorrelationId  string     `json:"correlation_id"`
	ConversationId string     `json:"conversation_id"`
	FromUserId     gocql.UUID `json:"from_user_id"`
	Content        string     `json:"content"`
	SentTime       time.Time  `json:"sent_time"`
}

var ChatMessageDocumentMappings = esdsl.NewTypeMapping().
	AddProperty("id", esdsl.NewKeywordProperty()).
	AddProperty("conversation_id", esdsl.NewKeywordProperty()).
	AddProperty("from_user_id", esdsl.NewKeywordProperty()).
	AddProperty("content", esdsl.NewKeywordProperty()).
	AddProperty("created_at", esdsl.NewLongNumberProperty())

var ChatMessageRepository = struct {
	recipes.CRUD
}{
	recipes.CRUD{
		TableInterface: gocqltable.NewKeyspace(consts.KeySpace).NewTable(
			consts.ChatMessageTableName,
			[]string{"id"},
			nil,
			ChatMessageEntity{},
		),
	},
}

func NewChatMessageEntity(data KafkaPendingMessage) ChatMessageEntity {
	return ChatMessageEntity{
		Id:             gocql.MustRandomUUID(),
		ConversationId: data.ConversationId,
		FromUserId:     data.FromUserId,
		Content:        data.Content,
		SentTime:       data.SentTime,
		CreatedAt:      time.Now(),
	}
}

func NewChatMessageDoc(entity ChatMessageEntity) ChatMessageDocument {
	return ChatMessageDocument{
		Id:             entity.Id,
		ConversationId: entity.ConversationId,
		FromUserId:     entity.FromUserId,
		Content:        entity.Content,
		CreatedAt:      int(entity.CreatedAt.UnixMilli()),
	}
}

func NewChatMessagePb(entity ChatMessageEntity) pb.ChatMessage {
	return pb.ChatMessage{
		Id:             entity.Id.String(),
		ConversationId: entity.ConversationId,
		FromUserId:     entity.FromUserId.String(),
		Content:        entity.Content,
		CreatedAt:      timestamppb.New(entity.CreatedAt),
	}
}
