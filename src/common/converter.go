package common

import (
	"encoding/json"
	"fmt"
)

func ConvertESToStruct(document map[string]interface{}, target interface{}) error {
	jsonData, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	err = json.Unmarshal(jsonData, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal into struct: %w", err)
	}

	return nil
}
