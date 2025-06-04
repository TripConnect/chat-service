package common

import "encoding/json"

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
