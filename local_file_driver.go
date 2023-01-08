package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const STEAM_CLOUD_FS_DEFAULT = "SteamCloudLocalBackup"

type LocalFsCloudFile struct {
	Name    string
	Id      string
	ModTime string
}

func (f *LocalFsCloudFile) GetName() string {
	return f.Name
}

func (f *LocalFsCloudFile) GetId() string {
	return f.Id
}

func (f *LocalFsCloudFile) GetModTime() string {
	return f.ModTime
}

type LocalFsCloudDriver struct {
	root string
}

func (f *LocalFsCloudDriver) SetRoot(root string) error {
	f.root = root
	_, err := CreateDirIfNotExist(f.root)
	if err != nil {
		return err
	}

	return nil
}

func (f *LocalFsCloudDriver) InitDriver() error {
	separator := string(os.PathSeparator)
	root, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	f.root = root + separator + STEAM_CLOUD_FS_DEFAULT + separator
	_, err = CreateDirIfNotExist(f.root)
	if err != nil {
		return err
	}
	return nil
}

func (f *LocalFsCloudDriver) ListFiles(parentId string) ([]CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	dir, err := os.Open(parentId)
	if err != nil {
		return nil, err
	}

	files, err := dir.ReadDir(0)
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

		result = append(result, &LocalFsCloudFile{
			Name:    f.Name(),
			Id:      id,
			ModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	return result, nil
}

func CreateDirIfNotExist(path string) (string, error) {
	modtime := ""

	if stat, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(path, os.ModePerm)
		if err != nil {
			return "", err
		}
		stat, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		modtime = stat.ModTime().Format(time.RFC3339)
	} else {
		modtime = stat.ModTime().Format(time.RFC3339)
	}

	return modtime, nil
}

func (f *LocalFsCloudDriver) CreateDir(name string, parentId string) (CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	path := parentId + name + string(os.PathSeparator)

	modtime, err := CreateDirIfNotExist(path)
	if err != nil {
		return nil, err
	}

	return &LocalFsCloudFile{
		Name:    name,
		Id:      path,
		ModTime: modtime,
	}, nil
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func (f *LocalFsCloudDriver) DownloadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	err := copyFileContents(fileId, filePath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(fileId)
	if err != nil {
		return nil, err
	}

	return &LocalFsCloudFile{
		Name:    fileName,
		Id:      filePath,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalFsCloudDriver) UploadFile(fileId string, filePath string, fileName string) (CloudFile, error) {
	err := copyFileContents(filePath, fileId)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(fileId)
	if err != nil {
		return nil, err
	}

	return &LocalFsCloudFile{
		Name:    fileName,
		Id:      filePath,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalFsCloudDriver) CreateFile(parentId string, fileName string, filePath string) (CloudFile, error) {
	parentId = strings.Replace(parentId, "root", f.root, 1)
	err := copyFileContents(filePath, parentId+fileName)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(parentId + fileName)
	if err != nil {
		return nil, err
	}

	return &LocalFsCloudFile{
		Name:    fileName,
		Id:      parentId + fileName,
		ModTime: stat.ModTime().Format(time.RFC3339),
	}, nil
}

func (f *LocalFsCloudDriver) GetMetaData(parentId string, fileName string) (*GameMetadata, error) {
	metaFile, err := os.Open(parentId + fileName)
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

func (f *LocalFsCloudDriver) UpdateMetaData(parentId string, fileName string, filePath string, metaData *GameMetadata) error {

	bytes, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	return os.WriteFile(parentId+fileName, bytes, os.ModePerm)
}

func (f *LocalFsCloudDriver) DeleteFile(fileId string) error {
	info, err := os.Stat(fileId)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(fileId)
	} else {
		return os.Remove(fileId)
	}
}

func (f *LocalFsCloudDriver) IsFileInSync(fileName string, filePath string, fileId string, metadata *GameMetadata) (int, error) {
	localStat, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	fmt.Println(fileId)
	remoteStat, err := os.Stat(fileId)
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
