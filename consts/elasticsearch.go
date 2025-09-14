package consts

import (
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/tripconnect/go-common-utils/helper"
)

const (
	ConversationIndex      = "ks_chat_conversations"
	ChatMessageIndex       = "ks_chat_messages"
	ParticipantIndex       = "ks_chat_participant"
	ElasticsearchSeparator = "|"
)

var ElasticsearchClient *elasticsearch.TypedClient

func init() {
	host, hostErr := helper.ReadConfig[string]("database.elasticsearch.host")

	if hostErr != nil {
		log.Fatalf("failed to load elasticsearch config")
	}

	var err error
	ElasticsearchClient, err = elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{fmt.Sprintf("http://%s:9200", host)},
	})

	if err != nil {
		log.Fatalf("Error creating the Elasticsearch client: %s", err)
	}
}
