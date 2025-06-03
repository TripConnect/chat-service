package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	common "github.com/TripConnect/chat-service/src/common"
	constants "github.com/TripConnect/chat-service/src/consts"
	"github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/gocql/gocql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateChatMessage(req *pb.CreateChatMessageRequest) (*pb.ChatMessage, error) {
	fromUserId, fromUserIdErr := gocql.ParseUUID(req.FromUserId)

	if fromUserIdErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid fromUserId")
	}

	chatMessage := models.ChatMessageEntity{
		Id:             gocql.MustRandomUUID(),
		ConversationId: req.GetConversationId(),
		FromUserId:     fromUserId,
		Content:        req.GetContent(),
		CreatedAt:      time.Now(),
	}

	if insertError := models.ChatMessageRepository.Insert(chatMessage); insertError != nil {
		fmt.Printf("failed to create chat message %v", insertError)
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	encodedIndex, _ := json.Marshal(models.NewChatMessageIndex(chatMessage))
	constants.ElasticsearchClient.Index(constants.ChatMessageIndex, bytes.NewReader(encodedIndex))

	chatMessagePb := models.NewChatMessagePb(chatMessage)

	return &chatMessagePb, nil
}

func GetChatMessages(req *pb.GetChatMessagesRequest) (*pb.ChatMessages, error) {
	// FIXME: handle for all nill range
	var gt, lt int64
	if before := req.GetBefore(); before != nil {
		lt = req.GetAfter().AsTime().UnixMilli()
	}
	if after := req.GetAfter(); after != nil {
		gt = req.GetAfter().AsTime().UnixMilli()
	}

	query := fmt.Sprintf(
		`{
			"from": 0,
			"size": %d,
			"query": {
				"bool": {
					"must": [
						{
							"match_phrase": {
								"conversation_id": "%s"
							}
						},
						{
						"range": {
							"created_at": {
								"gt": "%d",
								"lt": "%d"
							}
						}
					]
				}
			},
			"sort": [
				{
					"created_at": {
						"order": "desc",
						"unmapped_type": "long"
					}
				}
			]
		}`, req.GetPageSize(), req.GetConversationId(), gt, lt,
	)

	if docs, err := common.SearchWithElastic[models.ChatMessageDocument](constants.ConversationIndex, query); err != nil {
		fmt.Printf("error while SearchWithElastic %v", err)
		return nil, status.Error(codes.Internal, codes.Internal.String())
	} else {
		var pbMessages []*pb.ChatMessage
		for _, doc := range docs {
			if rawEntity, err := models.ChatMessageRepository.Get(doc.Id); err != nil {
				entity := models.NewChatMessagePb(rawEntity.(models.ChatMessageEntity))
				pbMessages = append(pbMessages, &entity)
			}
		}
		result := &pb.ChatMessages{Messages: pbMessages}
		return result, nil
	}
}
