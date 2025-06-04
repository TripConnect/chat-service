package services

import (
	"context"
	"fmt"
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

func CreateChatMessage(ctx context.Context, req *pb.CreateChatMessageRequest) (*pb.ChatMessage, error) {
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

func GetChatMessages(ctx context.Context, req *pb.GetChatMessagesRequest) (*pb.ChatMessages, error) {
	before := req.GetBefore().AsTime().String()
	after := req.GetAfter().AsTime().String()

	esQuery := &types.Query{
		Bool: &types.BoolQuery{
			Must: []types.Query{
				{MatchPhrase: map[string]types.MatchPhraseQuery{
					"conversation_id": {Query: req.GetConversationId()},
				}},
			},
			Filter: []types.Query{{
				Range: map[string]types.RangeQuery{
					"created_at": &types.DateRangeQuery{
						Gt: &after,
						Lt: &before,
					},
				},
			}},
		},
	}

	esResp, err := consts.ElasticsearchClient.Search().
		Index(consts.ChatMessageIndex).
		Query(esQuery).
		Size(int(req.GetPageSize())).
		Do(context.Background())
	if err != nil {
		return nil, status.Error(codes.Internal, codes.Internal.String())
	}

	docs := common.GetResponseDocs[models.ChatMessageDocument](esResp)

	var pbMessages []*pb.ChatMessage
	for _, doc := range docs {
		// FIXME: Need to find cass by id, not map directly from doc to cass
		pbMessage, err := common.ConvertStruct[models.ChatMessageDocument, pb.ChatMessage](&doc)
		if err != nil {
			continue
		}
		pbMessages = append(pbMessages, &pbMessage)
	}

	result := &pb.ChatMessages{Messages: pbMessages}
	return result, nil
}
