package common

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/TripConnect/chat-service/src/consts"
)

func ConvertStruct[S any, D any](src *S) (D, error) {
	var dst D

	data, err := json.Marshal(src)
	if err != nil {
		return dst, err
	}

	err = json.Unmarshal(data, &dst)
	if err != nil {
		return dst, err
	}

	return dst, nil
}

func GetCombinedId(ids []string) string {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})

	return strings.Join(ids, consts.ElasticsearchSeparator)
}
