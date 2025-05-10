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
	"google.golang.org/protobuf/types/known/timestamppb"
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

		conversation, _ := models.ConversationRepository.Get(conversationId)
		if conversation != nil {
			return nil, status.New(codes.InvalidArgument, "private conversation exist").Err()
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

	result := &pb.Conversation{
		Id:        conversation.Id,
		Type:      pb.ConversationType(conversation.Type),
		Name:      req.GetName(),
		CreatedAt: timestamppb.New(conversation.CreatedAt),
	}

	return result, nil
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
		conversations = append(conversations, &pb.Conversation{
			Id:        conv.Id,
			Name:      conv.Name,
			Type:      pb.ConversationType(conv.Type),
			CreatedAt: timestamppb.New(conv.CreatedAt),
		})
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
