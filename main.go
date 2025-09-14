package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/kafka/consumers"
	"github.com/TripConnect/chat-service/models"
	"github.com/TripConnect/chat-service/rpc"
	"github.com/gocql/gocql"
	"github.com/kristoiv/gocqltable"
	"github.com/tripconnect/go-common-utils/common"
	"github.com/tripconnect/go-common-utils/helper"
	"github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc"
)

func initCassandra() {
	host, hostErr := helper.ReadConfig[string]("database.cassandra.host")
	username, usernameErr := helper.ReadConfig[string]("database.cassandra.username")
	password, passwordErr := helper.ReadConfig[string]("database.cassandra.password")

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
	common.ElasticsearchClient.Indices.
		Create(consts.ConversationIndex).
		Mappings(models.ConversationDocumentMappings).
		Do(ctx)
	common.ElasticsearchClient.Indices.
		Create(consts.ChatMessageIndex).
		Mappings(models.ChatMessageDocumentMappings).
		Do(ctx)
	common.ElasticsearchClient.Indices.
		Create(consts.ParticipantIndex).
		Mappings(models.ParticipantDocumentMappings).
		Do(ctx)
}

func initKafka() {
	go consumers.ListenPendingMessageQueue()
}

func init() {
	// Cassandra initalization
	initCassandra()
	// Elasticsearch initalization
	initElasticsearch()
	// Kafka initalization
	initKafka()
}

func main() {
	port, err := helper.ReadConfig[int]("server.port")

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
