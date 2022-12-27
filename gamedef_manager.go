package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed gamedef_map.json
var gamedefMap []byte

// @TODO better utilize saves_cross_compatible to split saves between platforms
type GameDef struct {
	Display_name           string   `json:"display_name"`
	Win_path               string   `json:"win_path"`
	Linux_path             string   `json:"linux_path"`
	Darwin_path            string   `json:"darwin_path"`
	Saves_cross_compatible bool     `json:"saves_cross_compatible"`
	Save_ext               string   `json:"save_ext"`
	Relative_to_homedir    []string `json:"relative_to_homedir"`
}

func (d *GameDef) GetFilenames() (map[string]string, error) {
	syncpath, err := d.GetSyncpath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(syncpath)
	if err != nil {
		return nil, err
	}

	defer f.Close()
	files, err := f.Readdir(0)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, file := range files {
		if d.Save_ext == "" || filepath.Ext(file.Name()) == d.Save_ext {
			result[file.Name()] = syncpath + file.Name()
			// result = append(result, syncpath+file.Name())
			LogVerbose("Found Save Files: ", file.Name())
		}
	}

	return result, nil
}

func (d *GameDef) GetSyncpath() (string, error) {
	platform := runtime.GOOS
	prefix := ""
	separator := string(os.PathSeparator)

	for _, e := range d.Relative_to_homedir {
		if e == platform {
			homedir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}

			prefix = homedir + separator
		}
	}

	result := ""
	if platform == "windows" {
		if d.Win_path == "" {
			return "", fmt.Errorf("game %v save files not supported for platform %v", d.Display_name, platform)
		}

		result = prefix + d.Win_path + separator
	} else if platform == "darwin" {
		if d.Darwin_path == "" {
			return "", fmt.Errorf("game %v save files not supported for platform %v", d.Display_name, platform)
		}

		result = prefix + d.Darwin_path + separator
	} else if platform == "linux" {
		if d.Linux_path == "" {
			return "", fmt.Errorf("game %v save files not supported for platform %v", d.Display_name, platform)
		}

		result = prefix + d.Linux_path + separator
	} else {
		return "", fmt.Errorf("non-supported platform %v", platform)
	}

	if result[len(result)-1] != os.PathSeparator {
		result += separator
	}

	LogVerbose("Determined Savepath: ", result)
	return result, nil
}

type GameDefManager struct {
	gamedefs map[string]GameDef
}

func (d *GameDefManager) GetGameDefMap() map[string]GameDef {
	return d.gamedefs
}

func MakeGameDefManager() *GameDefManager {
	dm := &GameDefManager{
		gamedefs: make(map[string]GameDef),
	}

	mid := make(map[string]json.RawMessage)
	err := json.Unmarshal(gamedefMap, &mid)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range mid {
		d := &GameDef{}
		err = json.Unmarshal(v, d)
		if err != nil {
			log.Fatal(err)
		}
		dm.gamedefs[k] = *d
	}

	LogVerbose(dm.gamedefs)
	return dm
}

func (d *GameDefManager) GetFilesForGame(id string) (map[string]string, error) {
	driver, ok := d.gamedefs[id]
	if !ok {
		return nil, fmt.Errorf("failed to find game (%v)", id)
	}
	return driver.GetFilenames()
}

func (d *GameDefManager) GetSyncpathForGame(id string) (string, error) {
	driver, ok := d.gamedefs[id]
	if !ok {
		return "", fmt.Errorf("failed to find game (%v)", id)
	}

	return driver.GetSyncpath()
}
