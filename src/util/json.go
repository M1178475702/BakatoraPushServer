package util

import "encoding/json"

func O2s(obj interface{}) (string, error){
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
