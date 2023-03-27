//go:build windows

package platform

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/andygrunwald/vdf"
	"golang.org/x/sys/windows/registry"
)

func GetPath() string {
	return "./bin/rclone.exe"
}

func StripWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func (d *GameDef) GetInstallLocationFromSteamId() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		log.Panic(err)
	}
	defer key.Close()

	steamPath, _, err := key.GetStringValue("SteamPath")
	if err != nil {
		log.Panic(err)
	}

	libraryFoldersStr, err := os.Open(filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		log.Panic(err)
	}

	parser := vdf.NewParser(libraryFoldersStr)
	libraryFoldersMap, err := parser.Parse()
	if err != nil {
		log.Panic(err)
	}

	jsonStr, err := json.Marshal(libraryFoldersMap)
	if err != nil {
		log.Panic(err)
	}

	libraryFolders := LibraryFolders{}
	json.Unmarshal(jsonStr, &libraryFolders)

	for _, libraryFolder := range libraryFolders.LibraryFolders {
		for steamId := range libraryFolder.Apps {
			if steamId == d.SteamId {
				steamAppsDir := filepath.Join(libraryFolder.Path, "steamapps")

				appManifestStr, err := os.Open(filepath.Join(steamAppsDir, fmt.Sprintf("appmanifest_%s.acf", d.SteamId)))
				if err != nil {
					log.Panic(err)
				}

				parser := vdf.NewParser(appManifestStr)
				appManifestMap, err := parser.Parse()
				if err != nil {
					log.Panic(err)
				}

				jsonStr, err := json.Marshal(appManifestMap)
				if err != nil {
					log.Panic(err)
				}

				appManifest := AppManifest{}
				json.Unmarshal(jsonStr, &appManifest)

				result := filepath.Join(steamAppsDir, "common", appManifest.AppState.InstallDir)
				return result
			}
		}
	}

	return ""
}
