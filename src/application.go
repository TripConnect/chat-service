package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	constants "github.com/TripConnect/chat-service/src/consts"
	"github.com/TripConnect/chat-service/src/models"
	pb "github.com/TripConnect/chat-service/src/protos/defs"
	service "github.com/TripConnect/chat-service/src/services"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 3107, "The server port")
)

type server struct {
	pb.UnimplementedChatServiceServer
}

func (s *server) CreateConversation(_ context.Context, in *pb.CreateConversationRequest) (*pb.Conversation, error) {
	conversation, err := service.CreateConversation(in)
	return conversation, err
}

func (s *server) SearchConversations(_ context.Context, in *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	conversations, err := service.SearchConversations(in)
	return conversations, err
}

func (s *server) CreateChatMessage(_ context.Context, in *pb.CreateChatMessageRequest) (*pb.ChatMessage, error) {
	chatMessage, err := service.CreateChatMessage(in)
	return chatMessage, err
}

func (s *server) GetChatMessageRequest(_ context.Context, in *pb.CreateChatMessageRequest) (*pb.ChatMessages, error) {
	return nil, nil
}

func cassandraInitialize() {
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
	keyspace := gocqltable.NewKeyspace(constants.KeySpace)
	_ = keyspace.Create(map[string]interface{}{
		"class":              "SimpleStrategy",
		"replication_factor": 1,
	}, true)

	tableMap := map[string]any{
		constants.ConversationTableName: models.ConversationEntity{},
		constants.ChatMessageTableName:  models.ChatMessageEntity{},
	}

	for tableName, tableDes := range tableMap {
		table := keyspace.NewTable(
			tableName,
			[]string{"id"},
			nil,
			tableDes)
		table.Create()
	}

}

func init() {
	// Cassandra initalization
	cassandraInitialize()
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
