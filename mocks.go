package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
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
