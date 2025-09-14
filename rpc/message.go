package rpc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/models"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esdsl"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/sortorder"
	"github.com/gocql/gocql"
	"github.com/tripconnect/go-common-utils/common"
	"github.com/tripconnect/go-common-utils/helper"
	pb "github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) CreateChatMessage(ctx context.Context, req *pb.CreateChatMessageRequest) (*pb.CreateChatMessageAck, error) {
	fromUserId, fromUserIdErr := gocql.ParseUUID(req.FromUserId)
	convId, convIdErr := gocql.ParseUUID(req.ConversationId)

	if fromUserIdErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid fromUserId")
	}

	if convIdErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid conversationId")
	}

	chatMessage := &models.KafkaPendingMessage{
		CorrelationId:  gocql.MustRandomUUID().String(),
		ConversationId: convId,
		FromUserId:     fromUserId,
		Content:        req.GetContent(),
		SentTime:       time.Now(),
	}

	pendingTopic, _ := helper.ReadConfig[string]("kafka.topic.chatting-sys-internal-pending-queue")
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
	var musts []types.QueryVariant = []types.QueryVariant{
		esdsl.NewMatchPhraseQuery("conversation_id", req.GetConversationId()),
	}

	if req.GetBefore() != nil {
		before := types.Float64(req.GetBefore().AsTime().UnixMilli())
		musts = append(musts, esdsl.NewNumberRangeQuery("sent_time").Lt(before))
	}

	if req.GetAfter() != nil {
		after := types.Float64(req.GetAfter().AsTime().UnixMilli())
		musts = append(musts, esdsl.NewNumberRangeQuery("sent_time").Gt(after))
	}

	esQuery := esdsl.NewBoolQuery().
		Must(musts...)

	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		Sort(esdsl.NewSortOptions().AddSortOption("sent_time", esdsl.NewFieldSort(sortorder.Desc))).
		Size(int(req.GetLimit())).
		Do(ctx)

	if err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	docs := common.GetResponseDocs[models.ChatMessageDocument](esResp)

	pbMessages := make([]*pb.ChatMessage, len(docs))
	var wg sync.WaitGroup
	wg.Add(len(docs))

	for i, doc := range docs {
		go func(i int, docId gocql.UUID) {
			defer wg.Done()
			if message, err := models.ChatMessageRepository.Get(docId); err == nil {
				pbMsg := models.NewChatMessagePb(*message.(*models.ChatMessageEntity))
				pbMessages[i] = &pbMsg
			} else {
				fmt.Printf("Failed to get message for id %q: %v\n", docId, err)
			}
		}(i, doc.Id)
	}

	wg.Wait()

	result := &pb.ChatMessages{Messages: pbMessages}
	return result, nil
}

func (s *Server) SearchChatMessages(ctx context.Context, req *pb.SearchChatMessagesRequest) (*pb.ChatMessages, error) {
	var musts []types.QueryVariant = []types.QueryVariant{
		esdsl.NewWildcardQuery("content", req.GetTerm()),
	}

	if req.GetConversationId() != "" {
		musts = append(musts, esdsl.NewMatchPhraseQuery("conversation_id", req.GetConversationId()))
	}

	if req.GetBefore() != nil {
		before := types.Float64(req.GetBefore().AsTime().UnixMilli())
		musts = append(musts, esdsl.NewNumberRangeQuery("sent_time").Lt(before))
	}

	if req.GetAfter() != nil {
		after := types.Float64(req.GetAfter().AsTime().UnixMilli())
		musts = append(musts, esdsl.NewNumberRangeQuery("sent_time").Gt(after))
	}

	esQuery := esdsl.NewBoolQuery().
		Must(musts...)

	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		Sort(esdsl.NewSortOptions().AddSortOption("sent_time", esdsl.NewFieldSort(sortorder.Desc))).
		Size(int(req.GetLimit())).
		Do(ctx)

	if err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	docs := common.GetResponseDocs[models.ChatMessageDocument](esResp)

	var pbMessages []*pb.ChatMessage
	for _, doc := range docs {
		if message, err := models.ChatMessageRepository.Get(doc.Id); err == nil {
			pbMessage := models.NewChatMessagePb(*message.(*models.ChatMessageEntity))
			pbMessages = append(pbMessages, &pbMessage)
		}
	}

	result := &pb.ChatMessages{Messages: pbMessages}
	return result, nil
}
