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

var ElasticsearchClient *elasticsearch.Client

func init() {
	var err error
	ElasticsearchClient, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	})
	if err != nil {
		log.Fatalf("Error creating the Elasticsearch client: %s", err)
	}
}
