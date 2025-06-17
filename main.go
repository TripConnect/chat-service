package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/models"
	"github.com/TripConnect/chat-service/rpc"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc"
)

var (
	port = flag.Int("port", 31073, "The server port")
)

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

	var server = grpc.NewServer()
	protos.RegisterChatServiceServer(server, &rpc.Server{})

	log.Printf("server listening at %v", lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
