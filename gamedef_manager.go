package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed gamedef_map.json
var gamedefMap []byte

type Datapath struct {
	Path    string `json:"path"`
	Include string `json:"inc"`
}

type GameDef struct {
	DisplayName string      `json:"display_name"`
	SteamId     string      `json:"steam_id"`
	WinPath     []*Datapath `json:"win_path"`
	LinuxPath   []*Datapath `json:"linux_path"`
	DarwinPath  []*Datapath `json:"darwin_path"`
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
		result[syncpath.Path] = make(map[string]SyncFile)
		f, err := os.Open(syncpath.Path)
		if err != nil {
			// @TODO This is kind of a hack, but I need to think of a better way to handle this.
			// The goal is to capture games installed on the SD card as well.
			if runtime.GOOS == "linux" && strings.Index(syncpath.Path, "~/.local/share/Steam") == 0 {
				syncpath.Path = strings.Replace(syncpath.Path, "~/.local/share/Steam", "/run/media/mmcblk0p1", 1)
				f, err = os.Open(syncpath.Path)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		defer f.Close()
		files, err := f.Readdir(0)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			if file.IsDir() {
				fmt.Printf("Logging Directory %v\n", file.Name())
				result[syncpath.Path][file.Name()] = SyncFile{
					Name:  syncpath.Path + file.Name(),
					IsDir: true,
				}
				continue
			}

			result[syncpath.Path][file.Name()] = SyncFile{
				Name:  syncpath.Path + file.Name(),
				IsDir: false,
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

	// d.GetInstallLocationFromSteamId()

	result := []Datapath{}
	if platform == "windows" {
		if len(d.WinPath) == 0 {
			return nil, fmt.Errorf("game %v save files not supported for platform %v", d.DisplayName, platform)
		}

		for _, datapath := range d.WinPath {
			path := datapath.Path
			winpath := strings.Replace(path, "%APPDATA%", os.Getenv("APPDATA"), 1)
			winpath = strings.Replace(winpath, "%LOCALAPPDATA%", os.Getenv("LOCALAPPDATA"), 1)
			winpath = strings.Replace(winpath, "%USERPROFILE%", os.Getenv("USERPROFILE"), 1)
			winpath = strings.Replace(winpath, "%STEAM%", steamLocation, 1)

			current, err := user.Current()
			if err != nil {
				winpath = strings.ReplaceAll(winpath, "%USERID%", current.Username)
			}

			result = append(result, Datapath{
				Path:    prefix + winpath + separator,
				Include: datapath.Include,
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
				Path:    prefix + darwinPath + separator,
				Include: datapath.Include,
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
			linuxPath = strings.Replace(linuxPath, "~", homedir, 1)
			linuxPath = strings.Replace(linuxPath, "$HOME", os.Getenv("HOME"), 1)
			xdr := os.Getenv("XDG_CONFIG_HOME")
			if xdr != "" {
				linuxPath = strings.Replace(linuxPath, "$XDG_CONFIG_HOME", xdr, 1)
			} else {
				linuxPath = strings.Replace(linuxPath, "$XDG_CONFIG_HOME", homedir, 1)
			}

			if strings.Contains(datapath.Path, "/pfx/") {
				if len(d.WinPath) == 0 {
					fmt.Println("Unable to resolve " + datapath.Path)
					continue
				}

				winpath := d.WinPath[0].Path
				winpath = strings.Replace(winpath, "%APPDATA%", "users/steamuser/AppData/Roaming/", 1)
				winpath = strings.Replace(winpath, "%LOCALAPPDATA%", "users/steamuser/AppData/Local/", 1)
				winpath = strings.Replace(winpath, "%USERPROFILE%", "users/steamuser/", 1)
				winpath = strings.ReplaceAll(winpath, "%USERID", "steamuser")
				winpath = strings.ReplaceAll(winpath, "\\", "/")

				linuxPath = linuxPath + "drive_c/" + winpath
				fmt.Println("Datapath " + datapath.Path)
			}

			result = append(result, Datapath{
				Path:    prefix + linuxPath + separator,
				Include: datapath.Include,
			})
		}
	} else {
		return nil, fmt.Errorf("non-supported platform %v", platform)
	}

	return result, nil
}

type FsGameDefManager struct {
	gamedefs            map[string]*GameDef
	userOverrideLoction string
	cm                  *CloudManager
}

type GameDefManager interface {
	ApplyUserOverrides() error
	CommitUserOverrides() error
	AddUserOverride(key string, jsonOverride string) error
	GetGameDefMap() map[string]*GameDef
	GetSyncpathForGame(id string) ([]Datapath, error)
	GetUserOverrideLocation() string
	SetCloudManager(cm *CloudManager)
}

func (d *FsGameDefManager) SetCloudManager(cm *CloudManager) {
	d.cm = cm
}

func (d *FsGameDefManager) ApplyUserOverrides() error {
	fileName := d.GetUserOverrideLocation()
	content, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Println(err)
		return err
	}

	mid := make(map[string]json.RawMessage)
	err = json.Unmarshal(content, &mid)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range mid {
		def := &GameDef{}
		err = json.Unmarshal(v, def)
		if err != nil {
			log.Fatal(err)
		}
		d.gamedefs[k] = def
	}
	return nil
}

func (d *FsGameDefManager) GetUserOverrideLocation() string {
	if d.userOverrideLoction == "" {
		d.userOverrideLoction = GetDefaultUserOverridePath()
	}

	return d.userOverrideLoction
}

func (d *FsGameDefManager) CommitUserOverrides() error {
	newResult, err := json.Marshal(d.gamedefs)
	if err != nil {
		return err
	}

	fileName := d.GetUserOverrideLocation()
	err = os.WriteFile(fileName, newResult, os.ModePerm)
	if err != nil {
		return err
	}

	go d.CommitCloudUserOverride()

	return nil
}

func (d *FsGameDefManager) AddUserOverride(key string, jsonOverride string) error {
	def := &GameDef{}
	err := json.Unmarshal([]byte(jsonOverride), def)
	if err != nil {
		return err
	}

	d.gamedefs[key] = def

	return d.CommitUserOverrides()
}

func (d *FsGameDefManager) GetGameDefMap() map[string]*GameDef {
	return d.gamedefs
}

func GetDefaultUserOverridePath() string {
	cacheDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	separator := string(os.PathSeparator)

	fmt.Println("Getting User Override Path " + cacheDir + separator + APP_NAME + separator + "user_overrides.json")
	os.Stdout.WriteString("Getting User Override Path " + cacheDir + separator + APP_NAME + separator + "user_overrides.json")
	return cacheDir + separator + APP_NAME + separator + "user_overrides.json"
}

func MakeGameDefManager(userOverride string) GameDefManager {
	if userOverride == "" {
		userOverride = GetDefaultUserOverridePath()
	}

	dm := &FsGameDefManager{
		gamedefs:            make(map[string]*GameDef),
		userOverrideLoction: userOverride,
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
		dm.gamedefs[k] = d
	}

	dm.ApplyUserOverrides()

	return dm
}

func (d *FsGameDefManager) GetSyncpathForGame(id string) ([]Datapath, error) {
	driver, ok := d.gamedefs[id]
	if !ok {
		return nil, fmt.Errorf("failed to find game (%v)", id)
	}

	return driver.GetSyncpaths()
}

func (dm *FsGameDefManager) CommitCloudUserOverride() error {
	if dm.cm == nil {
		fmt.Println("No cloud driver present")
		return nil
	}

	userOverride := dm.GetUserOverrideLocation()
	return ApplyCloudUserOverride(dm.cm, userOverride)
}

func ApplyCloudUserOverride(cm *CloudManager, userOverride string) error {
	storage := GetCurrentStorageProvider()
	if storage == nil {
		return fmt.Errorf("no cloud storage set")
	}

	if userOverride == "" {
		userOverride = GetDefaultUserOverridePath()
	}

	path := filepath.Dir(userOverride)
	ops := GetDefaultCloudOptions()
	_, err := cm.BisyncDir(storage, ops, path, ToplevelCloudFolder+"user_settings/")
	return err
}
