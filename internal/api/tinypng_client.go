package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"tinypng-cli/internal/config"
)

type TinyPNGClient struct {
	Client *http.Client
}

var tinypngAPIHost = "https://api.tinify.com"
var client *TinyPNGClient
var compressedSuffix = "-compressed."

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

	resp, err := client.Client.Do(req)
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

func (result *CompressResult) SaveToLocal(savePath string, metadata []string) error {
	var fullPath string
	if IsUrl(result.OriginalFile) {
		path, err := url.Parse(result.OriginalFile)
		if err != nil {
			return err
		}
		filename := strings.TrimSuffix(filepath.Base(path.Path), filepath.Ext(path.Path)) + compressedSuffix + result.Input.Suffix()
		fullPath = filepath.Join(savePath, filename)

	} else {
		if savePath == "" {
			fullPath = strings.TrimSuffix(result.OriginalFile, filepath.Ext(result.OriginalFile)) + compressedSuffix + result.Input.Suffix()
		} else {
			filename := strings.TrimSuffix(filepath.Base(result.OriginalFile), filepath.Ext(result.OriginalFile)) + compressedSuffix + result.Input.Suffix()
			fullPath = filepath.Join(savePath, filename)
		}
	}

	log.Printf("save to new local file %s\n", fullPath)
	if metadata != nil && len(metadata) > 0 {
		err := downloadWithMetadata(result.DownloadUrl, fullPath, metadata)
		if err != nil {
			return err
		}
	} else {
		err := download(result.DownloadUrl, fullPath)
		if err != nil {
			return err
		}
	}
	return nil
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

func download(url string, newFile string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	err = writeFileFromResp(resp, newFile)
	if err != nil {
		return err
	}
	return nil
}

func downloadWithMetadata(url string, newFile string, metadata []string) error {
	params := map[string]any{
		"preserve": metadata,
	}
	b, _ := json.Marshal(params)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("api", config.GetAPIKey())

	resp, err := client.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New(resp.Status)
	}

	err = writeFileFromResp(resp, newFile)
	if err != nil {
		return err
	}

	return nil
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
