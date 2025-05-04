package services

import (
	"fmt"
	"log"
	"time"

	models "github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/gocql/gocql"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateConversation(req *pb.CreateConversationRequest) (*pb.Conversation, error) {
	var ownerId gocql.UUID

	if req.GetType().String() != pb.ConversationType_PRIVATE.String() {
		var err error
		ownerId, err = gocql.ParseUUID(req.GetOwnerId())
		if err != nil {
			log.Fatalf("Failed to create conversation with ownerId %s: %v", req.GetOwnerId(), err)
			return nil, err
		}
	}

	conversation := models.ConversationEntity{
		ID:        gocql.MustRandomUUID(),
		Name:      "Foo chat",
		Type:      1,
		OwnerId:   ownerId,
		CreatedAt: time.Now(),
	}

	err := models.ConversationRepository.Insert(conversation)
	if err != nil {
		log.Fatalf("Failed to insert conversation: %v", err)
		return nil, err
	}

	result := &pb.Conversation{
		Id:        conversation.ID.String(),
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
			Id:        conv.ID.String(),
			Name:      conv.Name,
			Type:      pb.ConversationType(conv.Type),
			CreatedAt: timestamppb.New(conv.CreatedAt),
		})
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
