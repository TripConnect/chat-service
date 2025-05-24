package common

import (
	"encoding/json"
	"fmt"
	"strings"

	constants "github.com/TripConnect/chat-service/src/consts"
)

func SearchWithElastic[T any](index string, query string) ([]T, error) {
	esResp, esErr := constants.ElasticsearchClient.Search(
		constants.ElasticsearchClient.Search.WithIndex(index),
		constants.ElasticsearchClient.Search.WithBody(strings.NewReader(query)))

	defer esResp.Body.Close()

	if esErr != nil {
		return nil, esErr
	}

	if esResp.IsError() {
		return nil, fmt.Errorf("elasticsearch response error")
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(esResp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("error decoding ES response: %w", err)
	}

	hitsSection, ok := raw["hits"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing hits section in response")
	}

	hitsArray, ok := hitsSection["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("hits is not an array")
	}

	var results []T
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		source, ok := hitMap["_source"]
		if !ok {
			continue
		}

		sourceBytes, err := json.Marshal(source)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal source: %w", err)
		}

		var t T
		if err := json.Unmarshal(sourceBytes, &t); err != nil {
			return nil, fmt.Errorf("failed to unmarshal into struct: %w", err)
		}

		results = append(results, t)
	}

	return results, nil
}
