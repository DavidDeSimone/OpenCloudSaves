package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGameRecordManifestFetcher_LoadETagFromFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "etag-*.ocs")
	assert.NoError(t, err, "Creating temporary ETag file should not return an error")

	etag := "test-etag"
	_, err = tmpFile.Write([]byte(etag))
	assert.NoError(t, err, "Writing ETag to file should not return an error")
	tmpFile.Close()

	fetcher := &GameRecordManifestFetcherImpl{
		etagFile: tmpFile.Name(),
	}
	fetcher.loadETagFromFile()
	assert.Equal(t, etag, fetcher.lastETag, "Loaded ETag should match the one in the file")

	defer os.Remove(tmpFile.Name())
}

func TestGameRecordManifestFetcher_SaveETagToFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "etag-*.ocs")
	assert.NoError(t, err, "Creating temporary ETag file should not return an error")
	tmpFile.Close()

	fetcher := &GameRecordManifestFetcherImpl{
		etagFile: tmpFile.Name(),
	}
	etag := "test-etag"
	err = fetcher.saveETagToFile(etag)
	assert.NoError(t, err, "Saving ETag to file should not return an error")

	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err, "Reading ETag from file should not return an error")
	assert.Equal(t, etag, string(data), "Saved ETag should match the one in the file")

	defer os.Remove(tmpFile.Name())
}

func TestGameRecordManifestFetcher_Fetch(t *testing.T) {
	if os.Getenv("GITHUB_CI_ENV") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	// Create a test server to simulate the manifest server
	content := "test-manifest-content"
	etag := "test-etag"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
		} else {
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(content))
		}
	}))
	defer ts.Close()

	configDir, err := os.UserConfigDir()
	assert.NoError(t, err, "Getting user config directory should not return an error")

	etagFile := filepath.Join(configDir, APP_NAME, "etag.ocs")

	// Delete the etag.ocs file if it exists to simulate a fresh run
	_ = os.Remove(etagFile)

	fetcher := &GameRecordManifestFetcherImpl{
		manifestURL: ts.URL,
		etagFile:    etagFile,
	}

	// Test fetching the manifest for the first time (no ETag)
	fetchedContent, fetchedETag, err := fetcher.Fetch()
	assert.NoError(t, err, "Fetching manifest should not return an error")
	assert.Equal(t, content, string(fetchedContent), "Fetched content should match the test content")
	assert.Equal(t, etag, fetchedETag, "Fetched ETag should match the test ETag")

	// Test fetching the manifest with an existing ETag (not modified)
	_, _, err = fetcher.Fetch()
	assert.EqualError(t, err, ErrManifestNotModified.Error(), "Fetching manifest with an existing ETag should return 'manifest not modified' error")

	// Clean up
	_ = os.Remove(etagFile)
}
