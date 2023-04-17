package core

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

//go:embed gamedef_map.json
var gamedefMap []byte

type Datapath struct {
	Path    string `json:"path"`
	Include string `json:"inc"`
}

type GameDef struct {
	DisplayName           string      `json:"display_name"`
	SteamId               string      `json:"steam_id"`
	WinPath               []*Datapath `json:"win_path"`
	LinuxPath             []*Datapath `json:"linux_path"`
	DarwinPath            []*Datapath `json:"darwin_path"`
	Hidden                bool        `json:"hidden"`
	CustomFlags           string      `json:"flags"`
	SelectInMultisyncMenu bool        `json:"selectMultiSync"`
}

type SyncFile struct {
	Name  string
	IsDir bool
}

const (
	WindowsPlatform = "windows"
	DarwinPlatform  = "darwin"
	LinuxPlatform   = "linux"
)

const UserOverrideFilename = "user_overrides.json"

func (d *GameDef) GetSteamLocation() string {
	switch runtime.GOOS {
	case WindowsPlatform:
		return "C:\\Program Files (x86)\\Steam"
	case DarwinPlatform:
		return "/Applications/Steam"
	case LinuxPlatform:
		return "~/.local/share/Steam"
	default:
		return ""
	}
}

func (d *GameDef) GetSyncpaths() ([]Datapath, error) {
	platform := runtime.GOOS
	separator := string(os.PathSeparator)
	steamLocation := d.GetSteamLocation()

	// d.GetInstallLocationFromSteamId()

	result := []Datapath{}
	if platform == WindowsPlatform {
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

			winpath = strings.Replace(winpath, "$HOME", current.HomeDir, 1)
			winpath = strings.Replace(winpath, "~", current.HomeDir, 1)
			winpath = strings.Replace(winpath, "%DOCUMENTS%", filepath.Join(current.HomeDir, "Documents"), 1)

			result = append(result, Datapath{
				Path:    winpath + separator,
				Include: datapath.Include,
			})
		}
	} else if platform == DarwinPlatform {
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

			if strings.TrimSpace(darwinPath) == "" {
				continue
			}

			result = append(result, Datapath{
				Path:    darwinPath + separator,
				Include: datapath.Include,
			})
		}
	} else if platform == LinuxPlatform {
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

			if strings.HasSuffix(datapath.Path, "pfx") || strings.HasSuffix(datapath.Path, "pfx/") {
				if len(d.WinPath) == 0 {
					continue
				}

				winpath := d.WinPath[0].Path
				winpath = strings.Replace(winpath, "C:\\", "", 1)
				winpath = strings.Replace(winpath, "%APPDATA%", "users/steamuser/AppData/Roaming/", 1)
				winpath = strings.Replace(winpath, "%LOCALAPPDATA%", "users/steamuser/AppData/Local/", 1)
				winpath = strings.Replace(winpath, "%USERPROFILE%", "users/steamuser/", 1)
				winpath = strings.ReplaceAll(winpath, "%USERID%", "steamuser")
				winpath = strings.ReplaceAll(winpath, "\\", "/")

				linuxPath = linuxPath + "drive_c/" + winpath
			}

			if strings.TrimSpace(linuxPath) == "" {
				continue
			}

			result = append(result, Datapath{
				Path:    linuxPath + separator,
				Include: datapath.Include,
			})
		}
	} else {
		return nil, fmt.Errorf("non-supported platform %v", platform)
	}

	return result, nil
}

var fsmtx sync.Mutex

type FsGameDefManager struct {
	gamedefs            map[string]*GameDef
	userOverrideLoction string
}

type GameDefManager interface {
	ApplyUserOverrides() error
	CommitUserOverrides() error
	AddUserOverride(key string, jsonOverride string) error
	GetGameDefMap() map[string]*GameDef
	RemoveGameDef(key string)
	GetSyncpathForGame(id string) ([]Datapath, error)
	GetUserOverrideLocation() string

	CommitCloudUserOverride(context.Context) error
}

func (d *FsGameDefManager) RemoveGameDef(key string) {
	InfoLogger.Println("Removing " + key)
	entry, ok := d.gamedefs[key]
	if ok {
		entry.Hidden = true
	}
}

func (d *FsGameDefManager) ApplyUserOverrides() error {
	fileName := d.GetUserOverrideLocation()
	fsmtx.Lock()
	content, err := os.ReadFile(fileName)
	fsmtx.Unlock()
	if err != nil {
		InfoLogger.Println(err)
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
	InfoLogger.Println("Commiting Gamedef Map...")
	newResult, err := json.Marshal(d.gamedefs)
	if err != nil {
		return err
	}

	fileName := d.GetUserOverrideLocation()
	fsmtx.Lock()
	err = os.WriteFile(fileName, newResult, os.ModePerm)
	fsmtx.Unlock()

	if err != nil {
		return err
	}

	GetUserSettingsManager().RequestSyncNonBlocking(context.Background(), GetDefaultUserOverridePath(), nil)
	// go d.CommitCloudUserOverride()

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

	return filepath.Join(cacheDir, APP_NAME, UserOverrideFilename)
}

func MakeDefaultGameDefManager() GameDefManager {
	return MakeGameDefManager("")
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

func (dm *FsGameDefManager) CommitCloudUserOverride(ctx context.Context) error {
	fsmtx.Lock()
	defer fsmtx.Unlock()
	return GetUserSettingsManager().RequestSync(ctx, dm.GetUserOverrideLocation())
}
