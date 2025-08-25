package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"tinypng-cli/internal/config"
)

type TinyPNGWebClient struct {
	Client *http.Client
}

var tinypngWebHost = "https://tinypng.com"
var webClient *TinyPNGWebClient

func GetTinyPNGWebClient() *TinyPNGWebClient {
	if webClient != nil {
		return webClient
	}
	webClient = &TinyPNGWebClient{
		Client: &http.Client{Timeout: time.Second * time.Duration(config.C.Timeout)},
	}
	return webClient
}

type WebUploadResult struct {
	Key  string `json:"key"`
	Url  string `json:"url"`
	Size int64  `json:"size"`
}

type WebDownloadResult struct {
	Key    string `json:"key"`
	Url    string `json:"url"`
	Size   int64  `json:"size"`
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func (c *TinyPNGWebClient) WebCompressFromFile(file string) (*WebDownloadResult, error) {
	body, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	fileTypeBuffer := make([]byte, 512)
	// a copy of first 512 bytes
	_, err = body.ReadAt(fileTypeBuffer, 0)
	if err != nil && err != io.EOF {
		panic(err)
	}
	mimeType := http.DetectContentType(fileTypeBuffer)

	// upload file
	req, _ := http.NewRequest(http.MethodPost, tinypngWebHost+"/backend/opt/store", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := GetTinyPNGWebClient().Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(resp.Status)
	}
	var result WebUploadResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	// compress post
	params := map[string]any{
		"key":          result.Key,
		"originalSize": result.Size,
		"originalType": mimeType,
	}
	b, _ := json.Marshal(params)
	log.Println(string(b))
	processReq, _ := http.NewRequest(http.MethodPost, tinypngWebHost+"/backend/opt/process", bytes.NewBuffer(b))
	processReq.Header.Set("Content-Type", "application/json")
	processResp, err := GetTinyPNGWebClient().Client.Do(processReq)
	if err != nil {
		return nil, err
	}
	defer processResp.Body.Close()
	if processResp.StatusCode != http.StatusCreated {
		return nil, errors.New(processResp.Status)
	}
	var downloadResult WebDownloadResult
	err = json.NewDecoder(processResp.Body).Decode(&downloadResult)
	if err != nil {
		return nil, err
	}
	log.Printf("downlaod url: %s\n", downloadResult.Url)

	return &downloadResult, nil
}
