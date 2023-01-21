package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DavidDeSimone/memfs"
)

type MockCloudFile struct {
	Name    string
	Id      string
	ModTime string
}

func (f *MockCloudFile) GetName() string {
	return f.Name
}

func (f *MockCloudFile) GetId() string {
	return f.Id
}

func (f *MockCloudFile) GetModTime() string {
	return f.ModTime
}

type MockCloudDriver struct {
	fs *memfs.FS
}

func (d *MockCloudDriver) InitDriver() error {
	d.fs = memfs.New()
	d.fs.MkdirAll("root", os.ModePerm)
	return nil
}
func (d *MockCloudDriver) ListFiles(parentId string) ([]CloudFile, error) {
	result := []CloudFile{}
	fs.WalkDir(d.fs, parentId, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		modtime, err := entry.Info()
		if err != nil {
			fmt.Println(err)
			return err
		}

		result = append(result, &MockCloudFile{
			Name:    entry.Name(),
			Id:      path,
			ModTime: modtime.ModTime().Format(time.RFC3339),
		})

		return nil
	})

	return result, nil
}

func (d *MockCloudDriver) CreateDir(name string, parentId string) (CloudFile, error) {
	name = strings.TrimSuffix(name, "/")
	err := d.fs.MkdirAll(parentId+"/"+name, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &MockCloudFile{
		Name:    name,
		Id:      parentId + "/" + name,
		ModTime: time.Now().Format(time.RFC3339),
	}, nil
}
func (d *MockCloudDriver) DownloadFile(fileId string, filePath string, fileName string, prorgress func(int64, int64)) (CloudFile, error) {
	fileContent, err := fs.ReadFile(d.fs, fileId)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(filePath, fileContent, os.ModePerm)
	if err != nil {
		return nil, err
	}

	remote, err := fs.Stat(d.fs, fileId)
	if err != nil {
		return nil, err
	}

	err = os.Chtimes(filePath, remote.ModTime(), remote.ModTime())
	if err != nil {
		return nil, err
	}

	return &MockCloudFile{
		Name:    fileName,
		Id:      fileId,
		ModTime: remote.ModTime().Format(time.RFC3339),
	}, nil
}
func (d *MockCloudDriver) UploadFile(fileId string, filePath string, fileName string, prorgress func(int64, int64)) (CloudFile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = d.fs.WriteFile(fileId, content, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &MockCloudFile{
		Name:    fileName,
		Id:      fileId,
		ModTime: time.Now().Format(time.RFC3339),
	}, nil
}
func (d *MockCloudDriver) CreateFile(parentId string, fileName string, filePath string, prorgress func(int64, int64)) (CloudFile, error) {
	fileId := parentId + "/" + fileName
	return d.UploadFile(fileId, filePath, fileName, prorgress)
}

func (d *MockCloudDriver) DeleteFile(fileId string) error {
	return d.fs.Remove(fileId)
}

func TestBasic(t *testing.T) {
	// Override global cloud service
	service = &MockCloudDriver{}
	service.InitDriver()
	dm := MakeGameDefManager("tests/test_useroverrides.json")
	defmap := dm.GetGameDefMap()
	defmap["Game"] = &GameDef{
		DisplayName: "Game",
		SteamId:     "0",
		LinuxPath: []*Datapath{
			{
				Path:    "tests/data",
				Parent:  "data",
				NetAuth: CloudOperationAll,
			},
		},
		WinPath: []*Datapath{
			{
				Path:    "tests/data",
				Parent:  "data",
				NetAuth: CloudOperationAll,
			},
		},
		DarwinPath: []*Datapath{
			{
				Path:    "tests/data",
				Parent:  "data",
				NetAuth: CloudOperationAll,
			},
		},
	}

	ops := &Options{
		Gamenames: []string{"Game"},
	}

	channels := &ChannelProvider{
		logs:     make(chan Message, 100),
		cancel:   make(chan Cancellation, 1),
		input:    make(chan SyncRequest, 10),
		output:   make(chan SyncResponse, 10),
		progress: make(chan ProgressEvent, 15),
	}

	go CliMain(ops, dm, channels, SyncOp)

	for {

		select {
		case result := <-channels.logs:
			if result.Finished {
				return
			}

			if result.Err != nil {
				fmt.Println(result.Err)
			} else {
				fmt.Println(result.Message)
			}
		case <-time.After(20 * time.Second):
			t.Error("timeout")
		}
	}
}
