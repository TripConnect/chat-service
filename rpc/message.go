package rpc

import (
	"context"
	"log"

	"github.com/TripConnect/chat-service/common"
	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/helpers"
	"github.com/TripConnect/chat-service/models"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/gocql/gocql"
	pb "github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateChatMessage(ctx context.Context, req *pb.CreateChatMessageRequest) (*pb.CreateChatMessageAck, error) {
	fromUserId, fromUserIdErr := gocql.ParseUUID(req.FromUserId)

	if fromUserIdErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid fromUserId")
	}

	chatMessage := &models.KafkaPendingMessage{
		CorrelationId:  gocql.MustRandomUUID().String(),
		ConversationId: req.GetConversationId(),
		FromUserId:     fromUserId,
		Content:        req.GetContent(),
	}

	pendingTopic, _ := helpers.ReadConfig[string]("kafka.topic.chatting-sys-internal-pending-queue")
	if err := common.Publish(ctx, pendingTopic, chatMessage); err != nil {
		log.Printf("Create chat message failed %s", err.Error())
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	chatMessagePb := &pb.CreateChatMessageAck{
		CorrelationId: chatMessage.CorrelationId,
	}

	return chatMessagePb, nil
}

func (s *Server) GetChatMessages(ctx context.Context, req *pb.GetChatMessagesRequest) (*pb.ChatMessages, error) {

	esQuery := esdsl.NewBoolQuery().
		Must(esdsl.NewMatchPhraseQuery("conversation_id", req.GetConversationId()))

	if req.GetBefore() != nil {
		before := types.Float64(req.GetBefore().AsTime().UnixMilli())
		esQuery.Must(esdsl.NewNumberRangeQuery("created_at").Lt(before))
	}

	if req.GetAfter() != nil {
		after := types.Float64(req.GetAfter().AsTime().UnixMilli())
		esQuery.Must(esdsl.NewNumberRangeQuery("created_at").Gt(after))
	}

	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		Size(int(req.GetLimit())).
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

	if req.GetBefore() != nil {
		esQuery.
			Filter(esdsl.NewNumberRangeQuery("created_at").
				Gt(types.Float64(req.GetAfter().AsTime().UnixMilli())))
	}

	if req.GetAfter() != nil {
		esQuery.
			Filter(esdsl.NewNumberRangeQuery("created_at").
				Gt(types.Float64(req.GetAfter().AsTime().UnixMilli())))
	}

	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		Size(int(req.GetLimit())).
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
