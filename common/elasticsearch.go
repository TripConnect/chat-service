package common

import (
	"encoding/json"
	"log"

	"github.com/elastic/go-elasticsearch/v9/typedapi/core/search"
)

func GetResponseDocs[T any](resp *search.Response) []T {
	var docs []T
	for _, hit := range resp.Hits.Hits {
		var doc T
		if err := json.Unmarshal(hit.Source_, &doc); err != nil {
			log.Printf("error parsing hit: %v", err)
			continue
		}
		docs = append(docs, doc)
	}
	return docs
}
