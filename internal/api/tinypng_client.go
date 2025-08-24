package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"tinypng-cli/internal/config"
)

type TinyPNGClient struct {
	Client *http.Client
}

var tinypngAPIHost = "https://api.tinify.com"
var client *TinyPNGClient

func GetTinyPNGClient() *TinyPNGClient {
	if client != nil {
		return client
	}
	client = &TinyPNGClient{
		Client: &http.Client{Timeout: time.Second * 60},
	}
	return client
}

func (client *TinyPNGClient) CompressFromFile(path string) (*CompressResult, error) {
	body, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return sendCompressPost(body)
}

func sendCompressPost(body io.Reader) (*CompressResult, error) {
	req, err := http.NewRequest(http.MethodPost, tinypngAPIHost+"/shrink", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("api", config.GetAPIKey())

	resp, err := GetTinyPNGClient().Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(resp.Status)
	}
	downloadUrl := resp.Header.Get("Location")
	log.Printf("compressed image url: %s\n", downloadUrl)

	var result CompressResult
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	result.DownloadUrl = downloadUrl

	return &result, nil
}

type CompressResult struct {
	DownloadUrl  string
	OriginalFile string         // file or url
	Output       CompressedFile `json:"output"` // upload from url result
	Input        CompressedFile `json:"input"`  // upload from file result
}

type CompressedFile struct {
	Size int    `json:"size"`
	Type string `json:"type"`
}

func (file CompressedFile) Suffix() string {
	if file.Type == "image/jpeg" {
		return "jpg"
	}
	return strings.Split(file.Type, "/")[1]
}

func (client *TinyPNGClient) CompressFromUrl(url string) (*CompressResult, error) {
	params := map[string]any{
		"source": map[string]string{
			"url": url,
		},
	}
	b, _ := json.Marshal(params)

	return sendCompressPost(bytes.NewBuffer(b))
}

func Download(url string, newFile string) error {
	return sendDownload(http.MethodGet, url, newFile, nil)
}

func DownloadWithMetadata(url string, newFile string, metadata []string) error {
	params := map[string]any{
		"preserve": metadata,
	}
	b, _ := json.Marshal(params)
	return sendDownload(http.MethodPost, url, newFile, bytes.NewBuffer(b))
}

func DownloadWithConvert(url string, newFile string, convertTo, convertBG string) error {
	t := ""
	if convertTo == "*" {
		t = "*/*"
	} else {
		t = "image/" + convertTo
	}
	params := map[string]any{
		"convert": map[string]string{
			"type": t,
		},
	}
	if convertBG != "" {
		params["transform"] = map[string]any{
			"background": convertBG,
		}
	}
	b, _ := json.Marshal(params)
	return sendDownload(http.MethodPost, url, newFile, bytes.NewBuffer(b))
}

func DownloadWithResize(url, newFile, resizeMethod string, width, height int) error {
	resize := map[string]any{
		"method": resizeMethod,
	}
	if width > 0 {
		resize["width"] = width
	}
	if height > 0 {
		resize["height"] = height
	}
	params := map[string]any{
		"resize": resize,
	}
	b, _ := json.Marshal(params)
	return sendDownload(http.MethodPost, url, newFile, bytes.NewBuffer(b))
}

func sendDownload(method string, url string, newFile string, body io.Reader) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("api", config.GetAPIKey())

	resp, err := GetTinyPNGClient().Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	showCompressingCount(resp)
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	err = writeFileFromResp(resp, newFile)
	if err != nil {
		return err
	}

	return nil
}

func showCompressingCount(resp *http.Response) {
	count := resp.Header.Get("Compression-Count")
	log.Printf("compressing count: %s\n", count)
}

func writeFileFromResp(resp *http.Response, newFile string) error {
	out, err := os.Create(newFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func IsUrl(url string) bool {
	return strings.HasPrefix(url, "http:") || strings.HasPrefix(url, "https:")
}
