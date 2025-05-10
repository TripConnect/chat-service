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

		iconversation, _ := models.ConversationRepository.Get(conversationId)
		if iconversation != nil {
			pbConversation := iconversation.(*models.ConversationEntity).ToPb()
			return &pbConversation, nil
		}
	} else {
		var ownerError error

		conversationId = gocql.MustRandomUUID().String()
		ownerId, ownerError = gocql.ParseUUID(req.GetOwnerId())
		if ownerError != nil {
			return nil, status.New(codes.InvalidArgument, "invalid owner id").Err()
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
	indexJson, _ := json.Marshal(conversation.ToEs())
	constants.ElasticsearchClient.Index(constants.ConversationIndex, bytes.NewReader(indexJson))

	pbConversation := conversation.ToPb()

	return &pbConversation, nil
}

func SearchConversations(req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
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
		conversation := conv.ToPb()
		conversations = append(conversations, &conversation)
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
