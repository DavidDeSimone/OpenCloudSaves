package core

// These are unit tests for GameRecordConverterImpl
import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGameRecordConverterImpl_Convert(t *testing.T) {
	converter := GetGameRecordConverter()

	// Test converting a game record with no properties
	gameRecord := &GameRecord{}
	converted, _ := converter.Convert("test", gameRecord)
	assert.NotEqual(t, nil, converted, "Converted game record should not be nil")

	// Test converting a game record with all properties
	gameRecord = &GameRecord{
		Steam: SteamProperties{
			Id: 12345,
		},
		Gog: GogProperties{
			Id: 67890,
		},
		Files: map[string]FileProperties{
			"file1": {
				Tags: []Tag{"save"},
				When: []FileConstraint{
					{
						Os:    "windows",
						Store: "steam",
					},
					{
						Os:    "linux",
						Store: "gog",
					},
				},
			},
		},
	}

	converted, err := converter.Convert("test", gameRecord)
	assert.Equal(t, nil, err, "Converting game record should not return an error")
	assert.Equal(t, "12345", converted.SteamId, "Converted game record should have Steam ID 12345")
	assert.Equal(t, "file1", converted.WinPath[0].Path)

	gameRecord = &GameRecord{
		Steam: SteamProperties{
			Id: 11111,
		},
		Gog: GogProperties{
			Id: 67890,
		},
		Files: map[string]FileProperties{
			"<home>/file1": {
				Tags: []Tag{"save"},
				When: []FileConstraint{
					{
						Os:    "windows",
						Store: "steam",
					},
					{
						Os:    "linux",
						Store: "gog",
					},
				},
			},
		},
	}

	converted, err = converter.Convert("test", gameRecord)
	assert.Equal(t, nil, err, "Converting game record should not return an error")
	assert.Equal(t, "11111", converted.SteamId, "Converted game record should have Steam ID 12345")
	assert.Equal(t, "$HOME/file1", converted.WinPath[0].Path)
}
