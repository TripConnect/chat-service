package rpc

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/TripConnect/chat-service/common"
	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/models"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/sortorder"
	"github.com/gocql/gocql"
	pb "github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.Conversation, error) {
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

func (s *Server) FindConversation(ctx context.Context, req *pb.FindConversationRequest) (*pb.Conversation, error) {
	conversation, err := models.ConversationRepository.Get(req.GetConversationId())
	if err != nil {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	pbConversation := models.NewConversationPb(*conversation.(*models.ConversationEntity))
	return &pbConversation, nil
}

func (s *Server) SearchConversations(ctx context.Context, req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	// TODO: Find joined conversation only
	esQuery := esdsl.NewBoolQuery().
		Must(esdsl.NewMatchPhraseQuery("type", strconv.Itoa(int(req.GetType().Number()))))

	if len(req.GetTerm()) > 0 {
		searchTerm := req.GetTerm()
		esQuery.
			Must(esdsl.NewWildcardQuery("name", searchTerm))
	}

	esResp, esErr := consts.ElasticsearchClient.Search().
		Index(consts.ConversationIndex).
		Query(esQuery).
		Sort(esdsl.NewSortOptions().AddSortOption("created_at", esdsl.NewFieldSort(sortorder.Desc))).
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
