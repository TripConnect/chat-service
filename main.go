package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/TripConnect/chat-service/consts"
	"github.com/TripConnect/chat-service/kafka/consumers"
	"github.com/TripConnect/chat-service/models"
	"github.com/TripConnect/chat-service/rpc"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"github.com/kristoiv/gocqltable"
	"github.com/tripconnect/go-common-utils/common"
	"github.com/tripconnect/go-common-utils/helper"
	"github.com/tripconnect/go-proto-lib/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
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

func registryConsul(port int) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = "127.0.0.1:8500"

	client, err := api.NewClient(consulConfig)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}

	// -------------------------------
	// Service details (change these!)
	// -------------------------------
	serviceName := "chat-service"
	serviceID := "chat-service-" + uuid.NewString()
	serviceAddress := "127.0.0.1"
	if hostname, err := os.Hostname(); err == nil {
		serviceAddress = hostname
	}

	tags := []string{"version=1.0", "env=dev", "team=backend"}

	check := &api.AgentServiceCheck{
		GRPC:                           fmt.Sprintf("%s:%d", serviceAddress, port),
		Interval:                       "10s",
		Timeout:                        "5s",
		DeregisterCriticalServiceAfter: "30s",
	}

	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: serviceAddress,
		Port:    port,
		Tags:    tags,
		Check:   check,
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}

	log.Printf("Service '%s' (%s) registered successfully on port %d", serviceName, serviceID, port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down... Deregistering from Consul")

	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		log.Printf("Failed to deregister: %v", err)
	} else {
		log.Println("Successfully deregistered")
	}

	os.Exit(0)
}

func main() {
	port, err := helper.ReadConfig[int]("server.port")

	if err != nil {
		log.Fatalf("failed to load port config %v", err)
		return
	}

	go func() {
		registryConsul(port)
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var server = grpc.NewServer()
	protos.RegisterChatServiceServer(server, &rpc.Server{})

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)

	log.Printf("server listening at %v", lis.Addr())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
