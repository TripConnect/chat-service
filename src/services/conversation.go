package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	constants "github.com/TripConnect/chat-service/src/consts"
	models "github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/gocql/gocql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateConversation(req *pb.CreateConversationRequest) (*pb.Conversation, error) {
	var conversationId string
	var ownerId gocql.UUID

	if req.GetType() == pb.ConversationType_PRIVATE {
		memberIds := req.GetMemberIds()
		sort.Slice(memberIds, func(i, j int) bool {
			return memberIds[i] > memberIds[j]
		})

		conversationId = strings.Join(memberIds, constants.ElasticsearchSeparator)
		ownerId, _ = gocql.ParseUUID("11111111-1111-1111-1111-111111111111")

		existConversation, _ := models.ConversationRepository.Get(conversationId)
		if existConversation != nil {
			conversationPb := models.NewConversationPb(*existConversation.(*models.ConversationEntity))
			return &conversationPb, nil
		}
	} else {
		var ownerError error

		conversationId = gocql.MustRandomUUID().String()
		ownerId, ownerError = gocql.ParseUUID(req.GetOwnerId())
		if ownerError != nil {
			return nil, status.New(codes.InvalidArgument, "invalid ownerId").Err()
		}
	}

	conversation := models.ConversationEntity{
		Id:        conversationId,
		Name:      req.GetName(),
		Type:      int(req.GetType()),
		OwnerId:   ownerId,
		CreatedAt: time.Now(),
	}

	insertErr := models.ConversationRepository.Insert(conversation)
	if insertErr != nil {
		log.Fatalf("Failed to insert conversation: %v", insertErr)
		return nil, insertErr
	}
	// indexJson, _ := json.Marshal(conversation.ToEs())
	indexJson, _ := json.Marshal(models.NewConversationIndex(conversation))

	constants.ElasticsearchClient.Index(constants.ConversationIndex, bytes.NewReader(indexJson))

	pbConversation := models.NewConversationPb(conversation)

	return &pbConversation, nil
}

func SearchConversations(req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	query := "FROM " + constants.ConversationIndex +
		" | WHERE name LIKE %" + req.GetTerm() + "%" +
		" | KEEP id"

	sefeQuery, _ := json.Marshal(query)
	esqlResp, esqlErr := constants.ElasticsearchClient.SQL.Query(
		bytes.NewReader([]byte(fmt.Sprintf(`{"query": %s}`, sefeQuery))))

	if esqlErr != nil || esqlResp.IsError() {
		return nil, status.New(codes.Internal, "internal service error").Err()
	}
	defer esqlResp.Body.Close()
	// TODO: Map esql ids to cassandra records

	rows, err := models.ConversationRepository.List()
	if err != nil {
		log.Printf("Error fetching conversations: %v", err)
		return nil, err
	}

	convs, ok := rows.([]*models.ConversationEntity)
	if !ok {
		log.Printf("Type assertion failed for rows")
		return nil, fmt.Errorf("unexpected type for rows")
	}

	var conversations []*pb.Conversation
	for _, conv := range convs {
		conversation := models.NewConversationPb(*conv)
		conversations = append(conversations, &conversation)
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
