package main

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/DavidDeSimone/memfs"
)

const STEAM_CLOUD_MEM_FS_DEFAULT = "SteamCloudLocalBackup"

type LocalMemFsFile struct {
	Name    string
	Id      string
	ModTime string
}

func (f *LocalMemFsFile) GetName() string {
	return f.Name
}

func (f *LocalMemFsFile) GetId() string {
	return f.Id
}

func (f *LocalMemFsFile) GetModTime() string {
	return f.ModTime
}

type LocalMemFsCloudDriver struct {
	root   string
	rootFS *memfs.FS
}

func (f *LocalMemFsCloudDriver) SetRoot(root string) error {
	f.root = root
	_, err := f.CreateMemDirIfNotExist(f.root)
	if err != nil {
		return err
	}

	return nil
}

func (f *LocalMemFsCloudDriver) GetRootFs() *memfs.FS {
	return f.rootFS
}

func (f *LocalMemFsCloudDriver) GetRoot() string {
	return f.root
}

func (f *LocalMemFsCloudDriver) InitDriver() error {
	f.rootFS = memfs.New()
	err := f.rootFS.MkdirAll("memfs", 0755)
	if err != nil {
		return err
	}
	f.root = "memfs/"
	return nil
}

func (f *LocalMemFsCloudDriver) ListFiles(parentId string) ([]CloudFile, error) {
	fmt.Println("Listing Files.... " + parentId)
	parentId = strings.Replace(parentId, "root", f.root, 1)
	scanId := strings.TrimSuffix(parentId, string(os.PathSeparator))
	result := []CloudFile{}
	fmt.Println("Walking ScanId... " + scanId)
	dirEntries, err := fs.ReadDir(f.rootFS, scanId)
	if err != nil {
		return nil, err
	}

	for _, file := range dirEntries {
		fmt.Println(file.Name())
		info, err := file.Info()
		if err != nil {
			return nil, err
		}

		id := parentId + file.Name()
		if info.IsDir() {
			id += string(os.PathSeparator)
		}
		fmt.Println("Id -> " + id)
		result = append(result, &LocalMemFsFile{
			Name:    file.Name(),
			Id:      id,
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (f *LocalMemFsCloudDriver) CreateMemDirIfNotExist(path string) (string, error) {
	modtime := ""

	path = strings.TrimSuffix(path, string(os.PathSeparator))
	stat, err := f.rootFS.Open(path)
	fmt.Println("Making Dir " + path)
	if err != nil {
		err := f.rootFS.MkdirAll(path, os.ModePerm)
		if err != nil {
			return "", err
		}
		modtime = time.Now().Format(time.RFC3339)
	} else {
		stat, err := stat.Stat()
		if err != nil {
			return "", err
		}
		modtime = stat.ModTime().Format(time.RFC3339)
	}

	fmt.Println("Mod Time -> " + modtime)
	return modtime, nil
}

func (f *LocalMemFsCloudDriver) CreateDir(name string, parentId string) (CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	path := parentId + name + string(os.PathSeparator)
	fmt.Println("Creating Dir " + path)

	modtime, err := f.CreateMemDirIfNotExist(path)
	if err != nil {
		return nil, err
	}

	fmt.Println("CreateDir Complete")
	return &LocalMemFsFile{
		Name:    name,
		Id:      path,
		ModTime: modtime,
	}, nil
}

func (f *LocalMemFsCloudDriver) DownloadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	fmt.Println("Operation Download File on -> " + fileId)

	content, err := fs.ReadFile(f.rootFS, fileId)
	if err != nil {
		return nil, err
	}

	err = f.rootFS.WriteFile(filePath, content, os.ModePerm)
	if err != nil {
		return nil, err
	}

	file, err := f.rootFS.Open(fileId) //os.Stat(fileId)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fmt.Println("Download complete.....")
	return &LocalMemFsFile{
		Name:    fileName,
		Id:      filePath,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalMemFsCloudDriver) UploadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	fmt.Println("Operation Upload File on -> " + fileId)

	bytes, err := fs.ReadFile(f.rootFS, filePath)
	if err != nil {
		return nil, err
	}

	err = f.rootFS.WriteFile(fileId, bytes, os.ModePerm)
	if err != nil {
		return nil, err
	}

	file, err := f.rootFS.Open(fileId) //os.Stat(fileId)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fmt.Println("Upload Complete....")
	return &LocalMemFsFile{
		Name:    fileName,
		Id:      filePath,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalMemFsCloudDriver) CreateFile(parentId string, fileName string, filePath string) (CloudFile, error) {
	fmt.Println("Operation Create File on -> " + parentId)
	parentId = strings.Replace(parentId, "root", f.root, 1)

	bytes, err := fs.ReadFile(f.rootFS, filePath)
	if err != nil {
		return nil, err
	}

	err = f.rootFS.WriteFile(parentId+fileName, bytes, os.ModePerm)
	if err != nil {
		return nil, err
	}

	file, err := f.rootFS.Open(parentId + fileName) //os.Stat(fileId)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fmt.Println("Creation Complete....")
	return &LocalMemFsFile{
		Name:    fileName,
		Id:      parentId + fileName,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalMemFsCloudDriver) DeleteFile(fileId string) error {
	fmt.Println("Deleteing File...")
	return f.rootFS.Remove(fileId)
}
