package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed gamedef_map.json
var gamedefMap []byte

type Datapath struct {
	Path   string   `json:"path"`
	Exts   []string `json:"exts"`
	Parent string   `json:"parent"`
}

// @TODO better utilize saves_cross_compatible to split saves between platforms
type GameDef struct {
	DisplayName          string     `json:"display_name"`
	WinPath              []Datapath `json:"win_path"`
	LinuxPath            []Datapath `json:"linux_path"`
	DarwinPath           []Datapath `json:"darwin_path"`
	SavesCrossCompatible bool       `json:"saves_cross_compatible"`
}

type SyncFile struct {
	Name  string
	IsDir bool
}

func (d *GameDef) GetSteamLocation() string {
	switch runtime.GOOS {
	case "windows":
		return "C:\\Program Files (x86)\\Steam"
	case "darwin":
		return "/Applications/Steam"
	case "linux":
		return "~/.local/share/Steam"
	default:
		return ""
	}
}

func (d *GameDef) GetFilenames() (map[string]map[string]SyncFile, error) {
	syncpaths, err := d.GetSyncpaths()
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]SyncFile)
	for _, syncpath := range syncpaths {
		result[syncpath.Parent] = make(map[string]SyncFile)
		f, err := os.Open(syncpath.Path)
		if err != nil {
			return nil, err
		}

		defer f.Close()
		files, err := f.Readdir(0)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if file.IsDir() {
				fmt.Printf("Logging Directory %v\n", file.Name())
				result[syncpath.Parent][file.Name()] = SyncFile{
					Name:  syncpath.Path + file.Name(),
					IsDir: true,
				}
				continue
			}

			if len(syncpath.Exts) == 0 {
				result[syncpath.Parent][file.Name()] = SyncFile{
					Name:  syncpath.Path + file.Name(),
					IsDir: false,
				}
				continue
			}

			for _, ext := range syncpath.Exts {
				if filepath.Ext(file.Name()) == ext {
					result[syncpath.Parent][file.Name()] = SyncFile{
						Name:  syncpath.Path + file.Name(),
						IsDir: false,
					}
					LogVerbose("Found Save Files: ", file.Name())
					break
				}
			}

		}
	}

	return result, nil
}

func (d *GameDef) GetSyncpaths() ([]Datapath, error) {
	platform := runtime.GOOS
	prefix := ""
	separator := string(os.PathSeparator)
	steamLocation := d.GetSteamLocation()

	result := []Datapath{}
	if platform == "windows" {
		if len(d.WinPath) == 0 {
			return nil, fmt.Errorf("game %v save files not supported for platform %v", d.DisplayName, platform)
		}

		for _, datapath := range d.WinPath {
			path := datapath.Path
			fmt.Println(os.Getenv("APPDATA"))
			winpath := strings.Replace(path, "%AppData%", os.Getenv("APPDATA"), 1)
			winpath = strings.Replace(winpath, "%LocalAppData%", os.Getenv("LOCALAPPDATA"), 1)
			winpath = strings.Replace(winpath, "%STEAM%", steamLocation, 1)
			result = append(result, Datapath{
				Path:   prefix + winpath + separator,
				Exts:   datapath.Exts,
				Parent: datapath.Parent,
			})
		}
	} else if platform == "darwin" {
		if len(d.DarwinPath) == 0 {
			return nil, fmt.Errorf("game %v save files not supported for platform %v", d.DisplayName, platform)
		}

		for _, datapath := range d.DarwinPath {
			homedir, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			path := datapath.Path
			darwinPath := strings.Replace(path, "%STEAM%", steamLocation, 1)
			darwinPath = strings.Replace(darwinPath, "~", homedir, 1)
			darwinPath = strings.Replace(darwinPath, "$HOME", os.Getenv("HOME"), 1)

			result = append(result, Datapath{
				Path:   prefix + darwinPath + separator,
				Exts:   datapath.Exts,
				Parent: datapath.Parent,
			})
		}
	} else if platform == "linux" {
		if len(d.LinuxPath) == 0 {
			return nil, fmt.Errorf("game %v save files not supported for platform %v", d.DisplayName, platform)
		}

		for _, datapath := range d.LinuxPath {
			homedir, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			path := datapath.Path
			linuxPath := strings.Replace(path, "%STEAM%", steamLocation, 1)
			linuxPath = strings.Replace(path, "~", homedir, 1)
			linuxPath = strings.Replace(linuxPath, "$HOME", os.Getenv("HOME"), 1)

			result = append(result, Datapath{
				Path:   prefix + linuxPath + separator,
				Exts:   datapath.Exts,
				Parent: datapath.Parent,
			})
		}
	} else {
		return nil, fmt.Errorf("non-supported platform %v", platform)
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

func (d *GameDefManager) GetFilesForGame(id string, parent string) (map[string]SyncFile, error) {
	driver, ok := d.gamedefs[id]
	if !ok {
		return nil, fmt.Errorf("failed to find game (%v)", id)
	}
	result, err := driver.GetFilenames()
	if err != nil {
		return nil, err
	}
	files, ok := result[parent]
	if !ok {
		return nil, fmt.Errorf("Failed to find parent (%v) for game (%v)", parent, id)
	}
	return files, nil
}

func (d *GameDefManager) GetSyncpathForGame(id string) ([]Datapath, error) {
	driver, ok := d.gamedefs[id]
	if !ok {
		return nil, fmt.Errorf("failed to find game (%v)", id)
	}

	return driver.GetSyncpaths()
}
