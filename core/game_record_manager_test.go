package core

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const manifestContent = `
key1:
  steam:
    id: 12345
key2:
  gog:
    id: 67890
`

func createTestManifestFile(content string) (string, error) {
	tmpFile, err := ioutil.TempFile("", "manifest-*.yaml")
	if err != nil {
		return "", err
	}
	_, err = tmpFile.Write([]byte(content))
	if err != nil {
		return "", err
	}
	tmpFile.Close()
	return tmpFile.Name(), nil
}

func TestGameRecordManager(t *testing.T) {
	manifestPath, err := createTestManifestFile(manifestContent)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(manifestPath)

	grm := NewGameRecordManager()
	grm.SetManifestLocation(manifestPath)

	err = grm.RefreshManifestFromDisk()
	assert.NoError(t, err, "Refreshing manifest from disk should not return an error")

	gameRecord, err := grm.GetGameRecordByKey("key1")
	assert.NoError(t, err, "Getting game record by key should not return an error")
	assert.NotNil(t, gameRecord, "Game record should not be nil")
	assert.Equal(t, 12345, gameRecord.Steam.Id, "Steam ID should be 12345")

	gameRecord, err = grm.GetGameRecordByKey("nonexistent")
	assert.Error(t, err, "Getting game record by non-existent key should return an error")
	assert.Nil(t, gameRecord, "Game record should be nil")

	newGameRecord := &GameRecord{
		Steam: SteamProperties{
			Id: 54321,
		},
	}

	grm.SetGameRecordByKey("key3", newGameRecord)
	err = grm.CommitCustomGameRecords()
	assert.NoError(t, err, "Committing custom game records should not return an error")

	err = grm.RefreshManifestFromDisk()
	assert.NoError(t, err, "Refreshing manifest from disk should not return an error")

	gameRecord, err = grm.GetGameRecordByKey("key3")
	assert.NoError(t, err, "Getting game record by key should not return an error")
	assert.NotNil(t, gameRecord, "Game record should not be nil")
	assert.Equal(t, 54321, gameRecord.Steam.Id, "Steam ID should be 54321")
}
