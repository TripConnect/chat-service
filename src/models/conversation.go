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
	Id        gocql.UUID `cql:"id"`
	AliasId   string     `cql:"alias_id"`
	OwnerId   gocql.UUID `cql:"owner_id"`
	Name      string     `cql:"name"`
	Type      int        `cql:"type"`
	CreatedAt time.Time  `cql:"created_at"`
}

type ConversationDocument struct {
	Id        gocql.UUID `json:"id"`
	AliasId   string     `json:"alias_id"`
	Name      string     `json:"name"`
	CreatedAt int        `json:"created_at"`
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

func NewConversationIndex(entity ConversationEntity) ConversationDocument {
	return ConversationDocument{
		Id:        entity.Id,
		AliasId:   entity.AliasId,
		Name:      entity.Name,
		CreatedAt: int(entity.CreatedAt.UnixMilli()),
	}
}

func NewConversationPb(entity ConversationEntity) pb.Conversation {
	var memberIds []string
	if entity.Type == int(pb.ConversationType_PRIVATE) {
		memberIds = strings.Split(entity.AliasId, constants.ElasticsearchSeparator)
	} else {
		memberIds = []string{} // TODO: Find on conversation_members
	}

	return pb.Conversation{
		Id:        entity.Id.String(),
		Type:      pb.ConversationType(entity.Type),
		Name:      entity.Name,
		MemberIds: memberIds,
		CreatedAt: timestamppb.New(entity.CreatedAt),
	}
}
