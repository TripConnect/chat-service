package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
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
	conversationId := gocql.MustRandomUUID()
	aliasId := conversationId.String()
	var ownerId gocql.UUID

	if req.GetType() == pb.ConversationType_PRIVATE {
		memberIds := req.GetMemberIds()
		sort.Slice(memberIds, func(i, j int) bool {
			return memberIds[i] > memberIds[j]
		})

		aliasId = strings.Join(memberIds, consts.ElasticsearchSeparator)
		ownerId, _ = gocql.ParseUUID("11111111-1111-1111-1111-111111111111")

		esQuery := &types.Query{
			Bool: &types.BoolQuery{
				Must: []types.Query{
					{MatchPhrase: map[string]types.MatchPhraseQuery{
						"alias_id": {Query: aliasId},
					}},
				},
			},
		}

		esResp, err := consts.ElasticsearchClient.Search().
			Index(consts.ChatMessageIndex).
			Query(esQuery).
			Size(1).
			Sort(). // FIXME: Sort desc by created_at here
			Do(ctx)

		if err != nil {
			return nil, status.Error(codes.Internal, codes.Internal.String())
		}

		if docs := common.GetResponseDocs[models.ChatMessageDocument](esResp); len(docs) > 0 {
			if existConversation, err := models.ConversationRepository.Get(docs[0].Id); err == nil {
				conversationPb := models.NewConversationPb(*existConversation.(*models.ConversationEntity))
				return &conversationPb, nil
			} else {
				fmt.Printf("Error while converting %v", err)
			}
		}

		conversation := models.ConversationEntity{
			Id:        gocql.MustRandomUUID(),
			AliasId:   aliasId,
			OwnerId:   ownerId,
			Name:      "",
			Type:      int(req.GetType()),
			CreatedAt: time.Now(),
		}

		if err := models.ConversationRepository.Insert(conversation); err != nil {
			return nil, status.Error(codes.Internal, codes.Internal.String())
		}

		conversationDoc := models.NewConversationDoc(conversation)
		consts.ElasticsearchClient.
			Index(consts.ConversationIndex).
			Id(conversationDoc.Id.String()).
			Request(&conversationDoc).
			Do(ctx)

		conversationPb := models.NewConversationPb(conversation)
		return &conversationPb, nil
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

	conversationDoc := models.NewConversationDoc(conversation)
	consts.ElasticsearchClient.
		Index(consts.ConversationIndex).
		Id(conversationDoc.Id.String()).
		Request(&conversationDoc).
		Do(ctx)

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

func SearchConversations(req *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	searchTerm := req.GetTerm()

	esQuery := &types.Query{
		Bool: &types.BoolQuery{
			Must: []types.Query{
				{
					Wildcard: map[string]types.WildcardQuery{
						"name": {
							Value: &searchTerm,
						},
					},
				},
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

	esResp, esErr := consts.ElasticsearchClient.Search().
		Index(consts.ConversationIndex).
		Query(esQuery).
		From(int(req.GetPageNumber() * req.GetPageSize())).
		Size(int(req.GetPageSize())).
		Do(context.Background())

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
		conversation := models.NewConversationPb(*conv)
		conversations = append(conversations, &conversation)
	}

	result := &pb.Conversations{Conversations: conversations}
	return result, nil
}
