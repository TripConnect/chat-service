package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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
	host, _ := helper.ReadConfig[string]("database.cassandra.host")
	username, _ := helper.ReadConfig[string]("database.cassandra.username")
	password, _ := helper.ReadConfig[string]("database.cassandra.password")

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

	keyspace := gocqltable.NewKeyspace(consts.KeySpace)
	_ = keyspace.Create(map[string]interface{}{
		"class":              "SimpleStrategy",
		"replication_factor": 1,
	}, true)

	models.ConversationRepository.TableInterface.Create()
	models.ChatMessageRepository.TableInterface.Create()
	models.ParticipantRepository.TableInterface.Create()
}

func initElasticsearch() {
	ctx := context.Background()

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

func initKafka(ctx context.Context) {
	go consumers.ListenPendingMessageQueue(ctx)
}

// ================= CONSUL =================

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

func registerConsul(port int) (*api.Client, string) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = "127.0.0.1:8500"

	client, err := api.NewClient(consulConfig)
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}

	serviceID := "chat-service-" + uuid.NewString()

	fmt.Printf("Consul address: %s\n", getOutboundIP())

	check := &api.AgentServiceCheck{
		GRPC:                           "host.docker.internal:" + strconv.Itoa(port),
		Interval:                       "10s",
		Timeout:                        "5s",
		DeregisterCriticalServiceAfter: "30s",
	}

	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "chat-service",
		Address: getOutboundIP(),
		Port:    port,
		Tags:    []string{"version=1.0", "env=dev"},
		Check:   check,
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}

	log.Printf("Registered to Consul with ID=%s", serviceID)
	return client, serviceID
}

func deregisterConsul(client *api.Client, serviceID string) {
	log.Println("Deregistering from Consul...")
	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		log.Printf("Failed to deregister: %v", err)
	} else {
		log.Println("Deregistered successfully")
	}
}

// ================= MAIN =================

func main() {
	// root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// init infra
	initCassandra()
	initElasticsearch()
	initKafka(ctx)

	port, err := helper.ReadConfig[int]("server.port")
	if err != nil {
		log.Fatalf("failed to load port config %v", err)
	}

	// register consul
	consulClient, serviceID := registerConsul(port)

	// gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := grpc.NewServer()
	protos.RegisterChatServiceServer(server, &rpc.Server{})

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)

	// run gRPC
	go func() {
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := server.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// listen signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutdown signal received")

	// cancel context → stop background jobs
	cancel()

	// graceful stop gRPC
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		log.Println("gRPC stopped gracefully")
	case <-time.After(10 * time.Second):
		log.Println("Force stopping gRPC...")
		server.Stop()
	}

	// deregister consul
	deregisterConsul(consulClient, serviceID)

	log.Println("Shutdown complete")
}
