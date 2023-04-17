package core

import (
	"fmt"
	"strings"
)

// this class converts a game record to a game def

type GameRecordConverter interface {
	Convert(name string, record *GameRecord) (*GameDef, error)
}

var gameRecordConverterInstance GameRecordConverter

func GetGameRecordConverter() GameRecordConverter {
	if gameRecordConverterInstance == nil {
		gameRecordConverterInstance = NewGameRecordConverter()
	}
	return gameRecordConverterInstance
}

// GameRecordConverterImpl is the default implementation of GameRecordConverter
type GameRecordConverterImpl struct {
}

// NewGameRecordConverter returns a new GameRecordConverter
func NewGameRecordConverter() GameRecordConverter {
	return &GameRecordConverterImpl{}
}

func getCurrentUserSteamId() string {
	return "12345"
}

// Convert converts a game record to a game def
func (grc *GameRecordConverterImpl) Convert(name string, record *GameRecord) (*GameDef, error) {
	// fmt.Println("Converting " + name)
	result := &GameDef{
		DisplayName: name,
		SteamId:     fmt.Sprintf("%v", record.Steam.Id),
		WinPath:     []*Datapath{},
		LinuxPath:   []*Datapath{},
		DarwinPath:  []*Datapath{},
	}

	// x, _ := json.Marshal(record)
	// fmt.Println(string(x))

	for fileName, fileProperty := range record.Files {
		// check if file tags contains special "save" tag
		isSave := false
		for _, tag := range fileProperty.Tags {
			if tag == "save" {
				isSave = true
				break
			}
		}

		if !isSave {
			continue
		}

		// @TODO convert these to actual paths instead of shortcuts
		fileName = strings.ReplaceAll(fileName, "<home>", "$HOME")
		fileName = strings.ReplaceAll(fileName, "<steam>", "%STEAM%")
		fileName = strings.ReplaceAll(fileName, "<steamid>", getCurrentUserSteamId())
		fileName = strings.ReplaceAll(fileName, "<appdata>", "%APPDATA%")
		fileName = strings.ReplaceAll(fileName, "<localappdata>", "%LOCALAPPDATA%")
		fileName = strings.ReplaceAll(fileName, "<userprofile>", "%USERPROFILE%")
		// fileName = strings.ReplaceAll(fileName, "<userid>", "%USERID%")
		fileName = strings.ReplaceAll(fileName, "<documents>", "%DOCUMENTS%")
		fileName = strings.ReplaceAll(fileName, "<winDocuments>", "%DOCUMENTS%")

		foundLinux := false
		for _, when := range fileProperty.When {
			if when.Os == "windows" {
				result.WinPath = append(result.WinPath, &Datapath{
					Path: fileName,
				})
			} else if when.Os == "linux" {
				result.LinuxPath = append(result.LinuxPath, &Datapath{
					Path: fileName,
				})
				foundLinux = true
			} else if when.Os == "darwin" {
				result.DarwinPath = append(result.DarwinPath, &Datapath{
					Path: fileName,
				})
			}
		}
		// fmt.Println(fileName)

		// Fall back to a proton style game reference is there is no dedicated linux reference
		if !foundLinux {
			result.LinuxPath = append(result.LinuxPath, &Datapath{
				Path: "%STEAM%/steamapps/compatdata/" + fmt.Sprintf("%v", record.Steam.Id) + "/pfx/",
			})
		}
	}

	// s, _ := json.Marshal(result)
	// fmt.Println(string(s))

	return result, nil
}
