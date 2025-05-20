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
	conversationId := gocql.MustRandomUUID()
	aliasId := conversationId.String()
	var ownerId gocql.UUID

	if req.GetType() == pb.ConversationType_PRIVATE {
		memberIds := req.GetMemberIds()
		sort.Slice(memberIds, func(i, j int) bool {
			return memberIds[i] > memberIds[j]
		})

		aliasId = strings.Join(memberIds, constants.ElasticsearchSeparator)
		ownerId, _ = gocql.ParseUUID("11111111-1111-1111-1111-111111111111")

		existConversation, _ := models.ConversationRepository.Get(aliasId)
		if existConversation != nil {
			conversationPb := models.NewConversationPb(*existConversation.(*models.ConversationEntity))
			return &conversationPb, nil
		}
	} else {
		var ownerError error

		ownerId, ownerError = gocql.ParseUUID(req.GetOwnerId())
		if ownerError != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid ownerId")
		}
	}

	conversation := models.ConversationEntity{
		Id:        conversationId,
		AliasId:   aliasId,
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

	encodedIndex, _ := json.Marshal(models.NewConversationIndex(conversation))
	constants.ElasticsearchClient.Index(constants.ConversationIndex, bytes.NewReader(encodedIndex))

	pbConversation := models.NewConversationPb(conversation)

	return &pbConversation, nil
}

func FindConversation(req *pb.FindConversationRequest) (*pb.Conversation, error) {
	conversation, err := models.ConversationRepository.Get(req.ConversationId)
	if err != nil {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	pbConversation := models.NewConversationPb(*conversation.(*models.ConversationEntity))
	return &pbConversation, nil
}

func SearchConversations(req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	query := fmt.Sprintf(
		`{
			"from": %d,
			"size": %d,
			"query": {
				"match_all": {}
			},
			"sort": [
				{
					"created_at": {
						"order": "desc"
					}
				}
			]
		}`, req.GetPageNumber()*req.GetPageSize(), req.GetPageSize(),
	)
	if len(req.GetTerm()) > 0 {
		query = fmt.Sprintf(
			`{
				"from": %d,
				"size": %d,
				"query": {
					"bool": {
						"must": [
							{
								"wildcard": {
									"name.keyword": "*%s*"
								}
							}
						]
					}
				},
				"sort": [
					{
						"created_at": {
							"order": "desc"
						}
					}
				]
			}`, req.GetPageNumber()*req.GetPageSize(), req.GetPageSize(), req.GetTerm(),
		)
	}

	esResp, esErr := constants.ElasticsearchClient.Search(
		constants.ElasticsearchClient.Search.WithIndex(constants.ConversationIndex),
		constants.ElasticsearchClient.Search.WithBody(strings.NewReader(query)))

	if esErr != nil || esResp.IsError() {
		fmt.Printf("ESQL error %v", esErr)
		return nil, status.Error(codes.Internal, "internal service error")
	}
	defer esResp.Body.Close()

	var r map[string]interface{}

	if err := json.NewDecoder(esResp.Body).Decode(&r); err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	var esConversations []models.ConversationIndex
	hits := r["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"]
		sourceBytes, err := json.Marshal(source)
		if err != nil {
			fmt.Println("failed to encode es response")
			return nil, status.Error(codes.Internal, codes.Internal.String())
		}

		var conv models.ConversationIndex
		if err := json.Unmarshal(sourceBytes, &conv); err != nil {
			fmt.Println("failed to unmarshal decoded es response")
			return nil, status.Error(codes.Internal, codes.Internal.String())
		}
		esConversations = append(esConversations, conv)
	}

	var ids []gocql.UUID
	for _, conv := range esConversations {
		ids = append(ids, conv.Id)
	}

	var convs []*models.ConversationEntity
	for _, id := range ids {
		if entity, err := models.ConversationRepository.Get(id); err == nil {
			convs = append(convs, entity.(*models.ConversationEntity))
		} else {
			fmt.Printf("failed to get conversation entity %s: %v", id, err)
		}
	}

	var conversations []*pb.Conversation
	for _, conv := range convs {
		conversation := models.NewConversationPb(*conv)
		conversations = append(conversations, &conversation)
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
