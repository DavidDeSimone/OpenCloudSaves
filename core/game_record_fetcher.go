package core

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var ErrManifestNotModified = errors.New("manifest not modified")

type GameRecordManifestFetcher interface {
	Fetch() (content []byte, etag string, err error)
}

type GameRecordManifestFetcherImpl struct {
	mu          sync.Mutex
	lastETag    string
	manifestURL string
	etagFile    string
}

func NewGameRecordManifestFetcher() GameRecordManifestFetcher {
	cacheDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	etagFile := filepath.Join(cacheDir, APP_NAME, "etag.ocs")

	return &GameRecordManifestFetcherImpl{
		manifestURL: "https://raw.githubusercontent.com/mtkennerly/ludusavi-manifest/master/data/manifest.yaml",
		etagFile:    etagFile,
	}
}

func (fetcher *GameRecordManifestFetcherImpl) loadETagFromFile() {
	data, err := os.ReadFile(fetcher.etagFile)
	if err == nil {
		fetcher.lastETag = string(data)
	}
}

func (fetcher *GameRecordManifestFetcherImpl) saveETagToFile(etag string) error {
	return os.WriteFile(fetcher.etagFile, []byte(etag), 0644)
}

func (fetcher *GameRecordManifestFetcherImpl) Fetch() ([]byte, string, error) {
	fetcher.mu.Lock()
	defer fetcher.mu.Unlock()

	fetcher.loadETagFromFile()

	client := &http.Client{}
	req, err := http.NewRequest("GET", fetcher.manifestURL, nil)
	if err != nil {
		ErrorLogger.Println(err)
		return nil, "", err
	}

	if fetcher.lastETag != "" {
		req.Header.Set("If-None-Match", fetcher.lastETag)
	}

	resp, err := client.Do(req)
	if err != nil {
		ErrorLogger.Println(err)
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, "", ErrManifestNotModified
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("failed to fetch manifest")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		ErrorLogger.Println(err)
		return nil, "", err
	}

	etag := resp.Header.Get("ETag")
	err = fetcher.saveETagToFile(etag)
	if err != nil {
		return nil, "", err
	}
	fetcher.lastETag = etag

	return content, etag, nil
}
