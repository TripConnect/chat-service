package consts

import (
	"log"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
)

const (
	ConversationIndex      = "ks_chat_conversations"
	ChatMessageIndex       = "ks_chat_messages"
	ElasticsearchSeparator = "|"
)

var ElasticsearchClient *elasticsearch.TypedClient

func init() {
	var err error
	ElasticsearchClient, err = elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	})
	if err != nil {
		log.Fatalf("Error creating the Elasticsearch client: %s", err)
	}
}
