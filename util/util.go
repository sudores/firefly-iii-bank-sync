package util

import (
	"encoding/json"
	"io"
	"net/http"
)

func HttpResponseToStruct(resp *http.Response, obj interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return err
	}
	return nil
}

func HttpRequestToStruct(req *http.Request, obj interface{}) error {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return err
	}
	return nil
}
