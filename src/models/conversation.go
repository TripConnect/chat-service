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

type ConversationEntity struct {
	Id        gocql.UUID `cql:"id"`
	AliasId   string     `cql:"alias_id"`
	OwnerId   gocql.UUID `cql:"owner_id"`
	Name      string     `cql:"name"`
	Type      int        `cql:"type"`
	CreatedAt time.Time  `cql:"created_at"`
}

type ParticipantEntity struct {
	ConversationId gocql.UUID `cql:"conversation_id"`
	UserId         gocql.UUID `cql:"user_id"`
	Status         int64      `cql:"status"`
}

type ConversationDocument struct {
	Id        gocql.UUID `json:"id"`
	AliasId   string     `json:"alias_id"`
	Name      string     `json:"name"`
	Type      int        `json:"type"`
	MemberIds []string   `json:"member_ids"`
	CreatedAt int        `json:"created_at"`
}

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

var ParticipantRepository = struct {
	recipes.CRUD
}{
	recipes.CRUD{
		TableInterface: gocqltable.NewKeyspace(constants.KeySpace).NewTable(
			constants.ParticipantTableName,
			[]string{"conversation_id", "user_id", "status"},
			nil,
			ParticipantEntity{},
		),
	},
}

func NewConversationDoc(entity ConversationEntity) ConversationDocument {
	return ConversationDocument{
		Id:        entity.Id,
		AliasId:   entity.AliasId,
		Name:      entity.Name,
		Type:      entity.Type,
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
