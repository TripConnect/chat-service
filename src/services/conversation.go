package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/TripConnect/chat-service/src/common"
	"github.com/TripConnect/chat-service/src/consts"
	"github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/gocql/gocql"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.Conversation, error) {
	var conversationId string
	var ownerId gocql.UUID

	if req.GetType() == pb.ConversationType_PRIVATE {
		conversationId = common.GetCombinedId(req.GetMemberIds())
		ownerId, _ = gocql.ParseUUID("11111111-1111-1111-1111-111111111111")
	} else {
		if parsedOwnerId, ownerError := gocql.ParseUUID(req.GetOwnerId()); ownerError != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid ownerId")
		} else {
			conversationId = gocql.MustRandomUUID().String()
			ownerId = parsedOwnerId
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

	conversationDoc := models.NewConversationDoc(conversation)
	consts.ElasticsearchClient.
		Index(consts.ConversationIndex).
		Id(conversationDoc.Id).
		Request(&conversationDoc).
		Do(ctx)

	for _, participantId := range req.GetMemberIds() {
		if userId, err := gocql.ParseUUID(participantId); err == nil {
			participant := models.ParticipantEntity{
				ConversationId: conversation.Id,
				UserId:         userId,
				Status:         int(models.Joined),
			}
			models.ParticipantRepository.Insert(participant)
		}
	}

	pbConversation := models.NewConversationPb(conversation)

	return &pbConversation, nil
}

func FindConversation(req *pb.FindConversationRequest) (*pb.Conversation, error) {
	conversation, err := models.ConversationRepository.Get(req.GetConversationId())
	if err != nil {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	pbConversation := models.NewConversationPb(*conversation.(*models.ConversationEntity))
	return &pbConversation, nil
}

func SearchConversations(ctx context.Context, req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	esQuery := &types.Query{
		Bool: &types.BoolQuery{
			Must: []types.Query{
				{
					Term: map[string]types.TermQuery{
						"type": {
							Value: req.GetType().Number(),
						},
					},
				},
			},
		},
	}

	if len(req.GetTerm()) > 0 {
		searchTerm := req.GetTerm()
		esQuery.Bool.Must = append(esQuery.Bool.Must,
			types.Query{
				Wildcard: map[string]types.WildcardQuery{
					"name": {
						Value: &searchTerm,
					},
				},
			},
		)
	}

	esResp, esErr := consts.ElasticsearchClient.Search().
		Index(consts.ConversationIndex).
		Query(esQuery).
		From(int(req.GetPageNumber() * req.GetPageSize())).
		Size(int(req.GetPageSize())).
		Do(ctx)

	if esErr != nil {
		log.Fatalf("Search failed: %v", esErr)
	}

	esConversations := common.GetResponseDocs[models.ConversationDocument](esResp)

	var ids []string
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
