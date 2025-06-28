package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/helpers"
	"github.com/TripConnect/chat-service/models"
	"github.com/TripConnect/chat-service/rpc"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc"
)

func initCassandra() {
	host, hostErr := helpers.ReadConfig[string]("database.cassandra.host")
	username, usernameErr := helpers.ReadConfig[string]("database.cassandra.username")
	password, passwordErr := helpers.ReadConfig[string]("database.cassandra.password")

	if hostErr != nil || usernameErr != nil || passwordErr != nil {
		log.Fatal("failed to load cassandra config")
	}

	// Authentication
	cluster := gocql.NewCluster(host)
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: username,
		Password: password,
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
	// Create indexes
	consts.ElasticsearchClient.Indices.
		Create(consts.ConversationIndex).
		Mappings(models.ConversationDocumentMappings).
		Do(ctx)
	consts.ElasticsearchClient.Indices.
		Create(consts.ChatMessageIndex).
		Mappings(models.ChatMessageDocumentMappings).
		Do(ctx)
}

func init() {
	// Cassandra initalization
	initCassandra()
	// Elastic search initalization
	initElasticsearch()
}

func main() {
	port, err := helpers.ReadConfig[int]("server.port")

	if err != nil {
		log.Fatalf("failed to load port config %v", err)
		return
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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
