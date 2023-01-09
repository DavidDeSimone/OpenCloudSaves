package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	err := fs.WalkDir(f.rootFS, scanId, func(path string, file fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
			// return err
		}

		if path == scanId {
			return nil
		}

		fmt.Println("Path : " + path)
		fmt.Println(file.Name())
		info, err := file.Info()
		if err != nil {
			return err
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

		return nil
	})

	if err != nil {
		panic(err)
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

func (f *LocalMemFsCloudDriver) GetMetaData(parentId string, fileName string) (*GameMetadata, error) {
	fmt.Println("Getting Metadata -> " + parentId + fileName)
	metaFile, err := f.rootFS.Open(parentId + fileName)
	if err != nil {
		return nil, nil
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

	fmt.Println("Fetched metadata -> " + metadata.fileId)
	return metadata, nil
}

func (f *LocalMemFsCloudDriver) UpdateMetaData(parentId string, fileName string, filePath string, metaData *GameMetadata) error {
	fmt.Println("Updating Metadata...")
	bytes, err := json.Marshal(metaData)
	if err != nil {
		return err
	}

	fmt.Println("Meta Upload complete...")
	return f.rootFS.WriteFile(parentId+fileName, bytes, os.ModePerm)
}

func (f *LocalMemFsCloudDriver) DeleteFile(fileId string) error {
	fmt.Println("Deleteing File...")
	return f.rootFS.Remove(fileId)
}

func (f *LocalMemFsCloudDriver) IsFileInSync(fileName string, filePath string, fileId string, metadata *GameMetadata) (int, error) {
	fmt.Printf("Comparing Files.....")

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

	h := sha256.New()
	if _, err := io.Copy(h, remoteFile); err != nil {
		log.Fatal(err)
	}

	remoteFileHash := hex.EncodeToString(h.Sum(nil))

	if localStat.ModTime().Equal(remoteStat.ModTime()) || localFileHash == remoteFileHash {
		return InSync, nil
	} else if localStat.ModTime().After(remoteStat.ModTime()) {
		return LocalNewer, nil
	} else {
		return RemoteNewer, nil
	}

}
