package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

func (f *LocalMemFsCloudDriver) InitDriver() error {
	f.rootFS = memfs.New()
	err := f.rootFS.MkdirAll("./", os.ModePerm)
	if err != nil {
		return err
	}

	f.root = "./"
	return nil
}

func (f *LocalMemFsCloudDriver) ListFiles(parentId string) ([]CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	files, err := fs.ReadDir(f.rootFS, parentId)
	if err != nil {
		return nil, err
	}

	result := []CloudFile{}
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			return nil, err
		}

		id := parentId + f.Name()
		if info.IsDir() {
			id += string(os.PathSeparator)
		}

		result = append(result, &LocalMemFsFile{
			Name:    f.Name(),
			Id:      id,
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	return result, nil
}

func (f *LocalMemFsCloudDriver) CreateMemDirIfNotExist(path string) (string, error) {
	modtime := ""

	if stat, err := f.rootFS.Open(path); errors.Is(err, os.ErrNotExist) {
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

	return modtime, nil
}

func (f *LocalMemFsCloudDriver) CreateDir(name string, parentId string) (CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	path := parentId + name + string(os.PathSeparator)

	modtime, err := f.CreateMemDirIfNotExist(path)
	if err != nil {
		return nil, err
	}

	return &LocalMemFsFile{
		Name:    name,
		Id:      path,
		ModTime: modtime,
	}, nil
}

func (f *LocalMemFsCloudDriver) copyMemFileContents(src, dst string) (err error) {
	content, err := fs.ReadFile(f.rootFS, src)
	if err != nil {
		return err
	}

	err = f.rootFS.WriteFile(dst, content, os.ModePerm)
	if err != nil {
		return err
	}
	return
}

func (f *LocalMemFsCloudDriver) DownloadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	err := f.copyMemFileContents(fileId, filePath)
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

	return &LocalMemFsFile{
		Name:    fileName,
		Id:      filePath,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalMemFsCloudDriver) UploadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	err := f.copyMemFileContents(filePath, fileId)
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

	return &LocalMemFsFile{
		Name:    fileName,
		Id:      filePath,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalMemFsCloudDriver) CreateFile(parentId string, fileName string, filePath string) (CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	err := f.copyMemFileContents(filePath, parentId+fileName)
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

	return &LocalMemFsFile{
		Name:    fileName,
		Id:      parentId + fileName,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalMemFsCloudDriver) GetMetaData(parentId string, fileName string) (*GameMetadata, error) {
	metaFile, err := f.rootFS.Open(parentId + fileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		} else {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	defer metaFile.Close()

	bytes, err := io.ReadAll(metaFile)
	if err != nil {
		return nil, err
	}

	metadata := &GameMetadata{}
	err = json.Unmarshal(bytes, metadata)
	if err != nil {
		return nil, err
	}

	metadata.fileId = parentId + fileName

	return metadata, nil
}

func (f *LocalMemFsCloudDriver) UpdateMetaData(parentId string, fileName string, filePath string, metaData *GameMetadata) error {

	bytes, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	return f.rootFS.WriteFile(parentId+fileName, bytes, os.ModePerm)
}

func (f *LocalMemFsCloudDriver) DeleteFile(fileId string) error {
	file, err := f.rootFS.Open(fileId)
	if err != nil {
		return err
	}
	defer file.Close()
	return f.rootFS.Remove(fileId)
}

func (f *LocalMemFsCloudDriver) IsFileInSync(fileName string, filePath string, fileId string, metadata *GameMetadata) (int, error) {
	localFile, err := f.rootFS.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer localFile.Close()
	localStat, err := localFile.Stat()
	if err != nil {
		return 0, err
	}
	fmt.Println(fileId)
	remoteFile, err := f.rootFS.Open(fileId)
	if err != nil {
		return 0, err
	}
	defer remoteFile.Close()

	remoteStat, err := remoteFile.Stat()
	if err != nil {
		return 0, err
	}

	localFileHash, err := getFileHash(filePath)
	if err != nil {
		return 0, err
	}
	remoteFileHash, err := getFileHash(fileId)
	if err != nil {
		return 0, err
	}

	if localStat.ModTime().Equal(remoteStat.ModTime()) || localFileHash == remoteFileHash {
		return InSync, nil
	} else if localStat.ModTime().After(remoteStat.ModTime()) {
		return LocalNewer, nil
	} else {
		return RemoteNewer, nil
	}

}
