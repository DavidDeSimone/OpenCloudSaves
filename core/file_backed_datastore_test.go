package core_test

import (
	"context"
	"opencloudsave/core"
	"os/exec"
	"testing"
)

type MockCloudManager struct {
	syncCalled bool
}

type MockStorageProvider struct{}

type MockCloudOptions struct{}

func (m *MockCloudManager) PerformSyncOperation(ctx context.Context, storage core.Storage, ops core.CloudOptions, src, dst string) (bool, error) {
	m.syncCalled = true
	return true, nil
}

func (s *MockStorageProvider) GetCreationCommand(ctx context.Context) *exec.Cmd {
	return nil
}

func (s *MockStorageProvider) GetName() string {
	return "opencloudsave-mockstorage"
}

type TestData struct {
	Name  string
	Value int
}

func TestCloudBasedDatastore_StoreAndFetch(t *testing.T) {
	cm := &MockCloudManager{}
	sp := &MockStorageProvider{}
	co := &MockCloudOptions{}

	ds := core.NewCloudBasedDatastore[TestData]("test.json", cm, sp, co)
	ds.Initialize()

	testData := TestData{
		Name:  "Test",
		Value: 42,
	}

	// Test Store
	ds.Store(testData)

	// Test Flush
	err := ds.Flush()
	if err != nil {
		t.Error(err)
	}

	// Test Fetch
	fetchedData, err := ds.Fetch()
	if err != nil {
		t.Error(err)
	}
	if fetchedData == nil {
		t.Error("fetch failed to retrieve data.")
	}
	if fetchedData != nil && *fetchedData != testData {
		t.Errorf("data mismatch %v / %v", *fetchedData, testData)
	}
}

func TestCloudBasedDatastore_FetchError(t *testing.T) {
	cm := &MockCloudManager{
		syncCalled: true,
	}
	sp := &MockStorageProvider{}
	co := &MockCloudOptions{}

	ds := core.NewCloudBasedDatastore[TestData]("nonexistent.json", cm, sp, co)
	ds.Initialize()

	_, err := ds.Fetch()
	if err == nil {
		t.Error("succeeded to fetch non-existent file")
	}
}
