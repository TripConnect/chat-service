package common

import (
	"sort"
	"strings"

	"github.com/TripConnect/chat-service/consts"
)

func GetCombinedId(ids []string) string {
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})

	return strings.Join(ids, consts.ElasticsearchSeparator)
}
