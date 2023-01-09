package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/DavidDeSimone/memfs"
)

type MockGameDefManager struct {
	gamedefs map[string]*GameDef
	rootFs   fs.FS
}

func (m *MockGameDefManager) ApplyUserOverrides() error {
	return nil
}
func (m *MockGameDefManager) CommitUserOverrides() error {
	return nil
}
func (m *MockGameDefManager) AddUserOverride(key string, jsonOverride string) error {
	return nil
}
func (m *MockGameDefManager) GetGameDefMap() map[string]*GameDef {
	return m.gamedefs
}

// @TODO this could be cut down by making a function to mock the FS access,
// which will better test GetFilesForGame too
func (m *MockGameDefManager) GetFilesForGame(id string, parent string) (map[string]SyncFile, error) {
	syncpaths, err := m.GetSyncpathForGame(id)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]SyncFile)
	for _, syncpath := range syncpaths {
		fmt.Println(syncpath)
		ignoreMap := make(map[string]bool)
		fmt.Println(syncpath.Ignore)
		for _, ignore := range syncpath.Ignore {
			fmt.Println("Will ignore -> " + ignore)
			ignoreMap[ignore] = true
		}

		result[syncpath.Parent] = make(map[string]SyncFile)
		dirPath := strings.TrimSuffix(syncpath.Path, string("/"))
		fmt.Println("[Mock] Examining Dirpath -> " + dirPath)
		files, err := fs.ReadDir(m.rootFs, dirPath) //f.Readdir(0)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			fmt.Println("[Mock] Examining File -> " + file.Name())
			_, ok := ignoreMap[file.Name()]
			fmt.Println("Examining " + file.Name())
			if ok {
				fmt.Println("Ignoring " + file.Name())
				continue
			}

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
					break
				}
			}

		}
	}

	return result[parent], nil
}

func (m *MockGameDefManager) GetSyncpathForGame(id string) ([]Datapath, error) {
	d, ok := m.gamedefs[id]
	if !ok {
		return nil, errors.New("failed to find gamedef " + id)
	}
	return d.GetSyncpaths()
}

type MockLocalFs struct {
	rootFs *memfs.FS
}

func (d *MockLocalFs) WriteFile(path string, data []byte, mode fs.FileMode) error {
	return d.rootFs.WriteFile(path, data, mode)
}

func (d *MockLocalFs) ReadFile(path string) ([]byte, error) {
	return fs.ReadFile(d.rootFs, path)
}

func (d *MockLocalFs) ReadDir(path string) ([]fs.DirEntry, error) {
	return fs.ReadDir(d.rootFs, path)
}

func (d *MockLocalFs) Stat(path string) (fs.FileInfo, error) {
	return fs.Stat(d.rootFs, path)
}

func (d *MockLocalFs) GetFileHash(path string) (string, error) {
	f, err := d.rootFs.Open(path)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
