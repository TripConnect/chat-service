package models

import (
	"strings"
	"time"

	constants "github.com/TripConnect/chat-service/src/consts"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/kristoiv/gocqltable/recipes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConversationEntity represents a conversation record in the database.
type ConversationEntity struct {
	Id        string     `cql:"id"`
	OwnerId   gocql.UUID `cql:"owner_id"`
	Name      string     `cql:"name"`
	Type      int        `cql:"type"`
	CreatedAt time.Time  `cql:"created_at"`
}

type ConversationIndex struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int    `json:"created_at"`
}

// ConversationRepository provides CRUD operations for the conversations table.
var ConversationRepository = struct {
	recipes.CRUD
}{
	recipes.CRUD{
		TableInterface: gocqltable.NewKeyspace(constants.KeySpace).NewTable(
			constants.ConversationTableName,
			[]string{"id"},
			nil,
			ConversationEntity{},
		),
	},
}

func NewConversationIndex(entity ConversationEntity) ConversationIndex {
	return ConversationIndex{
		Id:        entity.Id,
		Name:      entity.Name,
		CreatedAt: int(entity.CreatedAt.UnixMilli()),
	}
}

func NewConversationPb(entity ConversationEntity) pb.Conversation {
	var memberIds []string
	if entity.Type == int(pb.ConversationType_PRIVATE) {
		memberIds = strings.Split(entity.Id, constants.ElasticsearchSeparator)
	} else {
		memberIds = []string{} // TODO: Find on conversation_members
	}

	return pb.Conversation{
		Id:        entity.Id,
		Type:      pb.ConversationType(entity.Type),
		Name:      entity.Name,
		MemberIds: memberIds,
		CreatedAt: timestamppb.New(entity.CreatedAt),
	}
}
