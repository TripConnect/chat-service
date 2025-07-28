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

func getConversationMembers(
	ctx context.Context,
	conversationId gocql.UUID, status models.ParticipantStatus,
	pagerNumber int, pageSize int) ([]models.ParticipantEntity, error) {
	esQuery := esdsl.NewBoolQuery().
		Must(esdsl.NewMatchPhraseQuery("conversation_id", conversationId.String())).
		Must(esdsl.NewMatchPhraseQuery("status", strconv.Itoa(int(status))))

	esResp, esErr := consts.ElasticsearchClient.Search().
		Index(consts.ParticipantIndex).
		Query(esQuery).
		Sort(esdsl.NewSortOptions().AddSortOption("created_at", esdsl.NewFieldSort(sortorder.Desc))).
		From(pagerNumber * pageSize).
		Size(pageSize).
		Do(ctx)

	if esErr != nil {
		return nil, esErr
	}

	participantDocs := common.GetResponseDocs[models.ParticipantDocument](esResp)

	participants := []models.ParticipantEntity{}
	for _, doc := range participantDocs {
		pk := map[string]interface{}{
			"conversation_id": conversationId,
			"user_id":         doc.UserId,
			"status":          int(status),
		}
		if participant, err := models.ParticipantRepository.Get(pk); err == nil {
			participants = append(participants, participant.(models.ParticipantEntity))
		}
	}

	return participants, nil
}

func (s *Server) CreateConversation(ctx context.Context, req *pb.CreateConversationRequest) (*pb.Conversation, error) {
	var conversationId gocql.UUID
	var ownerId gocql.UUID

	if req.GetType() == pb.ConversationType_PRIVATE {
		conversationId, _ = gocql.UUIDFromBytes(common.BuildUUID(req.GetMemberIds()...).Bytes())
		ownerId, _ = gocql.ParseUUID("11111111-1111-1111-1111-111111111111")
	} else {
		if parsedOwnerId, ownerError := gocql.ParseUUID(req.GetOwnerId()); ownerError != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid ownerId")
		} else {
			conversationId = gocql.MustRandomUUID()
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

	conversationDoc := models.NewConversationDoc(conversation, req.GetMemberIds())
	consts.ElasticsearchClient.
		Index(consts.ConversationIndex).
		Id(conversationDoc.Id.String()).
		Request(&conversationDoc).
		Do(ctx)

	for _, participantId := range req.GetMemberIds() {
		if userId, err := gocql.ParseUUID(participantId); err == nil {
			participant := models.ParticipantEntity{
				ConversationId: conversation.Id,
				UserId:         userId,
				NickName:       "",
				Status:         int(models.Joined),
				CreatedAt:      time.Now(),
			}
			models.ParticipantRepository.Insert(participant)
			participantDoc := models.NewParticipantDoc(participant, req.GetMemberIds())
			consts.ElasticsearchClient.
				Index(consts.ParticipantIndex).
				Request(&participantDoc).
				Do(ctx)
		}
	}

	// TODO: Adding to proto params
	pbJoinedMembers, err := getConversationMembers(ctx, conversationId, models.Joined, 0, 50)
	if err != nil {
		fmt.Printf("cannot get conversation memebers %s %v", conversationId, err)
		pbJoinedMembers = []models.ParticipantEntity{}
	}

	pbConversation := models.NewConversationPb(conversation, pbJoinedMembers)

	return &pbConversation, nil
}

func (s *Server) FindConversation(ctx context.Context, req *pb.FindConversationRequest) (*pb.Conversation, error) {
	conversation, err := models.ConversationRepository.Get(req.GetConversationId())
	if err != nil {
		return nil, status.Error(codes.NotFound, codes.NotFound.String())
	}

	// TODO: Move pagination of member to proto file
	pbJoinedMembers, err := getConversationMembers(ctx, conversation.(*models.ConversationEntity).Id, models.Joined, 0, 50)
	if err != nil {
		fmt.Printf("cannot get conversation memebers %s %v", req.GetConversationId(), err)
		pbJoinedMembers = []models.ParticipantEntity{}
	}
	pbConversation := models.NewConversationPb(*conversation.(*models.ConversationEntity), pbJoinedMembers)

	return &pbConversation, nil
}

func (s *Server) SearchConversations(ctx context.Context, req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	esQuery := esdsl.NewBoolQuery().
		Must(
			esdsl.NewMatchPhraseQuery("member_ids", req.GetUserId()),
			esdsl.NewMatchPhraseQuery("type", strconv.Itoa(int(req.GetType().Number()))),
		)

	if len(req.GetTerm()) > 0 {
		searchTerm := req.GetTerm()
		esQuery.Must(esdsl.NewWildcardQuery("name", searchTerm))
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
		pbJoinedMembers, err := getConversationMembers(ctx, conv.Id, models.Joined, 0, 50)
		if err != nil {
			fmt.Printf("cannot get conversation memebers %s %v", conv.Id, err)
			pbJoinedMembers = []models.ParticipantEntity{}
		}

		conversation := models.NewConversationPb(*conv, pbJoinedMembers)
		conversations = append(conversations, &conversation)
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
