package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/TripConnect/chat-service/common"
	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/models"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/gocql/gocql"
	pb "github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateChatMessage(ctx context.Context, req *pb.CreateChatMessageRequest) (*pb.ChatMessage, error) {
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

	chatMessageDoc := models.NewChatMessageDoc(chatMessage)
	consts.ElasticsearchClient.
		Index(consts.ChatMessageIndex).
		Id(chatMessageDoc.Id.String()).
		Request(&chatMessageDoc).
		Do(ctx)

	chatMessagePb := models.NewChatMessagePb(chatMessage)

	return &chatMessagePb, nil
}

func (s *Server) GetChatMessages(ctx context.Context, req *pb.GetChatMessagesRequest) (*pb.ChatMessages, error) {
	before := types.Float64(req.GetBefore().AsTime().UnixMilli())
	after := types.Float64(req.GetAfter().AsTime().UnixMilli())

	esQuery := esdsl.NewBoolQuery().
		Must(esdsl.NewMatchPhraseQuery("conversation_id", req.GetConversationId())).
		Filter(esdsl.NewNumberRangeQuery("created_at").Gt(after).Lt(before))

	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		Size(int(req.GetPageSize())).
		Do(ctx)

	if err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	docs := common.GetResponseDocs[models.ChatMessageDocument](esResp)

	var pbMessages []*pb.ChatMessage
	for _, doc := range docs {
		if message, err := models.ChatMessageRepository.Get(doc.Id); err == nil {
			pbMessage := models.NewChatMessagePb(message.(models.ChatMessageEntity))
			pbMessages = append(pbMessages, &pbMessage)
		}
	}

	result := &pb.ChatMessages{Messages: pbMessages}
	return result, nil
}

func (s *Server) SearchChatMessages(ctx context.Context, req *pb.SearchChatMessagesRequest) (*pb.ChatMessages, error) {
	esQuery := esdsl.NewBoolQuery().
		Must(esdsl.NewWildcardQuery("content", req.GetTerm()))

	if req.GetConversationId() != "" {
		esQuery.
			Must(esdsl.NewMatchPhraseQuery("conversation_id", req.GetConversationId()))
	}

	// TODO: Consider apply cursor-based pagination
	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		From(int(req.GetPageNumber()) * int(req.GetPageSize())).
		Size(int(req.GetPageSize())).
		Do(ctx)

	if err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	docs := common.GetResponseDocs[models.ChatMessageDocument](esResp)

	var pbMessages []*pb.ChatMessage
	for _, doc := range docs {
		if message, err := models.ChatMessageRepository.Get(doc.Id); err == nil {
			pbMessage := models.NewChatMessagePb(message.(models.ChatMessageEntity))
			pbMessages = append(pbMessages, &pbMessage)
		}
	}

	result := &pb.ChatMessages{Messages: pbMessages}
	return result, nil
}
