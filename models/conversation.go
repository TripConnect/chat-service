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

type ParticipantStatus int

const (
	Requested ParticipantStatus = 0
	Joined    ParticipantStatus = 1
)

type ConversationEntity struct {
	Id        gocql.UUID `cql:"id"`
	OwnerId   gocql.UUID `cql:"owner_id"`
	Name      string     `cql:"name"`
	Type      int        `cql:"type"`
	CreatedAt time.Time  `cql:"created_at"`
}

type ParticipantEntity struct {
	ConversationId gocql.UUID `cql:"conversation_id"`
	NickName       string     `cql:"nick_name"`
	UserId         gocql.UUID `cql:"user_id"`
	Status         int        `cql:"status"`
	CreatedAt      time.Time  `cql:"created_at"`
}

type ConversationDocument struct {
	Id        gocql.UUID `json:"id"`
	Name      string     `json:"name"`
	Type      int        `json:"type"`
	MemberIds []string   `json:"member_ids"`
	CreatedAt int        `json:"created_at"`
}

type ParticipantDocument struct {
	ConversationId gocql.UUID `json:"conversation_id"`
	UserId         gocql.UUID `json:"user_id"`
	Status         int        `json:"status"`
	CreatedAt      int        `json:"created_at"`
}

var ConversationDocumentMappings = esdsl.NewTypeMapping().
	AddProperty("id", esdsl.NewKeywordProperty()).
	AddProperty("name", esdsl.NewKeywordProperty()).
	AddProperty("type", esdsl.NewIntegerNumberProperty()).
	AddProperty("member_ids", esdsl.NewKeywordProperty()).
	AddProperty("created_at", esdsl.NewLongNumberProperty())

var ParticipantDocumentMappings = esdsl.NewTypeMapping().
	AddProperty("conversation_id", esdsl.NewKeywordProperty()).
	AddProperty("user_id", esdsl.NewKeywordProperty()).
	AddProperty("status", esdsl.NewIntegerNumberProperty()).
	AddProperty("created_at", esdsl.NewLongNumberProperty())

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

func NewConversationDoc(entity ConversationEntity, membersIds []string) ConversationDocument {
	return ConversationDocument{
		Id:        entity.Id,
		Name:      entity.Name,
		Type:      entity.Type,
		CreatedAt: int(entity.CreatedAt.UnixMilli()),
		MemberIds: membersIds,
	}
}

func NewParticipantDoc(entity ParticipantEntity, membersIds []string) ParticipantDocument {
	return ParticipantDocument{
		ConversationId: entity.ConversationId,
		UserId:         entity.UserId,
		Status:         entity.Status,
		CreatedAt:      int(entity.CreatedAt.UnixMilli()),
	}
}

func NewConversationPb(entity ConversationEntity, joinedMembers []ParticipantEntity) pb.Conversation {
	memberIds := []string{}
	for _, member := range joinedMembers {
		memberIds = append(memberIds, member.UserId.String())
	}

	return pb.Conversation{
		Id:        entity.Id.String(),
		Type:      pb.ConversationType(entity.Type),
		Name:      entity.Name,
		MemberIds: memberIds,
		CreatedAt: timestamppb.New(entity.CreatedAt),
	}
}
