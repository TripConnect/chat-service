package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	query := fmt.Sprintf(
		`{
			"from": %d,
			"size": %d,
			"query": {
				"bool": {
					"must": [
						{
							"match_phrase": {
								"conversation_id": "%s"
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
		}`, req.GetPageNumber()*req.GetPageSize(), req.GetPageSize(), req.GetConversationId(),
	)

	esResp, esErr := constants.ElasticsearchClient.Search(
		constants.ElasticsearchClient.Search.WithIndex(constants.ChatMessageIndex),
		constants.ElasticsearchClient.Search.WithBody(strings.NewReader(query)))

	if esErr != nil || esResp.IsError() {
		fmt.Printf("Failed to get chat messages %v", esErr)
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}
	defer esResp.Body.Close()

	var r map[string]interface{}

	if err := json.NewDecoder(esResp.Body).Decode(&r); err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	var esChatMessages []models.ChatMessageIndex
	hits := r["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		sourceBytes, err := json.Marshal(source)
		if err != nil {
			fmt.Println("failed to encode es response")
			return nil, status.Error(codes.Internal, codes.Internal.String())
		}

		var esChatMessage models.ChatMessageIndex
		if err := json.Unmarshal(sourceBytes, &esChatMessage); err != nil {
			fmt.Println("failed to unmarshal decoded es response")
			return nil, status.Error(codes.Internal, codes.Internal.String())
		}
		esChatMessages = append(esChatMessages, esChatMessage)
	}

	var ids []gocql.UUID
	for _, chatMsg := range esChatMessages {
		ids = append(ids, chatMsg.Id)
	}

	var chatMessageEntities []*models.ChatMessageEntity
	for _, id := range ids {
		if entity, err := models.ConversationRepository.Get(id); err == nil {
			chatMessageEntities = append(chatMessageEntities, entity.(*models.ChatMessageEntity))
		} else {
			fmt.Printf("failed to get conversation entity %s: %v", id, err)
		}
	}

	var chatMessages []*pb.ChatMessage
	for _, chatMessageEntity := range chatMessageEntities {
		chatMessage := models.NewChatMessagePb(*chatMessageEntity)
		chatMessages = append(chatMessages, &chatMessage)
	}

	result := &pb.ChatMessages{Messages: chatMessages}
	return result, nil
}
