package services

import (
	"time"

	"github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/gocql/gocql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateChatMessage(req *pb.CreateChatMessageRequest) (*pb.ChatMessage, error) {
	fromUserId, fromUserIdErr := gocql.ParseUUID(req.FromUserId)

	if fromUserIdErr != nil {
		return nil, status.New(codes.InvalidArgument, "invalid fromUserId").Err()
	}

	chatMessage := models.ChatMessageEntity{
		Id:             gocql.MustRandomUUID(),
		ConversationId: req.ConversationId,
		FromUserId:     fromUserId,
		Content:        req.GetContent(),
		CreatedAt:      time.Now(),
	}
	insertError := models.ChatMessageRepository.Insert(chatMessage)
	if insertError != nil {
		return nil, status.New(codes.Internal, "invalid fromUserId").Err()
	}

	chatMessagePb := models.NewChatMessagePb(chatMessage)

	return &chatMessagePb, nil
}
