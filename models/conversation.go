package models

import (
	"time"

	"github.com/TripConnect/chat-service/consts"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/kristoiv/gocqltable/recipes"
	pb "github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ParticipantStatus int

const (
	Requested ParticipantStatus = 0
	Joined    ParticipantStatus = 1
)

type ConversationEntity struct {
	Id        string     `cql:"id"`
	OwnerId   gocql.UUID `cql:"owner_id"`
	Name      string     `cql:"name"`
	Type      int        `cql:"type"`
	CreatedAt time.Time  `cql:"created_at"`
}

type ParticipantEntity struct {
	ConversationId string     `cql:"conversation_id"`
	UserId         gocql.UUID `cql:"user_id"`
	Status         int        `cql:"status"`
}

type ConversationDocument struct {
	Id        string   `json:"id"`
	Name      string   `json:"name"`
	Type      int      `json:"type"`
	MemberIds []string `json:"member_ids"`
	CreatedAt int      `json:"created_at"`
}

var ConversationRepository = struct {
	recipes.CRUD
}{
	recipes.CRUD{
		TableInterface: gocqltable.NewKeyspace(consts.KeySpace).NewTable(
			consts.ConversationTableName,
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
		TableInterface: gocqltable.NewKeyspace(consts.KeySpace).NewTable(
			consts.ParticipantTableName,
			[]string{"conversation_id", "user_id", "status"},
			nil,
			ParticipantEntity{},
		),
	},
}

func NewConversationDoc(entity ConversationEntity) ConversationDocument {
	return ConversationDocument{
		Id:        entity.Id,
		Name:      entity.Name,
		Type:      entity.Type,
		CreatedAt: int(entity.CreatedAt.UnixMilli()),
	}
}

func NewConversationPb(entity ConversationEntity) pb.Conversation {
	// TODO: Sept members to another rpc with pagination, find on conversation_participants
	memberIds := []string{}

	return pb.Conversation{
		Id:        entity.Id,
		Type:      pb.ConversationType(entity.Type),
		Name:      entity.Name,
		MemberIds: memberIds,
		CreatedAt: timestamppb.New(entity.CreatedAt),
	}
}
