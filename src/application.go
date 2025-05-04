package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/TripConnect/chat-service/src/protos/defs"
	service "github.com/TripConnect/chat-service/src/services"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 3107, "The server port")
)

type server struct {
	pb.UnimplementedChatServiceServer
}

func (s *server) CreateConversation(_ context.Context, in *pb.CreateConversationRequest) (*pb.Conversation, error) {
	conversation := service.CreateConversation(in)
	return conversation, nil
}

func (s *server) SearchConversations(_ context.Context, in *pb.SearchConversationsRequest) (*pb.Conversations, error) {
	conversations := service.SearchConversations(in)
	return conversations, nil
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
