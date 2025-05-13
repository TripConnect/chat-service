package models

import (
	"time"

	constants "github.com/TripConnect/chat-service/src/consts"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/kristoiv/gocqltable/recipes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ChatMessageEntity struct {
	Id             gocql.UUID `cql:"id"`
	ConversationId string     `cql:"conversation_id"`
	FromUserId     gocql.UUID `cql:"from_user_id"`
	Content        string     `cql:"content"`
	CreatedAt      time.Time  `cql:"created_at"`
}

type ChatMessageIndex struct {
	Id             gocql.UUID `json:"id"`
	ConversationId string     `json:"conversation_id"`
	FromUserId     gocql.UUID `json:"from_user_id"`
	Content        string     `json:"content"`
	CreatedAt      int        `json:"created_at"`
}

var ChatMessageRepository = struct {
	recipes.CRUD
}{
	recipes.CRUD{
		TableInterface: gocqltable.NewKeyspace(constants.KeySpace).NewTable(
			constants.ChatMessageTableName,
			[]string{"id"},
			nil,
			ChatMessageEntity{},
		),
	},
}

func NewChatMessageIndex(entity ChatMessageEntity) ChatMessageIndex {
	return ChatMessageIndex{
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
