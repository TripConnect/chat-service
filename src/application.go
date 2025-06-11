package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/TripConnect/chat-service/src/consts"
	"github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	service "github.com/TripConnect/chat-service/src/services"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 31073, "The server port")
)

type server struct {
	pb.UnimplementedChatServiceServer
}

func (s *server) CreateConversation(ctx context.Context, in *pb.CreateConversationRequest) (*pb.Conversation, error) {
	conversation, err := service.CreateConversation(ctx, in)
	return conversation, err
}

func (s *server) FindConversation(_ context.Context, in *pb.FindConversationRequest) (*pb.Conversation, error) {
	conversation, err := service.FindConversation(in)
	return conversation, err
}

func (s *server) SearchConversations(_ context.Context, in *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	conversations, err := service.SearchConversations(in)
	return conversations, err
}

func (s *server) CreateChatMessage(ctx context.Context, in *pb.CreateChatMessageRequest) (*pb.ChatMessage, error) {
	chatMessage, err := service.CreateChatMessage(ctx, in)
	return chatMessage, err
}

func (s *server) GetChatMessages(ctx context.Context, in *pb.GetChatMessagesRequest) (*pb.ChatMessages, error) {
	chatMessages, err := service.GetChatMessages(ctx, in)
	return chatMessages, err
}

func initCassandra() {
	// Authentication
	cluster := gocql.NewCluster("localhost")
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: "cassandra",
		Password: "tripconnect3107",
	}
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to connect to Cassandra: %v", err)
	}
	gocqltable.SetDefaultSession(session)

	// Create keyspace
	keyspace := gocqltable.NewKeyspace(consts.KeySpace)
	_ = keyspace.Create(map[string]interface{}{
		"class":              "SimpleStrategy",
		"replication_factor": 1,
	}, true)

	// Create tables
	models.ConversationRepository.TableInterface.Create()
	models.ChatMessageRepository.TableInterface.Create()
	models.ParticipantRepository.TableInterface.Create()
}

func initElasticsearch() {
	ctx := context.Background()
	consts.ElasticsearchClient.Indices.Create(consts.ConversationIndex).Do(ctx)
	consts.ElasticsearchClient.Indices.Create(consts.ChatMessageIndex).Do(ctx)
}

func init() {
	// Cassandra initalization
	initCassandra()
	// Elastic search initalization
	initElasticsearch()
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
